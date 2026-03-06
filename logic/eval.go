package logic

// isSafeDir returns true if moving snake s in direction d doesn't immediately
// hit a wall or any alive snake's body segment (segments 1..len-1).
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
		for seg := 1; seg < len(s2.Body); seg++ {
			if next == s2.Body[seg] {
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
	vr := VoronoiTerritory(g, myIdx)
	if vr.IsPartitioned && lateBlend < 0.5 {
		lateBlend = 0.5
	}
	wTerritory := 1.0 - 0.2*earlyBlend + 0.3*lateBlend
	score := wTerritory * float64(vr.MyTerritory-vr.OppTerritory)

	// Early-game food control.
	score += 1.5 * earlyBlend * float64(vr.MyFood)

	// Phase-modulated weights.
	wLen := 2.0 + 1.0*earlyBlend - 0.5*lateBlend
	wH2H := 5.0 - 2.0*lateBlend

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

		// Opponent confinement.
		switch safeMoveCount(g, opp) {
		case 0:
			score += 50.0
		case 1:
			score += 15.0
		}
	}

	// Self-confinement penalty.
	switch safeMoveCount(g, me) {
	case 0:
		score -= 25.0
	case 1:
		score -= 5.0
	}

	// Tail chase bonus: reward proximity to own tail when space is tight.
	if lateBlend > 0 {
		tail := me.Tail()
		tailDist := abs(myHead.X-tail.X) + abs(myHead.Y-tail.Y)
		if tailDist > 0 {
			score += 3.0 * lateBlend / float64(tailDist)
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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
