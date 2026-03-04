package logic

import (
	"math"
	"time"
)

// searchContext carries a deadline for iterative deepening time management.
type searchContext struct {
	deadline time.Time
	timedOut bool
}

// BestMove runs paranoid minimax with alpha-beta pruning to the given depth.
// Each depth level = one simultaneous turn (our move + all opponent moves + Step).
func (g *GameSim) BestMove(myID string, depth int) Direction {
	// Collect alive opponent IDs.
	var oppIDs []string
	for i := range g.Snakes {
		s := &g.Snakes[i]
		if s.ID != myID && s.IsAlive() {
			oppIDs = append(oppIDs, s.ID)
		}
	}

	bestDir := Down
	bestScore := math.Inf(-1)

	for _, myDir := range AllDirections {
		score := minimaxMin(g, depth, math.Inf(-1), math.Inf(1), myDir, myID, oppIDs, nil)
		if score > bestScore {
			bestScore = score
			bestDir = myDir
		}
	}

	return bestDir
}

// BestMoveIterative runs iterative deepening with a time budget.
// It searches depth 1, 2, 3, ... and returns the best move from the deepest
// completed search. Always has a valid move ready from at least depth 1.
func (g *GameSim) BestMoveIterative(myID string, budget time.Duration) Direction {
	var oppIDs []string
	for i := range g.Snakes {
		s := &g.Snakes[i]
		if s.ID != myID && s.IsAlive() {
			oppIDs = append(oppIDs, s.ID)
		}
	}

	deadline := time.Now().Add(budget)
	bestDir := Down

	const maxDepth = 5
	for depth := 1; depth <= maxDepth; depth++ {
		// Before starting depth > 1, check if enough budget remains.
		if depth > 1 {
			remaining := time.Until(deadline)
			if remaining < budget*3/10 {
				break
			}
		}

		ctx := &searchContext{deadline: deadline}
		depthBest := Down
		depthBestScore := math.Inf(-1)

		for _, myDir := range AllDirections {
			score := minimaxMin(g, depth, math.Inf(-1), math.Inf(1), myDir, myID, oppIDs, ctx)
			if ctx.timedOut {
				break
			}
			if score > depthBestScore {
				depthBestScore = score
				depthBest = myDir
			}
		}

		if ctx.timedOut {
			break
		}
		bestDir = depthBest
	}

	return bestDir
}

// minimaxMin is the minimizing layer: enumerates all opponent move combos,
// applies our move + opponent moves via Step, and returns the worst-case score.
func minimaxMin(g *GameSim, depth int, alpha, beta float64, myDir Direction, myID string, oppIDs []string, ctx *searchContext) float64 {
	if ctx != nil && time.Now().After(ctx.deadline) {
		ctx.timedOut = true
		return 0
	}

	// No opponents — just simulate our move alone.
	if len(oppIDs) == 0 {
		sim := g.Clone()
		sim.Step(map[string]Direction{myID: myDir})
		if depth <= 1 || sim.IsOver() {
			return Evaluate(sim, myID)
		}
		return minimaxMax(sim, depth-1, alpha, beta, myID, oppIDs, ctx)
	}

	worstScore := math.Inf(1)

	forEachOppCombo(oppIDs, func(oppMoves map[string]Direction) bool {
		if ctx != nil && ctx.timedOut {
			return false
		}

		moves := make(map[string]Direction, len(oppMoves)+1)
		for id, d := range oppMoves {
			moves[id] = d
		}
		moves[myID] = myDir

		sim := g.Clone()
		sim.Step(moves)

		var val float64
		if depth <= 1 || sim.IsOver() {
			val = Evaluate(sim, myID)
		} else {
			val = minimaxMax(sim, depth-1, alpha, beta, myID, oppIDs, ctx)
		}

		if ctx != nil && ctx.timedOut {
			return false
		}

		if val < worstScore {
			worstScore = val
		}
		if val < beta {
			beta = val
		}
		// Beta cutoff: maximizer already has a better option.
		return beta > alpha // return false to stop iteration
	})

	return worstScore
}

// minimaxMax is the maximizing layer: tries each of our 4 moves and returns
// the best score after the opponent responds (via minimaxMin).
func minimaxMax(g *GameSim, depth int, alpha, beta float64, myID string, oppIDs []string, ctx *searchContext) float64 {
	if ctx != nil && time.Now().After(ctx.deadline) {
		ctx.timedOut = true
		return 0
	}

	bestScore := math.Inf(-1)

	for _, myDir := range AllDirections {
		val := minimaxMin(g, depth, alpha, beta, myDir, myID, oppIDs, ctx)
		if ctx != nil && ctx.timedOut {
			break
		}
		if val > bestScore {
			bestScore = val
		}
		if val > alpha {
			alpha = val
		}
		// Alpha cutoff: minimizer already has a better option.
		if alpha >= beta {
			break
		}
	}

	return bestScore
}

// forEachOppCombo calls fn with every combination of moves for oppIDs.
// fn returns true to continue, false to stop early (for alpha-beta cutoffs).
func forEachOppCombo(ids []string, fn func(map[string]Direction) bool) {
	m := make(map[string]Direction, len(ids))
	forEachOppComboRec(ids, 0, m, fn)
}

// forEachOppComboRec returns false if iteration was stopped early.
func forEachOppComboRec(ids []string, idx int, m map[string]Direction, fn func(map[string]Direction) bool) bool {
	if idx == len(ids) {
		return fn(m)
	}
	for _, d := range AllDirections {
		m[ids[idx]] = d
		if !forEachOppComboRec(ids, idx+1, m, fn) {
			return false
		}
	}
	return true
}
