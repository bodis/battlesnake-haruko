package logic

import "sync"

// voronoiWorkspace holds pre-allocated arrays for VoronoiTerritory BFS.
type voronoiWorkspace struct {
	owner   [maxBoardCells]int8
	dist    [maxBoardCells]int16
	blocked [maxBoardCells]bool
	queue   []voronoiEntry
}

type voronoiEntry struct {
	x, y int
}

var voronoiPool = sync.Pool{
	New: func() any {
		return &voronoiWorkspace{
			queue: make([]voronoiEntry, 0, maxBoardCells),
		}
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
	MyClosestFoodDist  int     // BFS dist to nearest food in our territory (0 = no food)
	OppClosestFoodDist int     // opponent's nearest food distance (0 = no food)
	MyFoodValue        float64 // sum of 1.0/dist for each food we own

	// Territory shape
	MyTerritoryDepth int // max BFS distance among our territory cells

	// Positional (centroids)
	MyCenterX, MyCenterY   float64
	OppCenterX, OppCenterY float64

	// Tail reachability
	MyTailReachable bool // true if our tail cell is in our Voronoi territory
}

// VoronoiTerritory performs a multi-source BFS from all alive snake heads
// and returns territory counts, food ownership, and partition status.
// Cells reached by two snakes in the same BFS layer are unclaimed (ties).
func VoronoiTerritory(g *GameSim, myIdx int) VoronoiResult {
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

	// Count territory with centroid and depth tracking.
	var result VoronoiResult
	var mySumX, mySumY, oppSumX, oppSumY int
	var myMaxDist int16
	x, y := 0, 0
	for i := 0; i < size; i++ {
		o := ws.owner[i]
		if o > 0 {
			if o == myTag {
				result.MyTerritory++
				mySumX += x
				mySumY += y
				if d := ws.dist[i]; d > myMaxDist {
					myMaxDist = d
				}
			} else {
				result.OppTerritory++
				oppSumX += x
				oppSumY += y
			}
		}
		x++
		if x == g.Width {
			x = 0
			y++
		}
	}
	if result.MyTerritory > 0 {
		result.MyCenterX = float64(mySumX) / float64(result.MyTerritory)
		result.MyCenterY = float64(mySumY) / float64(result.MyTerritory)
	}
	if result.OppTerritory > 0 {
		result.OppCenterX = float64(oppSumX) / float64(result.OppTerritory)
		result.OppCenterY = float64(oppSumY) / float64(result.OppTerritory)
	}
	result.MyTerritoryDepth = int(myMaxDist)

	// Count food ownership with distance tracking.
	var myClosest, oppClosest int16
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
			if myClosest == 0 || d < myClosest {
				myClosest = d
			}
		default:
			if o > 0 {
				result.OppFood++
				if oppClosest == 0 || d < oppClosest {
					oppClosest = d
				}
			}
		}
	}
	result.MyClosestFoodDist = int(myClosest)
	result.OppClosestFoodDist = int(oppClosest)

	// Tail reachability.
	me := &g.Snakes[myIdx]
	if me.IsAlive() {
		tail := me.Tail()
		if tail.X >= 0 && tail.X < g.Width && tail.Y >= 0 && tail.Y < g.Height {
			result.MyTailReachable = ws.owner[tail.Y*g.Width+tail.X] == myTag
		}
	}

	// Partition: our wavefront never met any opponent.
	result.IsPartitioned = !myHasFrontier && aliveCount >= 2

	return result
}
