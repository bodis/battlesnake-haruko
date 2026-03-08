package logic

import (
	"math"
	"time"
)

const (
	mctsExploreC = 1.41421356 // sqrt(2) for UCB1
	mctsBatchSize = 64        // sims between time checks
)

// mctsRoot holds flat (depth-1) MCTS statistics for each root direction.
type mctsRoot struct {
	visits [4]int32
	total  [4]float64
}

// selectUCB1 picks the direction with the highest UCB1 score among safe moves.
func (r *mctsRoot) selectUCB1(safeMask [4]bool) Direction {
	totalVisits := int32(0)
	for _, v := range r.visits {
		totalVisits += v
	}

	best := Direction(0)
	bestUCB := math.Inf(-1)
	lnTotal := math.Log(float64(totalVisits + 1))

	for d := Direction(0); d < 4; d++ {
		if !safeMask[d] {
			continue
		}
		if r.visits[d] == 0 {
			return d // unvisited — try immediately
		}
		mean := r.total[d] / float64(r.visits[d])
		ucb := mean + mctsExploreC*math.Sqrt(lnTotal/float64(r.visits[d]))
		if ucb > bestUCB {
			bestUCB = ucb
			best = d
		}
	}
	return best
}

// xorshift64 is an inline zero-alloc PRNG.
type xorshift64 uint64

func (x *xorshift64) next() uint64 {
	v := uint64(*x)
	v ^= v << 13
	v ^= v >> 7
	v ^= v << 17
	*x = xorshift64(v)
	return v
}

// mctsSearch runs flat depth-1 MCTS simulations within the given budget.
// Each simulation: pick root dir (UCB1), pick random opp dir, step, evaluate.
func mctsSearch(g *GameSim, myIdx, oppIdx int, budget time.Duration) mctsRoot {
	var root mctsRoot
	deadline := time.Now().Add(budget)

	// Pre-compute wall-safe masks.
	var mySafe, oppSafe [4]bool
	mySafeCount := 0
	myHead := g.Snakes[myIdx].Head()
	for _, d := range AllDirections {
		next := myHead.Move(d)
		if next.X >= 0 && next.X < g.Width && next.Y >= 0 && next.Y < g.Height {
			mySafe[d] = true
			mySafeCount++
		}
	}

	// Build opp safe direction list for random selection.
	var oppSafeDirs [4]Direction
	oppSafeCount := 0
	if oppIdx >= 0 {
		oppHead := g.Snakes[oppIdx].Head()
		for _, d := range AllDirections {
			next := oppHead.Move(d)
			if next.X >= 0 && next.X < g.Width && next.Y >= 0 && next.Y < g.Height {
				oppSafe[d] = true
				oppSafeDirs[oppSafeCount] = d
				oppSafeCount++
			}
		}
	}
	_ = oppSafe

	if mySafeCount == 0 {
		return root // no safe moves
	}

	// Seed PRNG from game hash (or turn if no hash).
	rng := xorshift64(g.Hash() ^ uint64(g.Turn) | 1) // ensure nonzero

	for {
		for batch := 0; batch < mctsBatchSize; batch++ {
			myDir := root.selectUCB1(mySafe)

			var sim *GameSim
			if oppIdx >= 0 && oppSafeCount > 0 {
				oppDir := oppSafeDirs[rng.next()%uint64(oppSafeCount)]
				sim = g.CloneFromPool()
				sim.Step(newMoveSet2(myIdx, myDir, oppIdx, oppDir))
			} else {
				sim = g.CloneFromPool()
				sim.Step(newMoveSet1(myIdx, myDir))
			}

			score := Evaluate(sim, myIdx)
			sim.Release()

			root.visits[myDir]++
			root.total[myDir] += score
		}

		if time.Now().After(deadline) {
			break
		}
	}

	return root
}
