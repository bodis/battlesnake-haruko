package logic

// Evaluate scores a GameSim position from myID's perspective.
// Returns -1000 if we're dead, +1000 if all opponents are dead,
// otherwise the flood-fill reachable cell count.
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

	space := float64(floodFillSim(g, myID))

	// Food urgency: when health is low, bias toward moves closer to food.
	if len(g.Food) > 0 {
		dist := NearestFoodDistance(me.Head(), g.Food)
		if dist >= 0 {
			urgency := float64(100-me.Health) * 0.15
			space += urgency / float64(max(dist, 1))
		}
	}

	return space
}

// floodFillSim counts reachable cells from myID's head using BFS.
// Blocked: interior body segments (index 1..len-2) of any alive snake,
//          and enemy heads (Body[0] for non-myID alive snakes).
// Passable: tails (Body[len-1]) of each alive snake, empty cells, food, hazards.
func floodFillSim(g *GameSim, myID string) int {
	me := g.SnakeByID(myID)
	if me == nil {
		return 0
	}
	start := me.Head()
	if start.X < 0 || start.X >= g.Width || start.Y < 0 || start.Y >= g.Height {
		return 0
	}

	// Build blocked set: interior body segments + enemy heads.
	blocked := make(map[Coord]bool)
	for i := range g.Snakes {
		s := &g.Snakes[i]
		if !s.IsAlive() {
			continue
		}
		// Interior body segments (exclude head[0] and tail[len-1]).
		end := len(s.Body) - 1
		for seg := 1; seg < end; seg++ {
			blocked[s.Body[seg]] = true
		}
		// Enemy heads are also blocked.
		if s.ID != myID {
			blocked[s.Body[0]] = true
		}
	}

	visited := make(map[Coord]bool, g.Width*g.Height)
	queue := []Coord{start}
	visited[start] = true
	count := 0

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		count++

		for _, d := range AllDirections {
			next := cur.Move(d)
			if next.X < 0 || next.X >= g.Width || next.Y < 0 || next.Y >= g.Height {
				continue
			}
			if visited[next] || blocked[next] {
				continue
			}
			visited[next] = true
			queue = append(queue, next)
		}
	}
	return count
}
