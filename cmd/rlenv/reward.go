package main

import (
	"github.com/bodist/haruko/logic"
)

// computeReward calculates the reward for the current step.
// prevG is the game state before the step, g is the state after.
func computeReward(g *logic.GameSim, prevG *logic.GameSim, myIdx, oppIdx int, hasOpponent bool, oppType opponentType) float32 {
	if !hasOpponent {
		return computeRewardSolo(g, prevG, myIdx)
	}
	if oppType == oppSelf {
		return computeRewardSelfPlay(g, prevG, myIdx, oppIdx)
	}
	return computeRewardVsOpponent(g, prevG, myIdx, oppIdx)
}

// computeRewardSelfPlay computes reward for self-play mode.
// Sparse rewards: death -1, kill +1, alive +0.01.
func computeRewardSelfPlay(g *logic.GameSim, prevG *logic.GameSim, myIdx, oppIdx int) float32 {
	me := &g.Snakes[myIdx]
	if !me.IsAlive() {
		return -1.0
	}
	var reward float32 = 0.01
	if oppIdx >= 0 && oppIdx < len(g.Snakes) && !g.Snakes[oppIdx].IsAlive() {
		if prevG != nil && prevG.Snakes[oppIdx].IsAlive() {
			reward += 1.0
		}
	}
	return reward
}

// computeRewardSolo computes reward for solo (no opponent) mode.
// - Death: -1.0
// - Per turn alive: +0.01
// - Health < 30: -0.005 * (30 - health) (gradient toward food)
func computeRewardSolo(g *logic.GameSim, prevG *logic.GameSim, myIdx int) float32 {
	me := &g.Snakes[myIdx]
	if !me.IsAlive() {
		return -1.0
	}
	reward := float32(0.01)
	if me.Health < 30 {
		reward -= 0.005 * float32(30-me.Health)
	}
	return reward
}

// computeRewardVsOpponent computes reward for competitive (vs opponent) mode.
// - Our death: -10.0
// - Opponent dies: +10.0
// - Per turn alive: +0.01
// - Eat food: +0.1
// - Territory advantage: +0.01 * (my_territory - opp_territory)
// - Opponent confined (0-1 safe moves): +1.0
func computeRewardVsOpponent(g *logic.GameSim, prevG *logic.GameSim, myIdx, oppIdx int) float32 {
	me := &g.Snakes[myIdx]

	// Our death.
	if !me.IsAlive() {
		return -10.0
	}

	var reward float32

	// Opponent dies.
	if oppIdx >= 0 && oppIdx < len(g.Snakes) && !g.Snakes[oppIdx].IsAlive() {
		// Check if opponent was alive before this step.
		if prevG != nil && prevG.Snakes[oppIdx].IsAlive() {
			reward += 10.0
		}
	}

	// Per turn alive.
	reward += 0.01

	// Eat food: detect by length increase.
	if prevG != nil {
		prevMe := &prevG.Snakes[myIdx]
		if me.Length > prevMe.Length {
			reward += 0.1
		}
	}

	// Territory advantage (only if opponent is alive).
	if oppIdx >= 0 && oppIdx < len(g.Snakes) && g.Snakes[oppIdx].IsAlive() {
		vr := logic.VoronoiTerritory(g, myIdx, true)
		reward += 0.01 * float32(vr.MyTerritory-vr.OppTerritory)

		// Opponent confined.
		opp := &g.Snakes[oppIdx]
		safeMoves := logic.SafeMoveCount(g, opp)
		if safeMoves <= 1 {
			reward += 1.0
		}
	}

	return reward
}
