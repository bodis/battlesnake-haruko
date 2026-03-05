package logic

// safeMoveCount returns the number of directions from s's head that don't
// immediately hit a wall or any alive snake's body (segments 1..len-1).
func safeMoveCount(g *GameSim, s *SimSnake) int {
	count := 0
	head := s.Head()
	for _, d := range AllDirections {
		next := head.Move(d)
		if next.X < 0 || next.X >= g.Width || next.Y < 0 || next.Y >= g.Height {
			continue
		}
		hit := false
		for j := range g.Snakes {
			s2 := &g.Snakes[j]
			if !s2.IsAlive() {
				continue
			}
			for seg := 1; seg < len(s2.Body); seg++ {
				if next == s2.Body[seg] {
					hit = true
					break
				}
			}
			if hit {
				break
			}
		}
		if !hit {
			count++
		}
	}
	return count
}

// Evaluate scores a GameSim position from myID's perspective.
// Returns -1000 if we're dead, +1000 if all opponents are dead,
// otherwise Voronoi territory difference + length advantage +
// head-to-head pressure + opponent confinement + food urgency.
func Evaluate(g *GameSim, myID string) float64 {
	me := g.SnakeByID(myID)
	if me == nil || !me.IsAlive() {
		return -1000
	}

	// Check for alive opponents.
	allOppsDead := true
	for i := range g.Snakes {
		if g.Snakes[i].ID != myID && g.Snakes[i].IsAlive() {
			allOppsDead = false
			break
		}
	}
	if allOppsDead {
		return 1000
	}

	// Territory score (dominant factor).
	myTerritory, oppTerritory := VoronoiTerritory(g, myID)
	score := float64(myTerritory - oppTerritory)

	// Accumulate per-opponent scores.
	myHead := me.Head()
	wLen := 2.0
	wH2H := 5.0
	for i := range g.Snakes {
		opp := &g.Snakes[i]
		if opp.ID == myID || !opp.IsAlive() {
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

	// Food urgency: sharp scaling when health < 40.
	if me.Health < 40 && len(g.Food) > 0 {
		head := me.Head()
		minDist := 999
		for _, f := range g.Food {
			d := abs(head.X-f.X) + abs(head.Y-f.Y)
			if d < minDist {
				minDist = d
			}
		}
		foodWeight := float64(40-me.Health) * 0.5 // 0 at health=40, 20 at health=0
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
