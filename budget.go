package main

import (
	"log"
	"strconv"
	"sync"
	"time"
)

// Adaptive move budget: starts conservative, increases as we observe low network latency.
//
// The Battlesnake API sends:
//   - game.timeout: total allowed response time (typically 500ms)
//   - you.latency: total observed response time for our PREVIOUS move (string, ms)
//
// Since latency = our_compute_time + network_round_trip, we track our own compute
// time per turn to extract network overhead: network = latency - our_compute.
// Then: budget = server_timeout - network_estimate - safety_margin.

const (
	minBudget     = 350 * time.Millisecond // never go below this
	maxBudget     = 420 * time.Millisecond // never exceed this
	safetyMargin  = 50 * time.Millisecond  // headroom for jitter, JSON encode/decode, GC
	defaultBudget = 350 * time.Millisecond // starting budget before enough observations
	emaAlpha      = 0.3                    // EMA smoothing (higher = more reactive)
)

// budgetTracker maintains per-game adaptive budget state.
type budgetTracker struct {
	mu              sync.Mutex
	networkEMA      float64 // exponential moving average of network overhead (ms)
	observations    int     // number of network samples
	lastComputeMs   float64 // compute time of our last move (ms), for next turn's extraction
	serverTimeout   time.Duration
}

var (
	budgetTrackers sync.Map // key: gameID → *budgetTracker
)

// initBudgetTracker creates a tracker for a new game.
func initBudgetTracker(gameID string, serverTimeoutMs int) {
	timeout := time.Duration(serverTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	budgetTrackers.Store(gameID, &budgetTracker{
		serverTimeout: timeout,
	})
}

// removeBudgetTracker cleans up when a game ends.
func removeBudgetTracker(gameID string) {
	budgetTrackers.Delete(gameID)
}

// computeBudget returns the adaptive compute budget for this turn.
// latencyStr is state.You.Latency from the API (total response time of previous move, ms).
func computeBudget(gameID string, latencyStr string) time.Duration {
	v, ok := budgetTrackers.Load(gameID)
	if !ok {
		return defaultBudget
	}
	bt := v.(*budgetTracker)

	// Extract network overhead from previous turn: network = API_latency - our_compute.
	if latencyStr != "" {
		if totalMs, err := strconv.ParseFloat(latencyStr, 64); err == nil && totalMs > 0 {
			bt.mu.Lock()
			if bt.lastComputeMs > 0 {
				networkMs := totalMs - bt.lastComputeMs
				if networkMs < 0 {
					networkMs = 0 // clock skew protection
				}
				if bt.observations == 0 {
					bt.networkEMA = networkMs
				} else {
					bt.networkEMA = emaAlpha*networkMs + (1-emaAlpha)*bt.networkEMA
				}
				bt.observations++
			}
			bt.mu.Unlock()
		}
	}

	bt.mu.Lock()
	ema := bt.networkEMA
	obs := bt.observations
	timeout := bt.serverTimeout
	bt.mu.Unlock()

	// Not enough data — use conservative default.
	if obs < 3 {
		return defaultBudget
	}

	// budget = server_timeout - estimated_network - safety_margin
	networkEstimate := time.Duration(ema * float64(time.Millisecond))
	budget := timeout - networkEstimate - safetyMargin

	if budget < minBudget {
		budget = minBudget
	}
	if budget > maxBudget {
		budget = maxBudget
	}

	return budget
}

// recordComputeTime stores our compute duration so the next turn can extract network overhead.
func recordComputeTime(gameID string, computeTime time.Duration) {
	v, ok := budgetTrackers.Load(gameID)
	if !ok {
		return
	}
	bt := v.(*budgetTracker)
	bt.mu.Lock()
	bt.lastComputeMs = float64(computeTime) / float64(time.Millisecond)
	bt.mu.Unlock()

	// Log budget adaptation periodically.
	if bt.observations > 0 && bt.observations%20 == 0 {
		log.Printf("BUDGET: network_ema=%.1fms obs=%d budget=%v",
			bt.networkEMA, bt.observations, computeBudget(gameID, ""))
	}
}
