package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pion/webrtc/v3"
)

type SwitchProxyRequest struct {
	NewAddr string `json:"newAddr"`
}

type Client struct {
	dc *webrtc.DataChannel
}

type Message struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

var clients = make([]Client, 0)

func addHandler(w http.ResponseWriter, r *http.Request) {

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

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())
		client := Client{dc: d}
		clients = append(clients, client)

		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			m := Message{}
			if err := json.Unmarshal(msg.Data, &m); err != nil {
				panic(err)
			}
			fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), m.Payload)
			if m.Type == "switchProxy" {
				data := Message{"switchProxy", "dummy data"}
				payload, err := json.Marshal(data)
				if err != nil {
					panic(err)
				}
				resp, err := http.Post("http://localhost:9998/data", "application/json", bytes.NewBuffer(payload))
				if err != nil {
					panic(err)
				}
				fmt.Println(resp)
				respMsg := Message{"switchProxy", "switch"}
				respPayload, err := json.Marshal(respMsg)
				if err != nil {
					panic(err)
				}
				d.Send(respPayload)
			}

		})

		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open.\n", d.Label(), d.ID())
		})
	})
	done := webrtc.GatheringCompletePromise(peerConnection)
	offer := webrtc.SessionDescription{}
	if sdpErr := json.NewDecoder(r.Body).Decode(&offer); sdpErr != nil {
		panic(sdpErr)
	}
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	fmt.Printf("Received offer: %s\n", peerConnection.RemoteDescription().SDP)

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	if err := peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	<-done
	fmt.Printf("Generated answer: %s\n", peerConnection.LocalDescription().SDP)

	payload, err := json.Marshal(peerConnection.LocalDescription())
	if err != nil {
		panic(err)
	}

	//Write answer back to response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)

}

func handleTransfer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Transfering client to a new address")
	var req SwitchProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		panic(err)
	}
	fmt.Println(req.NewAddr)
	for _, client := range clients {
		message := Message{"switchProxy", req.NewAddr}
		payload, err := json.Marshal(message)
		if err != nil {
			panic(err)
		}
		client.dc.Send(payload)
	}
}

func handleData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received data")
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
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

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())
		client := Client{dc: d}
		clients = append(clients, client)

		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			m := Message{}
			if err := json.Unmarshal(msg.Data, &m); err != nil {
				panic(err)
			}
			fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), m.Payload)
			if m.Type == "switchProxy" {
				data := Message{"switchProxy", "dummy data"}
				payload, err := json.Marshal(data)
				if err != nil {
					panic(err)
				}
				resp, err := http.Post("http://localhost:9998/data", "application/json", bytes.NewBuffer(payload))
				if err != nil {
					panic(err)
				}
				fmt.Println(resp)
				respMsg := Message{"switchProxy", "switch"}
				respPayload, err := json.Marshal(respMsg)
				if err != nil {
					panic(err)
				}
				d.Send(respPayload)
			}

		})

		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open.\n", d.Label(), d.ID())
		})
	})
	done := webrtc.GatheringCompletePromise(peerConnection)
	offer := webrtc.SessionDescription{}
	if sdpErr := json.NewDecoder(r.Body).Decode(&offer); sdpErr != nil {
		panic(sdpErr)
	}
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	fmt.Printf("Received offer: %s\n", peerConnection.RemoteDescription().SDP)

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	if err := peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	<-done
	fmt.Printf("Generated answer: %s\n", peerConnection.LocalDescription().SDP)

	payload, err := json.Marshal(peerConnection.LocalDescription())
	if err != nil {
		panic(err)
	}

	//Write answer back to response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)

}

func main() {
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/transfer", handleTransfer)
	http.HandleFunc("/data", handleData)
	http.HandleFunc("/connect", handleConnect)
	http.ListenAndServe(":9999", nil)
}
