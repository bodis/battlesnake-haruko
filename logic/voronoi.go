package logic

import "sync"

// voronoiWorkspace holds pre-allocated arrays for VoronoiTerritory BFS.
type voronoiWorkspace struct {
	owner   [maxBoardCells]int8
	dist    [maxBoardCells]int16
	blocked [maxBoardCells]bool
	queue   []voronoiEntry

	// Tarjan's articulation point detection (iterative DFS).
	disc        [maxBoardCells]int16
	low         [maxBoardCells]int16
	subtree     [maxBoardCells]int16
	apCut       [maxBoardCells]int16
	isAP        [maxBoardCells]bool
	dfsStack    [maxBoardCells]tarjanFrame
	tarjanDirty [maxBoardCells]int16 // list of cells touched by Tarjan (for fast cleanup)
	tarjanCount int                  // number of dirty cells

	// Head-side flood fill (bottleneck routing).
	headQueue      [maxBoardCells]int16
	headFillDirty  [maxBoardCells]int16 // cells visited by head-side BFS (for subtree cleanup)
	headFillCount  int
}

type voronoiEntry struct {
	x, y int
}

type tarjanFrame struct {
	cell     int16
	parent   int16
	dirIdx   int8
	children int8
}

var voronoiPool = sync.Pool{
	New: func() any {
		ws := &voronoiWorkspace{
			queue: make([]voronoiEntry, 0, maxBoardCells),
		}
		// Initialize disc to -1 (unvisited) for Tarjan's.
		for i := range ws.disc {
			ws.disc[i] = -1
		}
		return ws
	},
}

// VoronoiResult holds enriched territory data from multi-source BFS.
type VoronoiResult struct {
	MyTerritory   int
	OppTerritory  int
	MyFood        int  // food cells in our Voronoi territory
	OppFood       int  // food cells in opponent territory
	IsPartitioned bool // our wavefront never met any opponent wavefront

	// Food quality (distance-weighted)
	MyFoodValue float64 // sum of 1.0/dist for each food we own

	// Territory quality (connectivity: avg same-owner neighbors per cell)
	MyConnectivity  float64 // higher = wider territory, lower = corridors
	OppConnectivity float64

	// Diagnostic: territory depth profile (BFS distance from head)
	MyNearTerritory  int // cells at BFS dist ≤ 3 (immediate surroundings)
	MyFarTerritory   int // cells at BFS dist > 6 (deep territory)
	OppNearTerritory int
	OppFarTerritory  int

	// Diagnostic: territory shape quality
	MyCorridorCells  int // cells with ≤ 1 same-owner neighbor (dead-ends, narrow passages)
	OppCorridorCells int

	// Bottleneck detection (articulation points in territory subgraph)
	MyThreatenedTerritory  int // cells behind live APs in our territory
	OppThreatenedTerritory int // cells behind live APs in opponent territory

	// Bottleneck routing: cells reachable from head without crossing APs.
	// 0 when skipBottleneck=true or no APs exist.
	HeadSideRegion int
}

// findThreatenedTerritory runs iterative Tarjan's algorithm on the territory
// subgraph for the given tag. Returns the total cells behind "live" articulation
// points (APs adjacent to non-owned cells). width is the board width.
func (ws *voronoiWorkspace) findThreatenedTerritory(tag int8, rootCell int16, territorySize, size, width int) int {
	if territorySize < 8 || rootCell < 0 {
		return 0
	}
	root := rootCell
	height := size / width

	// Clean up cells from previous Tarjan run (avoids full-array clear).
	for i := 0; i < ws.tarjanCount; i++ {
		c := ws.tarjanDirty[i]
		ws.disc[c] = -1
		ws.isAP[c] = false
		ws.apCut[c] = 0
	}
	ws.tarjanCount = 0

	// Mark root and track as dirty.
	ws.disc[root] = 0
	ws.low[root] = 0
	ws.subtree[root] = 1
	ws.tarjanDirty[0] = root
	ws.tarjanCount = 1

	// Iterative Tarjan's DFS.
	timer := int16(1)
	sp := 0 // stack pointer
	ws.dfsStack[0] = tarjanFrame{cell: root, parent: -1, dirIdx: 0, children: 0}

	for sp >= 0 {
		frame := &ws.dfsStack[sp]
		cell := frame.cell
		cx := int(cell) % width
		cy := int(cell) / width

		// Try next neighbor direction.
		advanced := false
		for frame.dirIdx < 4 {
			d := frame.dirIdx
			frame.dirIdx++

			var nx, ny int
			switch d {
			case 0:
				nx, ny = cx, cy+1
			case 1:
				nx, ny = cx, cy-1
			case 2:
				nx, ny = cx-1, cy
			case 3:
				nx, ny = cx+1, cy
			}

			if nx < 0 || nx >= width || ny < 0 || ny >= height {
				continue
			}
			ni := int16(ny*width + nx)
			if ws.owner[ni] != tag {
				continue
			}

			if ws.disc[ni] == -1 {
				// Tree edge: push child.
				ws.disc[ni] = timer
				ws.low[ni] = timer
				ws.subtree[ni] = 1
				ws.tarjanDirty[ws.tarjanCount] = ni
				ws.tarjanCount++
				timer++
				sp++
				ws.dfsStack[sp] = tarjanFrame{cell: ni, parent: cell, dirIdx: 0, children: 0}
				advanced = true
				break
			} else if ni != frame.parent {
				// Back edge: update low.
				if ws.disc[ni] < ws.low[cell] {
					ws.low[cell] = ws.disc[ni]
				}
			}
		}

		if advanced {
			continue
		}

		// Pop: propagate to parent.
		if sp > 0 {
			parent := &ws.dfsStack[sp-1]
			parentCell := parent.cell

			// Propagate low.
			if ws.low[cell] < ws.low[parentCell] {
				ws.low[parentCell] = ws.low[cell]
			}

			// Accumulate subtree size.
			ws.subtree[parentCell] += ws.subtree[cell]

			// Check AP condition for non-root.
			parent.children++
			if ws.low[cell] >= ws.disc[parentCell] {
				if parentCell != root {
					ws.isAP[parentCell] = true
					if ws.subtree[cell] > ws.apCut[parentCell] {
						ws.apCut[parentCell] = ws.subtree[cell]
					}
				}
			}
		}

		// Root AP check: 2+ DFS children.
		if cell == root && frame.children >= 2 {
			ws.isAP[root] = true
		}

		sp--
	}

	// Sum threatened territory from live APs (adjacent to non-owned cell).
	threatened := 0
	for i := 0; i < ws.tarjanCount; i++ {
		ci := ws.tarjanDirty[i]
		if !ws.isAP[ci] {
			continue
		}
		cx := int(ci) % width
		cy := int(ci) / width
		live := false
		for d := 0; d < 4; d++ {
			var nx, ny int
			switch d {
			case 0:
				nx, ny = cx, cy+1
			case 1:
				nx, ny = cx, cy-1
			case 2:
				nx, ny = cx-1, cy
			case 3:
				nx, ny = cx+1, cy
			}
			if nx < 0 || nx >= width || ny < 0 || ny >= height {
				continue
			}
			ni := ny*width + nx
			if ws.owner[ni] != tag {
				live = true
				break
			}
		}
		if live {
			threatened += int(ws.apCut[ci])
		}
	}

	return threatened
}

// headSideFloodFill BFS-floods from headCell through cells owned by tag that
// are NOT articulation points. Returns the count of reachable cells (the
// "head side" of any bottleneck). Uses ws.subtree[] as visited marker (safe to
// reuse after Tarjan's completes) and ws.headQueue[] as BFS queue.
func (ws *voronoiWorkspace) headSideFloodFill(tag int8, headCell int16, size, width int) int {
	height := size / width

	// Clean up subtree markers from previous head-side BFS.
	for i := 0; i < ws.headFillCount; i++ {
		ws.subtree[ws.headFillDirty[i]] = 0
	}
	ws.headFillCount = 0

	// Seed with head cell (always included, even if it's an AP).
	ws.subtree[headCell] = -1
	ws.headQueue[0] = headCell
	ws.headFillDirty[0] = headCell
	ws.headFillCount = 1
	qLen := 1
	count := 1

	for qi := 0; qi < qLen; qi++ {
		cell := ws.headQueue[qi]
		cx := int(cell) % width
		cy := int(cell) / width

		for d := 0; d < 4; d++ {
			var nx, ny int
			switch d {
			case 0:
				nx, ny = cx, cy+1
			case 1:
				nx, ny = cx, cy-1
			case 2:
				nx, ny = cx-1, cy
			case 3:
				nx, ny = cx+1, cy
			}
			if nx < 0 || nx >= width || ny < 0 || ny >= height {
				continue
			}
			ni := int16(ny*width + nx)
			if ws.owner[ni] != tag || ws.isAP[ni] || ws.subtree[ni] == -1 {
				continue
			}
			ws.subtree[ni] = -1
			ws.headQueue[qLen] = ni
			ws.headFillDirty[ws.headFillCount] = ni
			ws.headFillCount++
			qLen++
			count++
		}
	}

	return count
}

// escapeWorkspace holds pre-allocated arrays for EscapeReachabilityPooled BFS.
type escapeWorkspace struct {
	blocked [maxBoardCells]bool
	visited [maxBoardCells]bool
	queue   [maxBoardCells]struct{ x, y int16 }
}

var escapePool = sync.Pool{
	New: func() any { return &escapeWorkspace{} },
}

// EscapeReachabilityPooled counts cells reachable from a snake's head within
// maxDist BFS steps. Zero-alloc (uses pooled workspace). Body segments block,
// tails are passable (matching Voronoi semantics).
func EscapeReachabilityPooled(g *GameSim, snakeIdx, maxDist int) int {
	s := &g.Snakes[snakeIdx]
	if !s.IsAlive() {
		return 0
	}

	ws := escapePool.Get().(*escapeWorkspace)

	size := g.Width * g.Height
	for i := 0; i < size; i++ {
		ws.blocked[i] = false
		ws.visited[i] = false
	}

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
				ws.blocked[c.Y*g.Width+c.X] = true
			}
		}
	}

	head := s.Head()
	if head.X < 0 || head.X >= g.Width || head.Y < 0 || head.Y >= g.Height {
		escapePool.Put(ws)
		return 0
	}

	ws.visited[head.Y*g.Width+head.X] = true
	ws.queue[0] = struct{ x, y int16 }{int16(head.X), int16(head.Y)}
	qLen := 1
	count := 1
	layerStart := 0

	for d := 0; d < maxDist; d++ {
		layerEnd := qLen
		for qi := layerStart; qi < layerEnd; qi++ {
			cur := ws.queue[qi]
			cx, cy := int(cur.x), int(cur.y)
			for _, dd := range [4][2]int{{0, 1}, {0, -1}, {-1, 0}, {1, 0}} {
				nx, ny := cx+dd[0], cy+dd[1]
				if nx < 0 || nx >= g.Width || ny < 0 || ny >= g.Height {
					continue
				}
				ni := ny*g.Width + nx
				if ws.visited[ni] || ws.blocked[ni] {
					continue
				}
				ws.visited[ni] = true
				ws.queue[qLen] = struct{ x, y int16 }{int16(nx), int16(ny)}
				qLen++
				count++
			}
		}
		layerStart = layerEnd
	}

	escapePool.Put(ws)
	return count
}

// VoronoiTerritory performs a multi-source BFS from all alive snake heads
// and returns territory counts, food ownership, and partition status.
// Cells reached by two snakes in the same BFS layer are unclaimed (ties).
func VoronoiTerritory(g *GameSim, myIdx int, skipBottleneck bool) VoronoiResult {
	size := g.Width * g.Height

	ws := voronoiPool.Get().(*voronoiWorkspace)
	defer voronoiPool.Put(ws)

	// Clear workspace arrays for the board size.
	for i := 0; i < size; i++ {
		ws.owner[i] = 0
		ws.dist[i] = -1
		ws.blocked[i] = false
	}
	ws.queue = ws.queue[:0]

	// blocked: interior body segments (index 1..len-2) of alive snakes.
	aliveCount := 0
	for i := range g.Snakes {
		s := &g.Snakes[i]
		if !s.IsAlive() {
			continue
		}
		aliveCount++
		end := len(s.Body) - 1 // tail index (passable)
		for seg := 1; seg < end; seg++ {
			c := s.Body[seg]
			if c.X >= 0 && c.X < g.Width && c.Y >= 0 && c.Y < g.Height {
				ws.blocked[c.Y*g.Width+c.X] = true
			}
		}
	}

	// Seed queue with heads of alive snakes.
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
		if ws.dist[idx] == -1 {
			ws.dist[idx] = 0
			ws.owner[idx] = tag
			ws.queue = append(ws.queue, voronoiEntry{head.X, head.Y})
		} else if ws.dist[idx] == 0 {
			ws.owner[idx] = -1
		}
	}

	// BFS expansion.
	myTag := int8(myIdx + 1)
	myHasFrontier := false

	for qi := 0; qi < len(ws.queue); qi++ {
		cur := ws.queue[qi]
		ci := cur.y*g.Width + cur.x
		curDist := ws.dist[ci]
		curOwner := ws.owner[ci]

		if curOwner == -1 {
			continue
		}

		pos := Coord{cur.x, cur.y}
		for _, d := range AllDirections {
			next := pos.Move(d)
			if next.X < 0 || next.X >= g.Width || next.Y < 0 || next.Y >= g.Height {
				continue
			}
			ni := next.Y*g.Width + next.X
			if ws.blocked[ni] {
				continue
			}
			nd := curDist + 1
			if ws.dist[ni] == -1 {
				ws.dist[ni] = nd
				ws.owner[ni] = curOwner
				ws.queue = append(ws.queue, voronoiEntry{next.X, next.Y})
			} else if ws.dist[ni] == nd && ws.owner[ni] != curOwner && ws.owner[ni] != -1 {
				if ws.owner[ni] == myTag || curOwner == myTag {
					myHasFrontier = true
				}
				ws.owner[ni] = -1
			}
		}
	}

	// Count territory, connectivity, depth profile, and corridor cells.
	var result VoronoiResult
	var myNeighborSum, oppNeighborSum int
	for i := 0; i < size; i++ {
		o := ws.owner[i]
		if o <= 0 {
			continue
		}
		x := i % g.Width
		y := i / g.Width
		neighbors := 0
		if x > 0 && ws.owner[i-1] == o {
			neighbors++
		}
		if x < g.Width-1 && ws.owner[i+1] == o {
			neighbors++
		}
		if y > 0 && ws.owner[i-g.Width] == o {
			neighbors++
		}
		if y < g.Height-1 && ws.owner[i+g.Width] == o {
			neighbors++
		}
		d := ws.dist[i]
		if o == myTag {
			result.MyTerritory++
			myNeighborSum += neighbors
			if d <= 3 {
				result.MyNearTerritory++
			}
			if d > 6 {
				result.MyFarTerritory++
			}
			if neighbors <= 1 {
				result.MyCorridorCells++
			}
		} else {
			result.OppTerritory++
			oppNeighborSum += neighbors
			if d <= 3 {
				result.OppNearTerritory++
			}
			if d > 6 {
				result.OppFarTerritory++
			}
			if neighbors <= 1 {
				result.OppCorridorCells++
			}
		}
	}
	if result.MyTerritory > 0 {
		result.MyConnectivity = float64(myNeighborSum) / float64(result.MyTerritory)
	}
	if result.OppTerritory > 0 {
		result.OppConnectivity = float64(oppNeighborSum) / float64(result.OppTerritory)
	}

	// Count food ownership.
	for _, f := range g.Food {
		fi := f.Y*g.Width + f.X
		if fi < 0 || fi >= size {
			continue
		}
		o := ws.owner[fi]
		d := ws.dist[fi]
		switch o {
		case myTag:
			result.MyFood++
			if d > 0 {
				result.MyFoodValue += 1.0 / float64(d)
			}
		default:
			if o > 0 {
				result.OppFood++
			}
		}
	}

	// Bottleneck detection: find threatened territory behind articulation points.
	if !skipBottleneck {
		me := &g.Snakes[myIdx]
		myHead := me.Head()
		myRootCell := int16(myHead.Y*g.Width + myHead.X)
		result.MyThreatenedTerritory = ws.findThreatenedTerritory(myTag, myRootCell, result.MyTerritory, size, g.Width)
		result.HeadSideRegion = ws.headSideFloodFill(myTag, myRootCell, size, g.Width)
		// Find first alive opponent for opponent bottleneck.
		for i := range g.Snakes {
			if i != myIdx && g.Snakes[i].IsAlive() {
				oppTag := int8(i + 1)
				oppHead := g.Snakes[i].Head()
				oppRootCell := int16(oppHead.Y*g.Width + oppHead.X)
				result.OppThreatenedTerritory = ws.findThreatenedTerritory(oppTag, oppRootCell, result.OppTerritory, size, g.Width)
				break
			}
		}
	}

	// Partition: our wavefront never met any opponent.
	result.IsPartitioned = !myHasFrontier && aliveCount >= 2

	return result
}
