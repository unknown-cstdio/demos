package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
)

var currentProxyNum = 1
var stop = false
var pcStop = make(chan bool)
var currentPeerConnection *webrtc.PeerConnection

func signalCandidate(addr string, c *webrtc.ICECandidate) error {
	payload := []byte(c.ToJSON().Candidate)
	resp, err := http.Post(fmt.Sprintf("http://%s/candidate", addr), "application/json; charset=utf-8", bytes.NewReader(payload)) //nolint:noctx
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

func switchProxy() {
	if currentProxyNum == 1 {
		currentProxyNum = 2
	} else {
		currentProxyNum = 1
	}
}

func main() { //nolint:gocognit
	offerAddr := flag.String("offer-address", ":8888", "Address that the Offer HTTP server is hosted on.")
	answerAddr1 := flag.String("answer-address", ":9999", "Address that the Answer HTTP server is hosted on.")
	answerAddr2 := flag.String("answer-address2", ":9998", "Address that the Answer HTTP server is hosted on.")
	var answerAddr *string

	var candidatesMux sync.Mutex
	var pendingCandidates []*webrtc.ICECandidate

	http.HandleFunc("/candidate", func(w http.ResponseWriter, r *http.Request) {
		candidate, candidateErr := io.ReadAll(r.Body)
		if candidateErr != nil {
			panic(candidateErr)
		}
		if candidateErr := currentPeerConnection.AddICECandidate(webrtc.ICECandidateInit{Candidate: string(candidate)}); candidateErr != nil {
			panic(candidateErr)
		}
	})

	http.HandleFunc("/sdp", func(w http.ResponseWriter, r *http.Request) {
		sdp := webrtc.SessionDescription{}
		if sdpErr := json.NewDecoder(r.Body).Decode(&sdp); sdpErr != nil {
			panic(sdpErr)
		}

		if sdpErr := currentPeerConnection.SetRemoteDescription(sdp); sdpErr != nil {
			panic(sdpErr)
		}

		candidatesMux.Lock()
		defer candidatesMux.Unlock()

		for _, c := range pendingCandidates {
			if onICECandidateErr := signalCandidate(*answerAddr, c); onICECandidateErr != nil {
				panic(onICECandidateErr)
			}
		}
	})

	go func() { panic(http.ListenAndServe(*offerAddr, nil)) }()

	for {
		if currentProxyNum == 1 {
			answerAddr = answerAddr1
		} else {
			answerAddr = answerAddr2
		}
		flag.Parse()
		pendingCandidates := make([]*webrtc.ICECandidate, 0)

		config := webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
			},
		}

		peerConnection, err := webrtc.NewPeerConnection(config)
		if err != nil {
			panic(err)
		}

		currentPeerConnection = peerConnection

		peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
			if c == nil {
				return
			}

			candidatesMux.Lock()
			defer candidatesMux.Unlock()

			desc := peerConnection.RemoteDescription()
			if desc == nil {
				pendingCandidates = append(pendingCandidates, c)
			} else if onICECandidateErr := signalCandidate(*answerAddr, c); onICECandidateErr != nil {
				panic(onICECandidateErr)
			}
		})

		dataChannel, err := peerConnection.CreateDataChannel("data", nil)
		if err != nil {
			panic(err)
		}

		peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
			fmt.Printf("Peer Connection State has changed: %s\n", s.String())
			if s.String() == "closed" || s.String() == "disconnected" || s.String() == "failed" {
				pcStop <- true
			}

			if s == webrtc.PeerConnectionStateFailed {
				// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
				// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
				// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
				fmt.Println("Peer Connection has gone to failed exiting")
				os.Exit(0)
			}
		})

		dataChannel.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", dataChannel.Label(), dataChannel.ID())
			stop = false
			i := 0
			for range time.NewTicker(1 * time.Second).C {
				if stop {
					break
				}
				message := "message from client" + strconv.Itoa(i)
				i++
				fmt.Printf("client: Sending '%s'\n", message)

				// Send the message as text
				//fmt.Printf("data channel %d state: %s\n", dataChannel.ID(), dataChannel.ReadyState().String()) here there are closed data channels, find out why
				if dataChannel.ReadyState() != webrtc.DataChannelStateOpen {
					break
				}
				sendTextErr := dataChannel.SendText(message)
				if sendTextErr != nil {
					panic(sendTextErr)
				}
			}
		})

		dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("client: Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
			if string(msg.Data) == "switch" {
				reply := "received switch"
				fmt.Printf("client : Sending '%s'\n", reply)
				if sendTextErr := dataChannel.SendText(reply); sendTextErr != nil {
					panic(sendTextErr)
				}
				switchProxy()
				stop = true
			}
			if string(msg.Data) == "stop" {
				fmt.Printf("client : Received stop\n")
				dataChannel.Close()
				peerConnection.Close()
				//pcStop <- true
			}
		})
		dataChannel.OnClose(func() {
			fmt.Printf("Data channel '%s'-'%d' closed\n", dataChannel.Label(), dataChannel.ID())
			stop = true
		})

		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			panic(err)
		}

		if err = peerConnection.SetLocalDescription(offer); err != nil {
			panic(err)
		}

		payload, err := json.Marshal(offer)
		if err != nil {
			panic(err)
		}
		resp, err := http.Post(fmt.Sprintf("http://%s/sdp", *answerAddr), "application/json; charset=utf-8", bytes.NewReader(payload)) // nolint:noctx
		if err != nil {
			panic(err)
		} else if err := resp.Body.Close(); err != nil {
			panic(err)
		}

		<-pcStop
		fmt.Print("client: PeerConnection Closed\n")
	}
}
