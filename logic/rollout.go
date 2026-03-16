package logic

import "time"

const (
	rolloutBatchSize = 32 // rollouts between time checks
)

// Tunable MC parameters — settable via environment for A/B sweeps.
var (
	RolloutMaxTurns   = 100  // max turns per rollout
	MCWeight          = 15.0 // max MC weight at lateBlend=1.0
	MCSpreadThreshold = 0.10 // min survival spread to activate MC bias
	MCLateGate        = 0.1  // min lateBlend to activate MC bias
)

// RolloutStats tracks survival outcomes for one root direction.
type RolloutStats struct {
	Survived     int
	Total        int
	SurvivalRate float64
}

// MCRolloutResult holds per-direction survival statistics from MC rollouts.
type MCRolloutResult struct {
	Stats [4]RolloutStats
	Valid [4]bool // false if direction is wall collision
}

// MCRolloutTimed runs Monte Carlo rollouts within the given budget.
// For each valid root direction, it plays out random games and measures survival.
// Rollouts are round-robin across valid directions to ensure balanced sampling.
func MCRolloutTimed(g *GameSim, myIdx, oppIdx, maxTurns int, budget time.Duration) MCRolloutResult {
	var result MCRolloutResult
	deadline := time.Now().Add(budget)

	// Pre-compute valid root directions (wall check).
	myHead := g.Snakes[myIdx].Head()
	var validDirs [4]Direction
	nValid := 0
	for _, d := range AllDirections {
		next := myHead.Move(d)
		if next.X >= 0 && next.X < g.Width && next.Y >= 0 && next.Y < g.Height {
			result.Valid[d] = true
			validDirs[nValid] = d
			nValid++
		}
	}
	if nValid == 0 {
		return result
	}

	// Seed PRNG.
	rng := xorshift64(g.Hash() ^ uint64(g.Turn) | 1)

	// Round-robin rollouts across valid directions.
	dirIdx := 0
	for {
		for batch := 0; batch < rolloutBatchSize; batch++ {
			rootDir := validDirs[dirIdx]
			dirIdx++
			if dirIdx >= nValid {
				dirIdx = 0
			}

			survived := runOneRollout(g, myIdx, oppIdx, rootDir, maxTurns, &rng)
			result.Stats[rootDir].Total++
			if survived {
				result.Stats[rootDir].Survived++
			}
		}

		if time.Now().After(deadline) {
			break
		}
	}

	// Compute survival rates.
	for d := Direction(0); d < 4; d++ {
		if result.Stats[d].Total > 0 {
			result.Stats[d].SurvivalRate = float64(result.Stats[d].Survived) / float64(result.Stats[d].Total)
		}
	}

	return result
}

// runOneRollout plays a single random game from root position after applying rootDir.
// Returns true if our snake survived maxTurns.
func runOneRollout(g *GameSim, myIdx, oppIdx int, rootDir Direction, maxTurns int, rng *xorshift64) bool {
	// Apply root move.
	sim := g.CloneFromPool()
	oppRootDir := randomSafeMove(sim, oppIdx, rng)
	if oppIdx >= 0 {
		sim.Step(newMoveSet2(myIdx, rootDir, oppIdx, oppRootDir))
	} else {
		sim.Step(newMoveSet1(myIdx, rootDir))
	}

	if !sim.Snakes[myIdx].IsAlive() || sim.IsOver() {
		alive := sim.Snakes[myIdx].IsAlive()
		sim.Release()
		return alive
	}

	// Play out random moves.
	for turn := 1; turn < maxTurns; turn++ {
		myDir := randomSafeMove(sim, myIdx, rng)
		if oppIdx >= 0 {
			oppDir := randomSafeMove(sim, oppIdx, rng)
			sim.Step(newMoveSet2(myIdx, myDir, oppIdx, oppDir))
		} else {
			sim.Step(newMoveSet1(myIdx, myDir))
		}

		if !sim.Snakes[myIdx].IsAlive() {
			sim.Release()
			return false
		}
		if sim.IsOver() {
			sim.Release()
			return true // we're alive and game is over = we won
		}
	}

	sim.Release()
	return true // survived all turns
}

// randomSafeMove picks a random safe direction for the given snake.
// Falls back to wall-safe only if no isSafeDir moves exist, then any direction.
func randomSafeMove(g *GameSim, snakeIdx int, rng *xorshift64) Direction {
	if snakeIdx < 0 || !g.Snakes[snakeIdx].IsAlive() {
		return Down
	}

	s := &g.Snakes[snakeIdx]

	// Collect isSafeDir moves.
	var safe [4]Direction
	nSafe := 0
	for _, d := range AllDirections {
		if isSafeDir(g, s, d) {
			safe[nSafe] = d
			nSafe++
		}
	}
	if nSafe > 0 {
		return safe[rng.next()%uint64(nSafe)]
	}

	// Fallback: wall-safe only.
	head := s.Head()
	var wallSafe [4]Direction
	nWallSafe := 0
	for _, d := range AllDirections {
		next := head.Move(d)
		if next.X >= 0 && next.X < g.Width && next.Y >= 0 && next.Y < g.Height {
			wallSafe[nWallSafe] = d
			nWallSafe++
		}
	}
	if nWallSafe > 0 {
		return wallSafe[rng.next()%uint64(nWallSafe)]
	}

	return Down
}

// applyMCBias adjusts BRS result using MC rollout survival data.
// lateBlend is the game phase (0.0 = early, 1.0 = late game).
// MC weight scales with lateBlend and spread magnitude: negligible early,
// strong late when MC sees clear directional differences.
func applyMCBias(result BRSResult, mc MCRolloutResult, lateBlend float64) Direction {
	// No MC influence in early game.
	if lateBlend < MCLateGate {
		return result.BestDir
	}

	// Compute min/max survival across valid directions.
	minSurv := 2.0
	maxSurv := -1.0
	validCount := 0
	avgSurv := 0.0

	for d := Direction(0); d < 4; d++ {
		if !mc.Valid[d] || mc.Stats[d].Total == 0 {
			continue
		}
		rate := mc.Stats[d].SurvivalRate
		if rate < minSurv {
			minSurv = rate
		}
		if rate > maxSurv {
			maxSurv = rate
		}
		avgSurv += rate
		validCount++
	}

	if validCount == 0 {
		return result.BestDir
	}
	avgSurv /= float64(validCount)

	// Threshold 0.10: analysis showed MC has discriminative power at moderate
	// spreads, but too low (0.08) caused noisy overrides that hurt (33%).
	spread := maxSurv - minSurv
	if spread < MCSpreadThreshold {
		return result.BestDir
	}

	// Conservative phase-scaled weight. The v1 tuning (57%) used flat weight=20.
	// The v2 aggressive spread scaling (33%) proved MC overrides must be gentle.
	// Keep it simple: phase-scaled 15, no spread amplification.
	// At lateBlend=1.0: max adj ≈ ±15 * 0.5 = ±7.5 pts (BRS range ~[-50,+50]).
	mcWeight := MCWeight * clamp01((lateBlend-MCLateGate)/(1.0-MCLateGate))

	bestDir := result.BestDir
	bestScore := -1e18

	for d := Direction(0); d < 4; d++ {
		if !result.HasScore[d] {
			continue
		}
		adjusted := result.Scores[d]
		if mc.Valid[d] && mc.Stats[d].Total > 0 {
			adjusted += mcWeight * (mc.Stats[d].SurvivalRate - avgSurv)
		}
		if adjusted > bestScore {
			bestScore = adjusted
			bestDir = d
		}
	}

	return bestDir
}
