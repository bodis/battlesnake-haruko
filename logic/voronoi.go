package logic

// VoronoiTerritory performs a multi-source BFS from all alive snake heads
// and returns the number of cells claimed by myID and by all opponents combined.
// Cells reached by two snakes in the same BFS layer are unclaimed (ties).
func VoronoiTerritory(g *GameSim, myID string) (myCount, oppCount int) {
	size := g.Width * g.Height

	// owner[cell] == 0: unclaimed, positive: snake index+1, -1: tied
	owner := make([]int8, size)
	dist := make([]int16, size)
	for i := range dist {
		dist[i] = -1
	}

	// blocked: interior body segments (index 1..len-2) of alive snakes.
	blocked := make([]bool, size)
	for i := range g.Snakes {
		s := &g.Snakes[i]
		if !s.IsAlive() {
			continue
		}
		end := len(s.Body) - 1 // tail index (passable)
		for seg := 1; seg < end; seg++ {
			c := s.Body[seg]
			if c.X >= 0 && c.X < g.Width && c.Y >= 0 && c.Y < g.Height {
				blocked[c.Y*g.Width+c.X] = true
			}
		}
	}

	// Seed queue with heads of alive snakes.
	type entry struct {
		x, y int
	}
	var queue []entry

	for i := range g.Snakes {
		s := &g.Snakes[i]
		if !s.IsAlive() {
			continue
		}
		head := s.Head()
		if head.X < 0 || head.X >= g.Width || head.Y < 0 || head.Y >= g.Height {
			continue
		}
		idx := head.Y*g.Width + head.X
		tag := int8(i + 1)
		if dist[idx] == -1 {
			// First snake to seed this cell.
			dist[idx] = 0
			owner[idx] = tag
			queue = append(queue, entry{head.X, head.Y})
		} else if dist[idx] == 0 {
			// Two heads on same cell at distance 0 → tie.
			owner[idx] = -1
		}
	}

	// BFS expansion.
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		ci := cur.y*g.Width + cur.x
		curDist := dist[ci]
		curOwner := owner[ci]

		if curOwner == -1 {
			// Tied cell — don't expand from it.
			continue
		}

		pos := Coord{cur.x, cur.y}
		for _, d := range AllDirections {
			next := pos.Move(d)
			if next.X < 0 || next.X >= g.Width || next.Y < 0 || next.Y >= g.Height {
				continue
			}
			ni := next.Y*g.Width + next.X
			if blocked[ni] {
				continue
			}
			nd := curDist + 1
			if dist[ni] == -1 {
				// Unclaimed — claim it.
				dist[ni] = nd
				owner[ni] = curOwner
				queue = append(queue, entry{next.X, next.Y})
			} else if dist[ni] == nd && owner[ni] != curOwner && owner[ni] != -1 {
				// Same distance, different owner → tie.
				owner[ni] = -1
			}
		}
	}

	// Count territory.
	myTag := int8(-1)
	for i := range g.Snakes {
		if g.Snakes[i].ID == myID {
			myTag = int8(i + 1)
			break
		}
	}

	for i := 0; i < size; i++ {
		o := owner[i]
		if o <= 0 {
			continue
		}
		if o == myTag {
			myCount++
		} else {
			oppCount++
		}
	}
	return
}
