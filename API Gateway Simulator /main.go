package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Simulator is the main struct for the high-concurrency simulator
type Simulator struct {
	gateway *Gateway
	clients *ClientSimulator
}

// NewSimulator creates a new simulator instance
func NewSimulator() *Simulator {
	gw := NewGateway()
	clients := NewClientSimulator(gw)

	return &Simulator{
		gateway: gw,
		clients: clients,
	}
}

// Start starts the simulator
func (s *Simulator) Start() {
	// Start metrics server
	go s.startMetricsServer()

	// Start client simulation
	go s.clients.Start()

	// Start gateway server
	s.startGatewayServer()
}

// startMetricsServer starts the Prometheus metrics endpoint
func (s *Simulator) startMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Metrics server starting on :9090")
	log.Fatal(http.ListenAndServe(":9090", nil))
}

// startGatewayServer starts the gateway API server
func (s *Simulator) startGatewayServer() {
	r := gin.Default()

	// Gateway endpoints
	r.POST("/chat/completions", s.gateway.HandleRequest)

	log.Println("Gateway server starting on :8080")
	r.Run(":8080")
}

func main() {
	sim := NewSimulator()
	sim.Start()
}