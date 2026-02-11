package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// Provider represents a mock upstream provider
type Provider struct {
	ID       string
	Weight   int64 // Use atomic for thread-safety
	Latency  time.Duration
	ErrorRate float64 // Probability of 5xx/429
}

// ProviderHealth tracks provider state
type ProviderHealth struct {
	mu             sync.RWMutex
	ErrorCount     int
	LastFailure    time.Time
	CooldownUntil  time.Time
	DisabledUntil  time.Time
}

// Gateway handles routing, rate limiting, and failover
type Gateway struct {
	providers      []*Provider
	health         map[string]*ProviderHealth
	limiter        *RateLimiter
	totalWeight    int64
	mu             sync.RWMutex
}

// NewGateway creates a new gateway with 3 providers
func NewGateway() *Gateway {
	providers := []*Provider{
		{ID: "provider1", Weight: 70, Latency: 100 * time.Millisecond, ErrorRate: 0.05},
		{ID: "provider2", Weight: 20, Latency: 500 * time.Millisecond, ErrorRate: 0.10},
		{ID: "provider3", Weight: 10, Latency: 2 * time.Second, ErrorRate: 0.20},
	}

	health := make(map[string]*ProviderHealth)
	totalWeight := int64(0)
	for _, p := range providers {
		health[p.ID] = &ProviderHealth{}
		totalWeight += p.Weight
	}

	return &Gateway{
		providers:   providers,
		health:      health,
		limiter:     NewRateLimiter(1000), // 1000 RPS global limit
		totalWeight: totalWeight,
	}
}

// SelectProvider selects a provider using weighted random selection
func (g *Gateway) SelectProvider() *Provider {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Filter available providers (not in cooldown)
	available := make([]*Provider, 0)
	for _, p := range g.providers {
		if !g.isInCooldown(p.ID) {
			available = append(available, p)
		}
	}

	if len(available) == 0 {
		// All in cooldown, pick the one expiring soonest
		return g.selectSoonestExpiring()
	}

	// Weighted random selection
	r := rand.Int63n(g.totalWeight)
	currentSum := int64(0)
	for _, p := range available {
		currentSum += atomic.LoadInt64(&p.Weight)
		if r < currentSum {
			return p
		}
	}

	return available[0]
}

// isInCooldown checks if a provider is in cooldown
func (g *Gateway) isInCooldown(providerID string) bool {
	h := g.health[providerID]
	h.mu.RLock()
	defer h.mu.RUnlock()
	now := time.Now()
	return now.Before(h.CooldownUntil) || now.Before(h.DisabledUntil)
}

// selectSoonestExpiring selects the provider with soonest cooldown expiry
func (g *Gateway) selectSoonestExpiring() *Provider {
	var selected *Provider
	soonest := time.Now().Add(24 * time.Hour) // Far future

	for _, p := range g.providers {
		h := g.health[p.ID]
		h.mu.RLock()
		expiry := h.CooldownUntil
		if h.DisabledUntil.After(h.CooldownUntil) {
			expiry = h.DisabledUntil
		}
		h.mu.RUnlock()

		if expiry.Before(soonest) {
			soonest = expiry
			selected = p
		}
	}

	return selected
}

// MarkFailure marks a provider as failed and applies cooldown
func (g *Gateway) MarkFailure(providerID string, errorType string) {
	h := g.health[providerID]
	h.mu.Lock()
	defer h.mu.Unlock()

	h.ErrorCount++
	h.LastFailure = time.Now()

	cooldown := g.calculateCooldown(h.ErrorCount, errorType)
	if errorType == "billing" {
		h.DisabledUntil = time.Now().Add(cooldown)
	} else {
		h.CooldownUntil = time.Now().Add(cooldown)
	}
}

// calculateCooldown computes exponential backoff with jitter
func (g *Gateway) calculateCooldown(errorCount int, errorType string) time.Duration {
	base := 1 * time.Minute
	if errorCount <= 0 {
		return 0
	}

	exponent := math.Min(float64(errorCount-1), 3)
	multiplier := math.Pow(5, exponent)
	cooldown := time.Duration(float64(base) * multiplier)

	// Add jitter (±30%)
	jitter := time.Duration(rand.Float64()*0.6 - 0.3) * cooldown
	cooldown += jitter

	if cooldown > 1*time.Hour {
		cooldown = 1 * time.Hour
	}

	return cooldown
}

// HandleRequest handles incoming requests
func (g *Gateway) HandleRequest(c *gin.Context) {
	// Rate limiting
	if !g.limiter.Allow() {
		rateLimitHits.Inc()
		c.JSON(429, gin.H{"error": "Rate limit exceeded"})
		return
	}

	// Select provider
	provider := g.SelectProvider()

	// Simulate request to provider
	start := time.Now()
	err := g.simulateProviderCall(provider)
	duration := time.Since(start)

	if err != nil {
		g.MarkFailure(provider.ID, "rate_limit") // Assume 429 for simplicity
		providerErrors.WithLabelValues(provider.ID, "rate_limit").Inc()
		requestsTotal.WithLabelValues(provider.ID, "error").Inc()
		c.JSON(502, gin.H{"error": "Upstream error"})
		return
	}

	requestDuration.WithLabelValues(provider.ID).Observe(duration.Seconds())
	requestsTotal.WithLabelValues(provider.ID, "success").Inc()

	c.JSON(200, gin.H{
		"provider": provider.ID,
		"latency":  duration.Milliseconds(),
	})
}

// simulateProviderCall simulates calling a provider
func (g *Gateway) simulateProviderCall(p *Provider) error {
	// Simulate latency
	time.Sleep(p.Latency)

	// Simulate errors
	if rand.Float64() < p.ErrorRate {
		return fmt.Errorf("simulated error")
	}

	return nil
}