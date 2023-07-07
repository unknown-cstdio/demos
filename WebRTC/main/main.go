package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/pion/webrtc/v2"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

// EndpointListener ...
type EndpointListener struct {
	OnDescription chan string
	OnBye         chan bool
}

// Endpoint ...
type Endpoint struct {
	name        string
	PC          *webrtc.PeerConnection
	Listener    *EndpointListener
	IsInitiator bool

	dataChs map[string]*webrtc.DataChannel
}

var currentProxyNum = 1
var currentProxy *Endpoint
var blocked = false

// NewEndpoint ...
func NewEndpoint(name string) *Endpoint {
	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{},
	}

	// Create a new PeerConnection
	pc, err := webrtc.NewPeerConnection(config)
	check(err)

	return &Endpoint{
		name: name,
		PC:   pc,
		Listener: &EndpointListener{
			OnDescription: make(chan string),
			OnBye:         make(chan bool),
		},
		IsInitiator: false,
		dataChs:     map[string]*webrtc.DataChannel{},
	}
}

// Start assigns this epoint as an initiator and triggers sendding an offer.
func (ep *Endpoint) Start() {
	var msgID = 0
	ep.IsInitiator = true

	ordered := false
	maxRetransmits := uint16(0)

	options := &webrtc.DataChannelInit{
		Ordered:        &ordered,
		MaxRetransmits: &maxRetransmits,
	}

	// Create a datachannel with label 'data'
	dc, err := ep.PC.CreateDataChannel("data", options)
	check(err)

	// Register channel opening handling
	dc.OnOpen(func() {
		log.Printf("OnOpen: %s-%d. Random messages will now be sent to any connected DataChannels every second\n", dc.Label(), dc.ID())

		for range time.NewTicker(1000 * time.Millisecond).C {
			msgRaw := make([]byte, 15)
			rand.Read(msgRaw)
			msg := base64.StdEncoding.EncodeToString(msgRaw)
			log.Printf("['%s'] Sending (%d) msg: %s \n", ep.name, msgID, string(msg))
			msgID++

			if dc.ReadyState() != webrtc.DataChannelStateOpen {
				log.Printf("['%s'] DataChannel '%s' is not open. Skipping message '%s'\n", ep.name, dc.Label(), string(msg))
				break
			}
			err := dc.Send([]byte(msg))
			check(err)
		}
	})

	// Register the OnMessage to handle incoming messages
	dc.OnMessage(func(dcMsg webrtc.DataChannelMessage) {
		log.Printf("['%s'] Message ([]byte) from DataChannel '%s' with length %d: '%s'\n", ep.name, dc.Label(), len(dcMsg.Data), string(dcMsg.Data))
		if string(dcMsg.Data) == "switch" {
			reply := "received switch"
			log.Printf("['%s'] Sending (%d) msg: %s \n", ep.name, msgID, string(reply))
			msgID++
			err := dc.Send([]byte(reply))
			check(err)
			SwitchProxy()
		}
	})

	dc.OnClose(func() {
		log.Printf("['%s'] OnClose: %s-%d. a receiver has closed a data channel\n", ep.name, dc.Label(), dc.ID())
		ep.Listener.OnBye <- true
	})

	ep.dataChs[dc.Label()] = dc

	// Now, create an offer
	offer, err := ep.PC.CreateOffer(nil)
	check(err)

	ep.PC.SetLocalDescription(offer)

	desc, err := json.Marshal(offer)
	check(err)

	go func() {
		ep.Listener.OnDescription <- string(desc)
	}()
}

// OnRemoteDescription ...
func (ep *Endpoint) OnRemoteDescription(sdp string) {
	var desc webrtc.SessionDescription
	bytes := []byte(sdp)
	err := json.Unmarshal(bytes, &desc)
	check(err)

	// Apply the desc as the remote description
	err = ep.PC.SetRemoteDescription(desc)
	check(err)

	if ep.IsInitiator {
		return
	}

	// Set callback for new data channels
	ep.PC.OnDataChannel(func(dc *webrtc.DataChannel) {
		var msgID = 0
		// Register channel opening handling
		dc.OnOpen(func() {
			log.Printf("['%s'] OnOpen: %s-%d. a receiver has opened a data channel\n", ep.name, dc.Label(), dc.ID())

			go func() {
				time.Sleep(10 * time.Second)
				switchMsg := "switch"
				log.Printf("['%s'] Sending (%d) msg: %s \n", ep.name, msgID, string(switchMsg))
				msgID++
				err := dc.Send([]byte(switchMsg))
				check(err)
			}()

			for range time.NewTicker(1000 * time.Millisecond).C {
				msgRaw := make([]byte, 15)
				rand.Read(msgRaw)
				msg := base64.StdEncoding.EncodeToString(msgRaw)
				log.Printf("['%s'] Sending (%d) msg: %s \n", ep.name, msgID, string(msg))
				msgID++

				if blocked {
					break
				}

				if dc.ReadyState() != webrtc.DataChannelStateOpen {
					log.Printf("['%s'] DataChannel '%s' is not open. Skipping message '%s'\n", ep.name, dc.Label(), string(msg))
					break
				}
				err := dc.Send([]byte(msg))
				check(err)
			}
		})

		// Register the OnMessage to handle incoming messages
		dc.OnMessage(func(dcMsg webrtc.DataChannelMessage) {
			log.Printf("['%s'] Message ([]byte) from DataChannel '%s' with length %d: '%s'\n", ep.name, dc.Label(), len(dcMsg.Data), string(dcMsg.Data))
			if string(dcMsg.Data) == "received switch" {
				//transfer data
				blocked = true
				dc.Close()
				time.Sleep(1 * time.Second) //simulate data transfer
				blocked = false
			}
		})

		dc.OnClose(func() {
			log.Printf("['%s'] OnClose: %s-%d. a receiver has closed a data channel\n", ep.name, dc.Label(), dc.ID())
			ep.Listener.OnBye <- true
		})

		ep.dataChs[dc.Label()] = dc
	})

	answer, err := ep.PC.CreateAnswer(nil)
	check(err)

	ep.PC.SetLocalDescription(answer)

	desc2, err := json.Marshal(answer)
	check(err)

	go func() {
		ep.Listener.OnDescription <- string(desc2)
	}()
}

func SwitchProxy() {
	if currentProxyNum == 1 {
		currentProxyNum = 2
	} else {
		currentProxyNum = 1
	}

	log.Printf("Switching proxy to proxy%d\n", currentProxyNum)
}

func main() {

	for {
		ep1 := NewEndpoint("client")
		proxy1 := NewEndpoint("proxy1")
		proxy2 := NewEndpoint("proxy2")
		if currentProxyNum == 1 {
			currentProxy = proxy1
		} else {
			currentProxy = proxy2
		}
		fmt.Println("Current proxy: ", currentProxy.name)

		ep1.Start()

		loop := true
		for loop {
			// Block forever
			select {
			case sig := <-ep1.Listener.OnDescription:
				for blocked {
				}
				log.Printf("client => proxy:\n%s\n", sig)
				go currentProxy.OnRemoteDescription(sig)
			case <-ep1.Listener.OnBye:
				loop = false
				ep1.PC.Close()
				currentProxy.PC.Close()
			case sig := <-currentProxy.Listener.OnDescription:
				log.Printf("proxy => client:\n%s\n", sig)
				go ep1.OnRemoteDescription(sig)
			case <-ep1.Listener.OnBye:
				loop = false
				ep1.PC.Close()
				currentProxy.PC.Close()
			}
		}
	}

}
