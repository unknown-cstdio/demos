package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pion/webrtc/v3"
)

const (
	proxy1addr = "localhost:9998"
	proxy2addr = "localhost:9999"
)

type SwitchProxyRequest struct {
	NewAddr string `json:"newAddr"`
}

var currentProxy = proxy1addr

func clientHandler(w http.ResponseWriter, r *http.Request) {
	sdp := webrtc.SessionDescription{}
	if sdpErr := json.NewDecoder(r.Body).Decode(&sdp); sdpErr != nil {
		panic(sdpErr)
	}
	fmt.Println(sdp)
	payload, err := json.Marshal(sdp)
	if err != nil {
		panic(err)
	}

	// send sdp to proxy1
	resp, err := http.Post("http://"+proxy1addr+"/add", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		panic(err)
	}
	fmt.Println(resp)

	answer := webrtc.SessionDescription{}
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		panic(err)
	}
	fmt.Println(answer)
	payload, err = json.Marshal(answer)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)

}

func switchProxy() {
	newProxy := ""
	if currentProxy == proxy1addr {
		newProxy = proxy2addr
	} else {
		newProxy = proxy1addr
	}
	m := SwitchProxyRequest{newProxy}
	payload, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	resp, err := http.Post("http://"+currentProxy+"/transfer", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		panic(err)
	}
	fmt.Println(resp)
	currentProxy = newProxy
}

func main() {
	http.HandleFunc("/client", clientHandler)
	go func() { http.ListenAndServe(":8888", nil) }()

	ticker := time.NewTicker(10 * time.Second)
	for t := range ticker.C {
		fmt.Println("Tick at", t)
		switchProxy()
	}
}
