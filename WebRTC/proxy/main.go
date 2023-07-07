package main

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pion/webrtc/v3"
)

func main() {
	// Create a new WebRTC API instance
	api := webrtc.NewAPI()

	// Create the configuration for the WebRTC peer
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new WebRTC peer connection
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// Create a data channel
	dataChannel, err := peerConnection.CreateDataChannel("myDataChannel", nil)
	if err != nil {
		panic(err)
	}

	// Set up handlers for the data channel
	dataChannel.OnOpen(func() {
		fmt.Println("Data channel opened")
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Received message: %s\n", string(msg.Data))
	})

	// Wait for the encoded offer to be copied from the other endpoint and pasted into the console
	var encodedOffser string
	fmt.Println("Paste the encoded offer from the other endpoint:")
	fmt.Scanln(&encodedOffser)
	decodedOffer, err := base64.StdEncoding.DecodeString(encodedOffser)
	if err != nil {
		panic(err)
	}
	offser := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  string(decodedOffer),
	}
	peerConnection.SetRemoteDescription(offser)

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	peerConnection.SetLocalDescription(answer)
	encodedAnswer := base64.StdEncoding.EncodeToString([]byte(answer.SDP))
	fmt.Println("Copy the encoded answer below and paste it into the other endpoint's console:")
	fmt.Println(encodedAnswer)

	for dataChannel.ReadyState() != webrtc.DataChannelStateOpen {
		fmt.Println("Waiting for data channel to be ready...")
		time.Sleep(1 * time.Second)
	}
	// Now, the WebRTC connection is established and the data channel is ready to use

	// Send a message through the data channel
	err = dataChannel.SendText("Hello, WebRTC!")
	if err != nil {
		panic(err)
	}

	// Wait for a signal to exit
	<-make(chan struct{})
}
