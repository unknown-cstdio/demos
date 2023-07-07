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

	// Create an offer to start the WebRTC negotiation process
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// Set the local description of the peer connection
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		panic(err)
	}

	// Encode the offer SDP as base64
	encodedOffer := base64.StdEncoding.EncodeToString([]byte(peerConnection.LocalDescription().SDP))

	// Print the encoded offer and copy it to the other endpoint
	fmt.Println("Copy the encoded offer below and paste it into the other endpoint's console:")
	fmt.Println(encodedOffer)

	// Wait for the encoded answer to be copied from the other endpoint and pasted into the console
	var encodedAnswer string
	fmt.Println("Paste the encoded answer from the other endpoint:")
	fmt.Scanln(&encodedAnswer)

	// Decode the answer from base64
	decodedAnswer, err := base64.StdEncoding.DecodeString(encodedAnswer)
	if err != nil {
		panic(err)
	}

	// Parse the answer SDP
	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  string(decodedAnswer),
	}

	// Set the remote description of the peer connection
	err = peerConnection.SetRemoteDescription(answer)
	if err != nil {
		panic(err)
	}

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
