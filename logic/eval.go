package logic

// tailStacked returns true if the snake's tail is stacked (doubled up),
// meaning it won't retract next turn (e.g. after eating).
func tailStacked(s *SimSnake) bool {
	n := len(s.Body)
	return n >= 2 && s.Body[n-1] == s.Body[n-2]
}

// foodAdjacentToHead returns true if any food is within Manhattan distance 1
// of the snake's head (i.e. the snake might eat next turn, preventing tail retraction).
func foodAdjacentToHead(g *GameSim, s *SimSnake) bool {
	head := s.Head()
	for _, f := range g.Food {
		dx := head.X - f.X
		if dx < 0 {
			dx = -dx
		}
		dy := head.Y - f.Y
		if dy < 0 {
			dy = -dy
		}
		if dx+dy <= 1 {
			return true
		}
	}
	return false
}

// isSafeDir returns true if moving snake s in direction d doesn't immediately
// hit a wall or any alive snake's body segment. Tail segments are skipped when
// the tail will retract (not stacked and no food adjacent to that snake's head).
func isSafeDir(g *GameSim, s *SimSnake, d Direction) bool {
	next := s.Head().Move(d)
	if next.X < 0 || next.X >= g.Width || next.Y < 0 || next.Y >= g.Height {
		return false
	}
	for j := range g.Snakes {
		s2 := &g.Snakes[j]
		if !s2.IsAlive() {
			continue
		}
		lastSeg := len(s2.Body) - 1
		tailRetracts := lastSeg >= 1 && !tailStacked(s2) && !foodAdjacentToHead(g, s2)
		for seg := 1; seg < len(s2.Body); seg++ {
			if next == s2.Body[seg] {
				if tailRetracts && seg == lastSeg {
					continue
				}
				return false
			}
		}
	}
	return true
}

// safeMoveCount returns the number of directions from s's head that don't
// immediately hit a wall or any alive snake's body (segments 1..len-1).
func safeMoveCount(g *GameSim, s *SimSnake) int {
	count := 0
	for _, d := range AllDirections {
		if isSafeDir(g, s, d) {
			count++
		}
	}
	return count
}

// clamp01 clamps x to [0.0, 1.0].
func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// Evaluate scores a GameSim position from myIdx's perspective.
// Returns -1000 if we're dead, +1000 if all opponents are dead,
// otherwise phase-weighted: Voronoi territory + length advantage +
// head-to-head pressure + opponent confinement + food urgency + food control.
func Evaluate(g *GameSim, myIdx int) float64 {
	me := &g.Snakes[myIdx]
	if !me.IsAlive() {
		return -1000
	}

	// Check for alive opponents.
	allOppsDead := true
	for i := range g.Snakes {
		if i != myIdx && g.Snakes[i].IsAlive() {
			allOppsDead = false
			break
		}
	}
	if allOppsDead {
		return 1000
	}

	// Phase blend factors (continuous 0.0-1.0).
	earlyByLen := clamp01(float64(8-me.Length) / 4.0)  // 1.0@len4, 0.0@len8+
	earlyByTurn := clamp01(float64(35-g.Turn) / 20.0)  // 1.0@turn15, 0.0@turn35+
	earlyBlend := earlyByLen
	if earlyByTurn > earlyBlend {
		earlyBlend = earlyByTurn
	}

	totalBody := 0
	for i := range g.Snakes {
		if g.Snakes[i].IsAlive() {
			totalBody += g.Snakes[i].Length
		}
	}
	boardFill := float64(totalBody) / float64(g.Width*g.Height)
	lateBlend := clamp01((boardFill - 0.30) / 0.20) // 0.0@30%, 1.0@50%+

	// Territory score with phase modulation.
	// Always skip Tarjan AP — bottleneck signal was anti-correlated with wins
	// (winners have more threatened territory due to aggressive squeezing).
	vr := VoronoiTerritory(g, myIdx, true)
	if vr.IsPartitioned && lateBlend < 0.5 {
		lateBlend = 0.5
	}
	wTerritory := 1.5 - 0.3*earlyBlend + 0.45*lateBlend
	score := wTerritory * float64(vr.MyTerritory-vr.OppTerritory)

	// Early-game food control (distance-weighted, not flat count).
	score += 1.0 * earlyBlend * vr.MyFoodValue // wFoodCluster

	// Growth urgency: penalize being undersized in early game.
	if earlyBlend > 0 {
		expectedLen := 3 + g.Turn/10
		if me.Length < expectedLen {
			score -= 0.3 * earlyBlend * float64(expectedLen-me.Length) // wGrowthUrgency
		}
	}

	// Phase-modulated weights.
	wLen := 3.0 + 1.5*earlyBlend - 0.75*lateBlend
	wH2H := 8.0 - 3.2*lateBlend

	// Accumulate per-opponent scores.
	myHead := me.Head()
	for i := range g.Snakes {
		opp := &g.Snakes[i]
		if i == myIdx || !opp.IsAlive() {
			continue
		}

		// Length advantage.
		score += wLen * float64(me.Length-opp.Length)

		// Head-to-head pressure.
		oppHead := opp.Head()
		dist := abs(myHead.X-oppHead.X) + abs(myHead.Y-oppHead.Y)
		if dist <= 2 {
			if me.Length > opp.Length {
				score += wH2H
			} else if me.Length < opp.Length {
				score -= wH2H
			}
		}

		// Opponent confinement (stronger in late game where confinement kills).
		switch safeMoveCount(g, opp) {
		case 0:
			score += 50.0 + 25.0*lateBlend
		case 1:
			score += 15.0 + 10.0*lateBlend
		}
	}

	// Self-confinement penalty (stronger in late game).
	switch safeMoveCount(g, me) {
	case 0:
		score -= 25.0 + 25.0*lateBlend
	case 1:
		score -= 5.0 + 10.0*lateBlend
	}

	// Tail chase bonus: reward proximity to own tail when space is tight.
	if lateBlend > 0 {
		tail := me.Tail()
		tailDist := abs(myHead.X-tail.X) + abs(myHead.Y-tail.Y)
		if tailDist > 0 {
			score += 5.0 * lateBlend / float64(tailDist)
		}
	}

	// Food urgency: phase-modulated threshold.
	foodThreshold := 40 + int(15*earlyBlend)
	if me.Health < foodThreshold && len(g.Food) > 0 {
		head := me.Head()
		minDist := 999
		for _, f := range g.Food {
			d := abs(head.X-f.X) + abs(head.Y-f.Y)
			if d < minDist {
				minDist = d
			}
		}
		foodWeight := float64(foodThreshold-me.Health) * 0.5
		score += foodWeight * (1.0 / float64(max(minDist, 1)))
	}

	return score
}

// EvalBreakdown holds per-signal contributions for diagnostic tracing.
type EvalBreakdown struct {
	Total           float64
	Territory       float64
	LenAdvantage    float64
	H2H             float64
	OppConfinement  float64
	SelfConfinement float64
	FoodUrgency     float64
	FoodCluster     float64
	GrowthUrgency   float64
	TailChase       float64
	EarlyBlend      float64
	LateBlend       float64
}

// EvaluateDetailed mirrors Evaluate but returns a per-signal breakdown.
// Not used on the hot path — called once per turn for tracing.
func EvaluateDetailed(g *GameSim, myIdx int) EvalBreakdown {
	var b EvalBreakdown
	me := &g.Snakes[myIdx]
	if !me.IsAlive() {
		b.Total = -1000
		return b
	}

	allOppsDead := true
	for i := range g.Snakes {
		if i != myIdx && g.Snakes[i].IsAlive() {
			allOppsDead = false
			break
		}
	}
	if allOppsDead {
		b.Total = 1000
		return b
	}

	// Phase blend factors.
	earlyByLen := clamp01(float64(8-me.Length) / 4.0)
	earlyByTurn := clamp01(float64(35-g.Turn) / 20.0)
	earlyBlend := earlyByLen
	if earlyByTurn > earlyBlend {
		earlyBlend = earlyByTurn
	}
	b.EarlyBlend = earlyBlend

	totalBody := 0
	for i := range g.Snakes {
		if g.Snakes[i].IsAlive() {
			totalBody += g.Snakes[i].Length
		}
	}
	boardFill := float64(totalBody) / float64(g.Width*g.Height)
	lateBlend := clamp01((boardFill - 0.30) / 0.20)

	vr := VoronoiTerritory(g, myIdx, true)
	if vr.IsPartitioned && lateBlend < 0.5 {
		lateBlend = 0.5
	}
	b.LateBlend = lateBlend

	wTerritory := 1.5 - 0.3*earlyBlend + 0.45*lateBlend
	b.Territory = wTerritory * float64(vr.MyTerritory-vr.OppTerritory)

	// Food cluster.
	b.FoodCluster = 1.0 * earlyBlend * vr.MyFoodValue

	// Growth urgency.
	if earlyBlend > 0 {
		expectedLen := 3 + g.Turn/10
		if me.Length < expectedLen {
			b.GrowthUrgency = -(0.3 * earlyBlend * float64(expectedLen-me.Length))
		}
	}

	wLen := 3.0 + 1.5*earlyBlend - 0.75*lateBlend
	wH2H := 8.0 - 3.2*lateBlend

	myHead := me.Head()
	for i := range g.Snakes {
		opp := &g.Snakes[i]
		if i == myIdx || !opp.IsAlive() {
			continue
		}

		b.LenAdvantage += wLen * float64(me.Length-opp.Length)

		oppHead := opp.Head()
		dist := abs(myHead.X-oppHead.X) + abs(myHead.Y-oppHead.Y)
		if dist <= 2 {
			if me.Length > opp.Length {
				b.H2H += wH2H
			} else if me.Length < opp.Length {
				b.H2H -= wH2H
			}
		}

		switch safeMoveCount(g, opp) {
		case 0:
			b.OppConfinement += 50.0 + 25.0*lateBlend
		case 1:
			b.OppConfinement += 15.0 + 10.0*lateBlend
		}
	}

	switch safeMoveCount(g, me) {
	case 0:
		b.SelfConfinement = -(25.0 + 25.0*lateBlend)
	case 1:
		b.SelfConfinement = -(5.0 + 10.0*lateBlend)
	}

	// Tail chase.
	if lateBlend > 0 {
		tail := me.Tail()
		tailDist := abs(myHead.X-tail.X) + abs(myHead.Y-tail.Y)
		if tailDist > 0 {
			b.TailChase = 5.0 * lateBlend / float64(tailDist)
		}
	}

	// Food urgency.
	foodThreshold := 40 + int(15*earlyBlend)
	if me.Health < foodThreshold && len(g.Food) > 0 {
		head := me.Head()
		minDist := 999
		for _, f := range g.Food {
			d := abs(head.X-f.X) + abs(head.Y-f.Y)
			if d < minDist {
				minDist = d
			}
		}
		foodWeight := float64(foodThreshold-me.Health) * 0.5
		b.FoodUrgency = foodWeight * (1.0 / float64(max(minDist, 1)))
	}

	b.Total = b.Territory + b.LenAdvantage + b.H2H + b.OppConfinement +
		b.SelfConfinement + b.FoodUrgency + b.FoodCluster + b.GrowthUrgency +
		b.TailChase

	return b
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
