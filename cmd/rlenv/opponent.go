package main

import (
	"math/rand"
	"time"

	"github.com/bodist/haruko/logic"
)

// OpponentPolicy defines the interface for opponent move selection.
type OpponentPolicy interface {
	ChooseMove(g *logic.GameSim, oppIdx int) logic.Direction
}

// RandomPolicy picks a random safe direction. Falls back to a random
// direction (including unsafe) if no safe moves exist.
type RandomPolicy struct {
	rng *rand.Rand
}

// ChooseMove selects a random safe move for the opponent.
func (p *RandomPolicy) ChooseMove(g *logic.GameSim, oppIdx int) logic.Direction {
	if p.rng == nil {
		p.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	opp := &g.Snakes[oppIdx]
	if !opp.IsAlive() {
		return logic.Down
	}

	// Collect safe directions.
	var safe []logic.Direction
	for _, d := range logic.AllDirections {
		if logic.IsSafeDir(g, opp, d) {
			safe = append(safe, d)
		}
	}

	if len(safe) > 0 {
		return safe[p.rng.Intn(len(safe))]
	}

	// No safe moves: pick random.
	return logic.AllDirections[p.rng.Intn(4)]
}

// BRSPolicy uses the BRS search engine to select the opponent's move.
// It clones the game state to avoid mutating the original.
type BRSPolicy struct {
	Budget time.Duration
}

// ChooseMove runs BestMoveIterative on a clone and returns the result.
func (p *BRSPolicy) ChooseMove(g *logic.GameSim, oppIdx int) logic.Direction {
	opp := &g.Snakes[oppIdx]
	if !opp.IsAlive() {
		return logic.Down
	}

	clone := g.Clone()
	budget := p.Budget
	if budget <= 0 {
		budget = 50 * time.Millisecond
	}
	return clone.BestMoveIterative(opp.ID, budget)
}
