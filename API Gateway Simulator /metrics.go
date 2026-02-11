package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "simulator_requests_total",
			Help: "Total number of requests",
		},
		[]string{"provider", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "simulator_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider"},
	)

	rateLimitHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "simulator_rate_limit_hits_total",
			Help: "Total rate limit hits",
		},
	)

	providerErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "simulator_provider_errors_total",
			Help: "Total provider errors",
		},
		[]string{"provider", "error_type"},
	)

	activeGoroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "simulator_active_goroutines",
			Help: "Number of active goroutines",
		},
	)
)