package main

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ClientSimulator simulates concurrent clients
type ClientSimulator struct {
	gateway *Gateway
	clients int
	running bool
	wg      sync.WaitGroup
}

// NewClientSimulator creates a new client simulator
func NewClientSimulator(gw *Gateway) *ClientSimulator {
	return &ClientSimulator{
		gateway: gw,
		clients: 100,
	}
}

// Start begins the client simulation
func (cs *ClientSimulator) Start() {
	cs.running = true

	for i := 0; i < cs.clients; i++ {
		cs.wg.Add(1)
		go cs.simulateClient(i)
	}

	cs.wg.Wait()
}

// Stop stops the simulation
func (cs *ClientSimulator) Stop() {
	cs.running = false
}

// simulateClient simulates a single client making requests
func (cs *ClientSimulator) simulateClient(clientID int) {
	defer cs.wg.Done()

	client := &http.Client{Timeout: 10 * time.Second}

	for cs.running {
		// Make request
		reqBody := `{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}]}`
		resp, err := client.Post("http://localhost:8080/chat/completions", "application/json", bytes.NewBufferString(reqBody))

		if err != nil {
			fmt.Printf("Client %d: Error %v\n", clientID, err)
		} else {
			resp.Body.Close()
		}

		// Random delay between requests (simulate real client behavior)
		time.Sleep(time.Duration(100+clientID*10) * time.Millisecond)
	}
}