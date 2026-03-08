package logic

// EscapeReachability counts cells reachable from the given snake's head
// within maxDist BFS steps. Unlike Voronoi (multi-source BFS from all heads),
// this measures local reachability for a single snake — how much space
// the snake can actually reach in N moves. Body segments block, tails are
// passable (matching Voronoi semantics).
//
// This is NOT on the hot path — called only during trace recording.
func EscapeReachability(g *GameSim, snakeIdx, maxDist int) int {
	s := &g.Snakes[snakeIdx]
	if !s.IsAlive() {
		return 0
	}

	size := g.Width * g.Height
	visited := make([]bool, size)
	blocked := make([]bool, size)

	// Mark interior body segments as blocked (same as Voronoi).
	for i := range g.Snakes {
		s2 := &g.Snakes[i]
		if !s2.IsAlive() {
			continue
		}
		end := len(s2.Body) - 1 // tail passable
		for seg := 1; seg < end; seg++ {
			c := s2.Body[seg]
			if c.X >= 0 && c.X < g.Width && c.Y >= 0 && c.Y < g.Height {
				blocked[c.Y*g.Width+c.X] = true
			}
		}
	}

	type entry struct {
		x, y, dist int
	}
	head := s.Head()
	queue := make([]entry, 0, size)
	headIdx := head.Y*g.Width + head.X
	visited[headIdx] = true
	queue = append(queue, entry{head.X, head.Y, 0})
	count := 0

	for qi := 0; qi < len(queue); qi++ {
		cur := queue[qi]
		count++
		if cur.dist >= maxDist {
			continue
		}
		// Expand 4 directions.
		for _, dd := range [4][2]int{{0, 1}, {0, -1}, {-1, 0}, {1, 0}} {
			nx, ny := cur.x+dd[0], cur.y+dd[1]
			if nx < 0 || nx >= g.Width || ny < 0 || ny >= g.Height {
				continue
			}
			ni := ny*g.Width + nx
			if visited[ni] || blocked[ni] {
				continue
			}
			visited[ni] = true
			queue = append(queue, entry{nx, ny, cur.dist + 1})
		}
	}
	return count
}
