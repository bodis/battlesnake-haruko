package logic

// Evaluate scores a GameSim position from myID's perspective.
// Returns -1000 if we're dead, +1000 if all opponents are dead,
// otherwise Voronoi territory difference + food urgency.
func Evaluate(g *GameSim, myID string) float64 {
	me := g.SnakeByID(myID)
	if me == nil || !me.IsAlive() {
		return -1000
	}

	allOppsDead := true
	for i := range g.Snakes {
		s := &g.Snakes[i]
		if s.ID != myID && s.IsAlive() {
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
