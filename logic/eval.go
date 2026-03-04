package logic

// Evaluate scores a GameSim position from myID's perspective.
// Returns -1000 if we're dead, +1000 if all opponents are dead,
// otherwise Voronoi territory difference + length advantage +
// head-to-head pressure + opponent confinement + food urgency.
func Evaluate(g *GameSim, myID string) float64 {
	me := g.SnakeByID(myID)
	if me == nil || !me.IsAlive() {
		return -1000
	}

	// Find alive opponent (2-snake game assumption).
	var opp *SimSnake
	allOppsDead := true
	for i := range g.Snakes {
		s := &g.Snakes[i]
		if s.ID != myID && s.IsAlive() {
			allOppsDead = false
			if opp == nil {
				opp = s
			}
		}
	}
	if allOppsDead {
		return 1000
	}

	// Territory score (dominant factor).
	myTerritory, oppTerritory := VoronoiTerritory(g, myID)
	score := float64(myTerritory - oppTerritory)

	if opp != nil {
		// Length advantage: longer snake wins head-to-head collisions.
		wLen := 2.0
		score += wLen * float64(me.Length-opp.Length)

		// Head-to-head pressure: bonus when longer and near opponent head.
		myHead := me.Head()
		oppHead := opp.Head()
		dist := abs(myHead.X-oppHead.X) + abs(myHead.Y-oppHead.Y)
		if dist <= 2 {
			wH2H := 5.0
			if me.Length > opp.Length {
				score += wH2H
			} else if me.Length < opp.Length {
				score -= wH2H
			}
		}

		// Opponent confinement: fewer safe moves = more trapped.
		safeMoves := 0
		for _, d := range AllDirections {
			next := opp.Head().Move(d)
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
				safeMoves++
			}
		}
		switch safeMoves {
		case 0:
			score += 50.0 // opponent trapped — forced kill
		case 1:
			score += 15.0 // opponent nearly trapped
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
