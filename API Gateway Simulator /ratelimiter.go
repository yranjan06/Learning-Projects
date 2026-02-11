package main

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	rate       float64 // tokens per second
	capacity   float64
	tokens     int64   // atomic
	lastUpdate int64   // atomic unix nano
	mu         sync.Mutex // for comparison
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps float64) *RateLimiter {
	capacity := math.Max(10, rps*0.1) // Burst capacity
	return &RateLimiter{
		rate:       rps,
		capacity:   capacity,
		tokens:     int64(capacity * 1e9), // store as nano for atomic
		lastUpdate: time.Now().UnixNano(),
	}
}

// Allow checks if a request is allowed (atomic version)
func (rl *RateLimiter) Allow() bool {
	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&rl.lastUpdate)

	// Calculate elapsed time
	elapsed := float64(now-last) / 1e9

	// Refill tokens
	refill := elapsed * rl.rate
	currentTokens := float64(atomic.LoadInt64(&rl.tokens)) / 1e9
	newTokens := math.Min(rl.capacity, currentTokens+refill)

	// Try to consume a token
	if newTokens >= 1.0 {
		newTokens -= 1.0
		atomic.StoreInt64(&rl.tokens, int64(newTokens*1e9))
		atomic.StoreInt64(&rl.lastUpdate, now)
		return true
	}

	atomic.StoreInt64(&rl.lastUpdate, now)
	return false
}

// AllowMutex is the mutex-based version for comparison
func (rl *RateLimiter) AllowMutex() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(time.Unix(0, atomic.LoadInt64(&rl.lastUpdate)))

	refill := elapsed.Seconds() * rl.rate
	currentTokens := float64(atomic.LoadInt64(&rl.tokens)) / 1e9
	newTokens := math.Min(rl.capacity, currentTokens+refill)

	if newTokens >= 1.0 {
		newTokens -= 1.0
		atomic.StoreInt64(&rl.tokens, int64(newTokens*1e9))
		atomic.StoreInt64(&rl.lastUpdate, now.UnixNano())
		return true
	}

	atomic.StoreInt64(&rl.lastUpdate, now.UnixNano())
	return false
}