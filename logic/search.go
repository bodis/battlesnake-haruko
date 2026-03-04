package logic

import "math"

// BestMove runs 1-ply paranoid minimax: for each of our candidate moves,
// enumerate all opponent move combinations, simulate, score, and return
// the move that maximises our worst-case Evaluate score.
func (g *GameSim) BestMove(myID string) Direction {
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
		worstScore := math.Inf(1)

		forEachOppCombo(oppIDs, func(oppMoves map[string]Direction) {
			moves := make(map[string]Direction, len(oppMoves)+1)
			for id, d := range oppMoves {
				moves[id] = d
			}
			moves[myID] = myDir

			sim := g.Clone()
			sim.Step(moves)
			score := Evaluate(sim, myID)
			if score < worstScore {
				worstScore = score
			}
		})

		// If there are no opponents, worstScore stays +Inf — treat it as
		// the eval of the resulting state directly.
		if math.IsInf(worstScore, 1) {
			sim := g.Clone()
			sim.Step(map[string]Direction{myID: myDir})
			worstScore = Evaluate(sim, myID)
		}

		if worstScore > bestScore {
			bestScore = worstScore
			bestDir = myDir
		}
	}

	return bestDir
}

// forEachOppCombo calls fn with every combination of moves for oppIDs.
func forEachOppCombo(ids []string, fn func(map[string]Direction)) {
	m := make(map[string]Direction, len(ids))
	forEachOppComboRec(ids, 0, m, fn)
}

func forEachOppComboRec(ids []string, idx int, m map[string]Direction, fn func(map[string]Direction)) {
	if idx == len(ids) {
		fn(m)
		return
	}
	for _, d := range AllDirections {
		m[ids[idx]] = d
		forEachOppComboRec(ids, idx+1, m, fn)
	}
}
