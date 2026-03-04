package logic

import (
	"math"
	"time"
)

const maxDepth = 5

// killerTable stores up to 2 killer moves per depth level.
type killerTable [maxDepth + 1][2]Direction

// searchContext carries a deadline and killer move table for iterative deepening.
type searchContext struct {
	deadline   time.Time
	timedOut   bool
	killers    killerTable
	hasKillers [maxDepth + 1][2]bool
}

// storeKiller records a move that caused a beta cutoff at the given depth.
func (ctx *searchContext) storeKiller(depth int, d Direction) {
	if depth > maxDepth {
		return
	}
	// Don't store duplicates.
	if ctx.hasKillers[depth][0] && ctx.killers[depth][0] == d {
		return
	}
	// Shift slot 0 → slot 1, store new in slot 0.
	ctx.killers[depth][1] = ctx.killers[depth][0]
	ctx.hasKillers[depth][1] = ctx.hasKillers[depth][0]
	ctx.killers[depth][0] = d
	ctx.hasKillers[depth][0] = true
}

// orderedMoves returns a [4]Direction array with prioritized ordering:
// PV move first, then killer moves, then remaining directions.
func orderedMoves(pv Direction, hasPV bool, killers [2]Direction, hasKillers [2]bool) [4]Direction {
	var result [4]Direction
	used := [4]bool{}
	n := 0

	add := func(d Direction) {
		if !used[d] {
			result[n] = d
			used[d] = true
			n++
		}
	}

	if hasPV {
		add(pv)
	}
	if hasKillers[0] {
		add(killers[0])
	}
	if hasKillers[1] {
		add(killers[1])
	}
	for _, d := range AllDirections {
		add(d)
	}
	return result
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
	pvMove := Down
	hasPV := false

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

		rootMoves := orderedMoves(pvMove, hasPV, [2]Direction{}, [2]bool{})
		for _, myDir := range rootMoves {
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
		pvMove = depthBest
		hasPV = true
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

	var moves [4]Direction
	if ctx != nil && depth <= maxDepth {
		moves = orderedMoves(Down, false, ctx.killers[depth], ctx.hasKillers[depth])
	} else {
		moves = AllDirections
	}

	for _, myDir := range moves {
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
			if ctx != nil {
				ctx.storeKiller(depth, myDir)
			}
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
