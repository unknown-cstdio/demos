package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"github.com/pion/webrtc/v3"
)

type Message struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type Context struct {
	oldConnection *webrtc.PeerConnection
	newConnection *webrtc.PeerConnection
}

var context Context
var nextProxy string

func connect(newAddr string) {
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

	dataChannel, err := peerConnection.CreateDataChannel("data2", nil)
	if err != nil {
		panic(err)
	}

	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		//fmt.Printf("Peer Connection2 State has changed: %s\n", s.String())
	})

	peerConnection.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		//fmt.Printf("ICE Connection2 State has changed: %s\n", s.String())
	})

	peerConnection.OnICEGatheringStateChange(func(s webrtc.ICEGathererState) {
		//fmt.Printf("ICE Gathering2 State has changed: %s\n", s.String())
	})

	context.newConnection = peerConnection

	dataChannel.OnOpen(func() {
		//fmt.Printf("Data channel %s is open\n", dataChannel.Label())
	})

	dataChannel.OnClose(func() {
		//fmt.Printf("Data channel %s is closed\n", dataChannel.Label())
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
		m := Message{}
		err := json.Unmarshal(msg.Data, &m)
		if err != nil {
			panic(err)
		}
		if m.Type == "switchProxy" {
			if m.Payload != "switch" {
				nextProxy = m.Payload
				reply := Message{"switchProxy", "switch"}
				payload, err := json.Marshal(reply)
				if err != nil {
					panic(err)
				}
				connect(nextProxy)
				dataChannel.Send(payload)
			} else {
				context.oldConnection.Close()
				context.oldConnection = nil
				context.oldConnection = context.newConnection
			}
		}
	})

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	if err = peerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}

	<-gatherComplete

	fmt.Println("LocalDescription2 set")
	//fmt.Printf("LocalDescription set: %s\n", peerConnection.LocalDescription().SDP)

	payload, err := json.Marshal(peerConnection.LocalDescription())
	if err != nil {
		panic(err)
	}
	resp, err := http.Post(fmt.Sprintf("http://%s/connect", newAddr), "application/json", bytes.NewBuffer(payload))

	if err != nil {
		panic(err)
	}

	//receive answer
	answer := webrtc.SessionDescription{}
	err = json.NewDecoder(resp.Body).Decode(&answer)
	if err != nil {
		panic(err)
	}

	//fmt.Println(answer)

	error := peerConnection.SetRemoteDescription(answer)
	if error != nil {
		panic(error)
	}
	fmt.Println("Received answer2")
}

func main() {
	brokerAddr := flag.String("broker", "localhost:8888", "Address of the broker")
	flag.Parse()

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

	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		//fmt.Printf("Peer Connection State has changed: %s\n", s.String())
	})

	peerConnection.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		//fmt.Printf("ICE Connection State has changed: %s\n", s.String())
	})
	peerConnection.OnICEGatheringStateChange(func(s webrtc.ICEGathererState) {
		//fmt.Printf("ICE Gathering State has changed: %s\n", s.String())
	})

	context.oldConnection = peerConnection

	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	dataChannel.OnOpen(func() {
		//fmt.Printf("Data channel %s is open\n", dataChannel.Label())
	})

	dataChannel.OnClose(func() {
		//fmt.Printf("Data channel %s is closed\n", dataChannel.Label())
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
		m := Message{}
		err := json.Unmarshal(msg.Data, &m)
		if err != nil {
			panic(err)
		}
		if m.Type == "switchProxy" {
			if m.Payload != "switch" {
				nextProxy = m.Payload
				reply := Message{"switchProxy", "switch"}
				payload, err := json.Marshal(reply)
				if err != nil {
					panic(err)
				}
				connect(nextProxy)
				dataChannel.Send(payload)
			} else {
				context.oldConnection.Close()
				context.oldConnection = nil
				context.oldConnection = context.newConnection
			}
		}
	})

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(offer.SDP)

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	if err = peerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}

	<-gatherComplete

	fmt.Println("LocalDescription set")
	//fmt.Printf("LocalDescription set: %s\n", peerConnection.LocalDescription().SDP)

	payload, err := json.Marshal(peerConnection.LocalDescription())
	if err != nil {
		panic(err)
	}
	resp, err := http.Post(fmt.Sprintf("http://%s/client", *brokerAddr), "application/json", bytes.NewBuffer(payload))

	if err != nil {
		panic(err)
	}

	//receive answer
	answer := webrtc.SessionDescription{}
	err = json.NewDecoder(resp.Body).Decode(&answer)
	if err != nil {
		panic(err)
	}
	fmt.Println("Received answer")
	//fmt.Println(answer)

	peerConnection.SetRemoteDescription(answer)
	select {}

}
