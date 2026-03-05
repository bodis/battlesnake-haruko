package logic

import (
	"math"
	"time"
)

const (
	maxDepth    = 6  // paranoid minimax (retained for BestMove)
	brsMaxDepth = 14 // BRS ply depth cap
)

// killerTable stores up to 2 killer moves per depth level.
type killerTable [brsMaxDepth + 1][2]Direction

// searchContext carries a deadline, killer move table, and transposition table.
type searchContext struct {
	deadline   time.Time
	timedOut   bool
	killers    killerTable
	hasKillers [brsMaxDepth + 1][2]bool
	tt         *TranspositionTable
}

// storeKiller records a move that caused a beta cutoff at the given depth.
func (ctx *searchContext) storeKiller(depth int, d Direction) {
	if depth > brsMaxDepth {
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

// BestMoveIterative runs iterative deepening with BRS (Best-Reply Search).
// BRS alternates max (our) and min (opponent) plies instead of simultaneous moves,
// reducing pessimism and enabling deeper search than paranoid minimax.
func (g *GameSim) BestMoveIterative(myID string, budget time.Duration) Direction {
	var oppID string
	for i := range g.Snakes {
		s := &g.Snakes[i]
		if s.ID != myID && s.IsAlive() {
			oppID = s.ID
			break
		}
	}

	tt := getSharedTT()
	tt.NewGeneration()
	deadline := time.Now().Add(budget)
	bestDir := Down
	pvMove := Down
	hasPV := false

	for depth := 1; depth <= brsMaxDepth; depth++ {
		if depth > 1 {
			remaining := time.Until(deadline)
			if remaining < budget*3/10 {
				break
			}
		}

		ctx := &searchContext{deadline: deadline, tt: tt}
		depthBest := Down
		depthBestScore := math.Inf(-1)

		rootMoves := orderedMoves(pvMove, hasPV, ctx.killers[depth], ctx.hasKillers[depth])
		for _, myDir := range rootMoves {
			score := brsMin(g, depth-1, math.Inf(-1), math.Inf(1), myDir, myID, oppID, ctx)
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

// brsMax is the maximizing ply (our move) in Best-Reply Search.
func brsMax(g *GameSim, depth int, alpha, beta float64, myID, oppID string, ctx *searchContext) float64 {
	if time.Now().After(ctx.deadline) {
		ctx.timedOut = true
		return 0
	}

	if depth <= 0 || g.IsOver() {
		return Evaluate(g, myID)
	}

	alpha0 := alpha
	var hash uint64
	var ttMove Direction
	hasTTMove := false

	if ctx.tt != nil {
		hash = g.Hash()
		score, move, hasMove, hit := ctx.tt.Probe(hash, depth, alpha, beta)
		if hit {
			return score
		}
		ttMove = move
		hasTTMove = hasMove
	}

	bestScore := math.Inf(-1)
	bestMove := Down

	var moves [4]Direction
	if depth <= brsMaxDepth {
		moves = orderedMoves(ttMove, hasTTMove, ctx.killers[depth], ctx.hasKillers[depth])
	} else {
		moves = AllDirections
	}

	for _, myDir := range moves {
		val := brsMin(g, depth-1, alpha, beta, myDir, myID, oppID, ctx)
		if ctx.timedOut {
			break
		}
		if val > bestScore {
			bestScore = val
			bestMove = myDir
		}
		if val > alpha {
			alpha = val
		}
		if alpha >= beta {
			ctx.storeKiller(depth, myDir)
			break
		}
	}

	if ctx.tt != nil && !ctx.timedOut {
		ctx.tt.Store(hash, depth, bestScore, bestMove, alpha0, beta)
	}

	return bestScore
}

// brsMin is the minimizing ply (opponent's response) in Best-Reply Search.
// myDir is our pending move; this function picks the opponent's best reply,
// then applies both moves via Clone+Step.
func brsMin(g *GameSim, depth int, alpha, beta float64, myDir Direction, myID, oppID string, ctx *searchContext) float64 {
	if time.Now().After(ctx.deadline) {
		ctx.timedOut = true
		return 0
	}

	// No opponent — just simulate our move alone.
	if oppID == "" {
		sim := g.Clone()
		sim.Step(map[string]Direction{myID: myDir})
		if depth <= 0 || sim.IsOver() {
			return Evaluate(sim, myID)
		}
		return brsMax(sim, depth, alpha, beta, myID, oppID, ctx)
	}

	worstScore := math.Inf(1)

	var moves [4]Direction
	if depth <= brsMaxDepth {
		moves = orderedMoves(Down, false, ctx.killers[depth], ctx.hasKillers[depth])
	} else {
		moves = AllDirections
	}

	for _, oppDir := range moves {
		sim := g.Clone()
		sim.Step(map[string]Direction{myID: myDir, oppID: oppDir})

		var val float64
		if depth <= 0 || sim.IsOver() {
			val = Evaluate(sim, myID)
		} else {
			val = brsMax(sim, depth-1, alpha, beta, myID, oppID, ctx)
		}

		if ctx.timedOut {
			break
		}

		if val < worstScore {
			worstScore = val
		}
		if val < beta {
			beta = val
		}
		if beta <= alpha {
			ctx.storeKiller(depth, oppDir)
			break
		}
	}

	return worstScore
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

	alpha0 := alpha
	var hash uint64
	var ttMove Direction
	hasTTMove := false

	if ctx != nil && ctx.tt != nil {
		hash = g.Hash()
		score, move, hasMove, hit := ctx.tt.Probe(hash, depth, alpha, beta)
		if hit {
			return score
		}
		ttMove = move
		hasTTMove = hasMove
	}

	bestScore := math.Inf(-1)
	bestMove := Down

	var moves [4]Direction
	if ctx != nil && depth <= maxDepth {
		moves = orderedMoves(ttMove, hasTTMove, ctx.killers[depth], ctx.hasKillers[depth])
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
			bestMove = myDir
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

	// Store in TT (skip on timeout — partial results are unreliable).
	if ctx != nil && ctx.tt != nil && !ctx.timedOut {
		ctx.tt.Store(hash, depth, bestScore, bestMove, alpha0, beta)
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
