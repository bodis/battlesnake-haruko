package logic

import (
	"math"
	"time"
)

const (
	maxDepth    = 6  // paranoid minimax (retained for BestMove)
	brsMaxDepth = 14 // BRS ply depth cap
	qsMaxDepth  = 1  // quiescence search ply cap
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
	if ctx.hasKillers[depth][0] && ctx.killers[depth][0] == d {
		return
	}
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

// ResolveIdx finds the snake index for the given ID.
func (g *GameSim) ResolveIdx(id string) int {
	for i := range g.Snakes {
		if g.Snakes[i].ID == id {
			return i
		}
	}
	return -1
}

// BestMove runs paranoid minimax with alpha-beta pruning to the given depth.
func (g *GameSim) BestMove(myID string, depth int) Direction {
	myIdx := g.ResolveIdx(myID)
	if myIdx == -1 {
		return Down
	}

	var oppIdxs []int
	for i := range g.Snakes {
		if i != myIdx && g.Snakes[i].IsAlive() {
			oppIdxs = append(oppIdxs, i)
		}
	}

	bestDir := Down
	bestScore := math.Inf(-1)

	for _, myDir := range AllDirections {
		score := minimaxMin(g, depth, math.Inf(-1), math.Inf(1), myDir, myIdx, oppIdxs, nil)
		if score > bestScore {
			bestScore = score
			bestDir = myDir
		}
	}

	return bestDir
}

// BestMoveIterative runs iterative deepening with BRS (Best-Reply Search).
func (g *GameSim) BestMoveIterative(myID string, budget time.Duration) Direction {
	myIdx := g.ResolveIdx(myID)
	if myIdx == -1 {
		return Down
	}

	oppIdx := -1
	for i := range g.Snakes {
		if i != myIdx && g.Snakes[i].IsAlive() {
			oppIdx = i
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
			score := brsMin(g, depth-1, math.Inf(-1), math.Inf(1), myDir, myIdx, oppIdx, ctx)
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
		g.LastCompletedDepth = depth
	}

	return bestDir
}

// isQuiet returns true if the position is calm (no imminent combat).
func isQuiet(g *GameSim, myIdx, oppIdx int) bool {
	me := &g.Snakes[myIdx]
	if !me.IsAlive() {
		return true
	}

	for i := range g.Snakes {
		s := &g.Snakes[i]
		if !s.IsAlive() {
			continue
		}
		if safeMoveCount(g, s) == 0 {
			return false
		}
	}

	myHead := me.Head()
	for i := range g.Snakes {
		s := &g.Snakes[i]
		if i == myIdx || !s.IsAlive() {
			continue
		}
		oppHead := s.Head()
		dist := abs(myHead.X-oppHead.X) + abs(myHead.Y-oppHead.Y)
		if dist <= 1 {
			return false
		}
	}
	return true
}

// forcingMoves returns moves for snakeIdx that reduce Manhattan distance
// to oppIdx's head (i.e., moves toward the fight).
func forcingMoves(g *GameSim, snakeIdx, oppIdx int) []Direction {
	me := &g.Snakes[snakeIdx]
	opp := &g.Snakes[oppIdx]
	if !me.IsAlive() || !opp.IsAlive() {
		return nil
	}

	myHead := me.Head()
	oppHead := opp.Head()
	curDist := abs(myHead.X-oppHead.X) + abs(myHead.Y-oppHead.Y)

	var moves []Direction
	for _, d := range AllDirections {
		next := myHead.Move(d)
		newDist := abs(next.X-oppHead.X) + abs(next.Y-oppHead.Y)
		if newDist < curDist {
			moves = append(moves, d)
		}
	}
	return moves
}

// qsMax is the quiescence maximizer. It extends search in volatile positions.
func qsMax(g *GameSim, qsDepth int, alpha, beta float64, myIdx, oppIdx int, ctx *searchContext) float64 {
	if time.Now().After(ctx.deadline) {
		ctx.timedOut = true
		return 0
	}

	standPat := Evaluate(g, myIdx)

	if g.IsOver() || qsDepth <= 0 {
		return standPat
	}
	if isQuiet(g, myIdx, oppIdx) {
		return standPat
	}

	if standPat >= beta {
		return standPat
	}
	if standPat > alpha {
		alpha = standPat
	}

	moves := forcingMoves(g, myIdx, oppIdx)
	if len(moves) == 0 {
		return standPat
	}

	best := standPat
	for _, myDir := range moves {
		val := qsMin(g, qsDepth-1, alpha, beta, myDir, myIdx, oppIdx, ctx)
		if ctx.timedOut {
			break
		}
		if val > best {
			best = val
		}
		if val > alpha {
			alpha = val
		}
		if alpha >= beta {
			break
		}
	}
	return best
}

// qsMin is the quiescence minimizer. Applies myDir + opponent forcing move, then steps.
func qsMin(g *GameSim, qsDepth int, alpha, beta float64, myDir Direction, myIdx, oppIdx int, ctx *searchContext) float64 {
	if time.Now().After(ctx.deadline) {
		ctx.timedOut = true
		return 0
	}

	// No opponent — just simulate our move.
	if oppIdx == -1 {
		sim := g.CloneFromPool()
		sim.Step(newMoveSet1(myIdx, myDir))
		val := Evaluate(sim, myIdx)
		sim.Release()
		return val
	}

	moves := forcingMoves(g, oppIdx, myIdx)
	if len(moves) == 0 {
		sim := g.CloneFromPool()
		sim.Step(newMoveSet2(myIdx, myDir, oppIdx, Down))
		val := Evaluate(sim, myIdx)
		sim.Release()
		return val
	}

	worst := math.Inf(1)
	for _, oppDir := range moves {
		sim := g.CloneFromPool()
		sim.Step(newMoveSet2(myIdx, myDir, oppIdx, oppDir))

		var val float64
		if sim.IsOver() || qsDepth <= 0 {
			val = Evaluate(sim, myIdx)
		} else {
			val = qsMax(sim, qsDepth, alpha, beta, myIdx, oppIdx, ctx)
		}
		sim.Release()

		if ctx.timedOut {
			break
		}
		if val < worst {
			worst = val
		}
		if val < beta {
			beta = val
		}
		if beta <= alpha {
			break
		}
	}
	return worst
}

// brsMax is the maximizing ply (our move) in Best-Reply Search.
func brsMax(g *GameSim, depth int, alpha, beta float64, myIdx, oppIdx int, ctx *searchContext) float64 {
	if time.Now().After(ctx.deadline) {
		ctx.timedOut = true
		return 0
	}

	if depth <= 0 || g.IsOver() {
		return Evaluate(g, myIdx)
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
		val := brsMin(g, depth-1, alpha, beta, myDir, myIdx, oppIdx, ctx)
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
func brsMin(g *GameSim, depth int, alpha, beta float64, myDir Direction, myIdx, oppIdx int, ctx *searchContext) float64 {
	if time.Now().After(ctx.deadline) {
		ctx.timedOut = true
		return 0
	}

	// No opponent — just simulate our move alone.
	if oppIdx == -1 {
		sim := g.CloneFromPool()
		sim.Step(newMoveSet1(myIdx, myDir))
		if depth <= 0 || sim.IsOver() {
			val := Evaluate(sim, myIdx)
			sim.Release()
			return val
		}
		val := brsMax(sim, depth, alpha, beta, myIdx, oppIdx, ctx)
		sim.Release()
		return val
	}

	worstScore := math.Inf(1)

	var moves [4]Direction
	if depth <= brsMaxDepth {
		moves = orderedMoves(Down, false, ctx.killers[depth], ctx.hasKillers[depth])
	} else {
		moves = AllDirections
	}

	for _, oppDir := range moves {
		sim := g.CloneFromPool()
		sim.Step(newMoveSet2(myIdx, myDir, oppIdx, oppDir))

		var val float64
		if depth <= 0 || sim.IsOver() {
			val = Evaluate(sim, myIdx)
		} else {
			val = brsMax(sim, depth-1, alpha, beta, myIdx, oppIdx, ctx)
		}
		sim.Release()

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
func minimaxMin(g *GameSim, depth int, alpha, beta float64, myDir Direction, myIdx int, oppIdxs []int, ctx *searchContext) float64 {
	if ctx != nil && time.Now().After(ctx.deadline) {
		ctx.timedOut = true
		return 0
	}

	// No opponents — just simulate our move alone.
	if len(oppIdxs) == 0 {
		sim := g.CloneFromPool()
		sim.Step(newMoveSet1(myIdx, myDir))
		if depth <= 1 || sim.IsOver() {
			val := Evaluate(sim, myIdx)
			sim.Release()
			return val
		}
		val := minimaxMax(sim, depth-1, alpha, beta, myIdx, oppIdxs, ctx)
		sim.Release()
		return val
	}

	worstScore := math.Inf(1)

	forEachOppCombo(oppIdxs, func(oppMoves MoveSet) bool {
		if ctx != nil && ctx.timedOut {
			return false
		}

		// Merge our move into oppMoves.
		oppMoves.Dir[myIdx] = myDir
		oppMoves.Has[myIdx] = true

		sim := g.CloneFromPool()
		sim.Step(oppMoves)

		var val float64
		if depth <= 1 || sim.IsOver() {
			val = Evaluate(sim, myIdx)
		} else {
			val = minimaxMax(sim, depth-1, alpha, beta, myIdx, oppIdxs, ctx)
		}
		sim.Release()

		if ctx != nil && ctx.timedOut {
			return false
		}

		if val < worstScore {
			worstScore = val
		}
		if val < beta {
			beta = val
		}
		return beta > alpha
	})

	return worstScore
}

// minimaxMax is the maximizing layer: tries each of our 4 moves and returns
// the best score after the opponent responds (via minimaxMin).
func minimaxMax(g *GameSim, depth int, alpha, beta float64, myIdx int, oppIdxs []int, ctx *searchContext) float64 {
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
		val := minimaxMin(g, depth, alpha, beta, myDir, myIdx, oppIdxs, ctx)
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
		if alpha >= beta {
			if ctx != nil {
				ctx.storeKiller(depth, myDir)
			}
			break
		}
	}

	if ctx != nil && ctx.tt != nil && !ctx.timedOut {
		ctx.tt.Store(hash, depth, bestScore, bestMove, alpha0, beta)
	}

	return bestScore
}

// forEachOppCombo calls fn with every combination of moves for oppIdxs.
// fn returns true to continue, false to stop early (for alpha-beta cutoffs).
func forEachOppCombo(idxs []int, fn func(MoveSet) bool) {
	var ms MoveSet
	forEachOppComboRec(idxs, 0, &ms, fn)
}

// forEachOppComboRec returns false if iteration was stopped early.
func forEachOppComboRec(idxs []int, pos int, ms *MoveSet, fn func(MoveSet) bool) bool {
	if pos == len(idxs) {
		return fn(*ms)
	}
	i := idxs[pos]
	for _, d := range AllDirections {
		ms.Dir[i] = d
		ms.Has[i] = true
		if !forEachOppComboRec(idxs, pos+1, ms, fn) {
			return false
		}
	}
	return true
}
