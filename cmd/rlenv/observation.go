package main

import (
	"github.com/bodist/haruko/logic"
)

const (
	spatialChannels = 11
	spatialSize     = 21
	numScalars      = 8
	numActions      = 4
	centerOffset    = 10 // snake head placed at (10,10) in observation grid
)

// Channel indices for the spatial observation.
const (
	chWall       = 0  // Wall / out-of-bounds
	chOurBody    = 1  // Our body segments
	chOurHead    = 2  // Our head
	chOppBody    = 3  // Opponent body segments
	chOppHead    = 4  // Opponent head
	chFood       = 5  // Food
	chHazard     = 6  // Hazard
	chOurBodyAge = 7  // Our body age: 0.0 at head, 1.0 at tail (normalized)
	chOppBodyAge = 8  // Opponent body age: 0.0 at head, 1.0 at tail
	chOurStacked = 9  // Our body stacked (multiple segments on same cell)
	chOppStacked = 10 // Opponent body stacked
)

// computeObservation builds the head-centered spatial and scalar observation
// for the given snake index. Spatial is returned in CHW (channel-height-width)
// order, flattened into a single slice.
func computeObservation(g *logic.GameSim, myIdx int) (spatial []float32, scalars []float32) {
	spatial = make([]float32, spatialChannels*spatialSize*spatialSize)
	scalars = make([]float32, numScalars)

	me := &g.Snakes[myIdx]
	if !me.IsAlive() {
		// Return zeroed observation for dead snake.
		return
	}

	myHead := me.Head()

	// Helper to convert board coord to observation grid coord.
	toObs := func(c logic.Coord) (int, int, bool) {
		ox := c.X - myHead.X + centerOffset
		oy := c.Y - myHead.Y + centerOffset
		if ox < 0 || ox >= spatialSize || oy < 0 || oy >= spatialSize {
			return 0, 0, false
		}
		return ox, oy, true
	}

	// Helper to set a value in the CHW spatial tensor.
	// Layout: channel * H * W + y * W + x
	set := func(ch, ox, oy int, val float32) {
		spatial[ch*spatialSize*spatialSize+oy*spatialSize+ox] = val
	}

	// Fill wall channel: cells outside the board within the 21x21 view.
	for oy := 0; oy < spatialSize; oy++ {
		for ox := 0; ox < spatialSize; ox++ {
			boardX := ox - centerOffset + myHead.X
			boardY := oy - centerOffset + myHead.Y
			if boardX < 0 || boardX >= g.Width || boardY < 0 || boardY >= g.Height {
				set(chWall, ox, oy, 1.0)
			}
		}
	}

	// Our body.
	bodyLen := len(me.Body)
	bodyCounts := make(map[[2]int]int) // count segments per obs cell for stacking
	for seg := 0; seg < bodyLen; seg++ {
		ox, oy, ok := toObs(me.Body[seg])
		if !ok {
			continue
		}
		key := [2]int{ox, oy}
		bodyCounts[key]++

		if seg == 0 {
			set(chOurHead, ox, oy, 1.0)
		}
		set(chOurBody, ox, oy, 1.0)

		// Body age: 0.0 at head, 1.0 at tail.
		var age float32
		if bodyLen > 1 {
			age = float32(seg) / float32(bodyLen-1)
		}
		// Keep the oldest (largest) age if multiple segments overlap.
		cur := spatial[chOurBodyAge*spatialSize*spatialSize+oy*spatialSize+ox]
		if age > cur {
			set(chOurBodyAge, ox, oy, age)
		}
	}
	// Mark stacked cells.
	for key, count := range bodyCounts {
		if count > 1 {
			set(chOurStacked, key[0], key[1], 1.0)
		}
	}

	// Opponent snakes.
	for i := range g.Snakes {
		if i == myIdx || !g.Snakes[i].IsAlive() {
			continue
		}
		opp := &g.Snakes[i]
		oppBodyLen := len(opp.Body)
		oppCounts := make(map[[2]int]int)
		for seg := 0; seg < oppBodyLen; seg++ {
			ox, oy, ok := toObs(opp.Body[seg])
			if !ok {
				continue
			}
			key := [2]int{ox, oy}
			oppCounts[key]++

			if seg == 0 {
				set(chOppHead, ox, oy, 1.0)
			}
			set(chOppBody, ox, oy, 1.0)

			var age float32
			if oppBodyLen > 1 {
				age = float32(seg) / float32(oppBodyLen-1)
			}
			cur := spatial[chOppBodyAge*spatialSize*spatialSize+oy*spatialSize+ox]
			if age > cur {
				set(chOppBodyAge, ox, oy, age)
			}
		}
		for key, count := range oppCounts {
			if count > 1 {
				set(chOppStacked, key[0], key[1], 1.0)
			}
		}
	}

	// Food.
	for _, f := range g.Food {
		ox, oy, ok := toObs(f)
		if ok {
			set(chFood, ox, oy, 1.0)
		}
	}

	// Hazards.
	for _, h := range g.Hazards {
		ox, oy, ok := toObs(h)
		if ok {
			set(chHazard, ox, oy, 1.0)
		}
	}

	// Scalar features.
	scalars[0] = float32(me.Health) / 100.0
	scalars[1] = float32(me.Length) / 30.0
	scalars[4] = float32(g.Turn) / 500.0
	scalars[5] = float32(g.Width) / 19.0
	scalars[6] = float32(g.Height) / 19.0
	scalars[7] = float32(len(g.Food)) / 10.0

	// Opponent scalars: use the first alive opponent.
	for i := range g.Snakes {
		if i == myIdx || !g.Snakes[i].IsAlive() {
			continue
		}
		scalars[2] = float32(g.Snakes[i].Health) / 100.0
		scalars[3] = float32(g.Snakes[i].Length) / 30.0
		break
	}

	return
}

// oppositeDir returns the 180-degree reversal of a direction.
func oppositeDir(d logic.Direction) logic.Direction {
	switch d {
	case logic.Up:
		return logic.Down
	case logic.Down:
		return logic.Up
	case logic.Left:
		return logic.Right
	case logic.Right:
		return logic.Left
	}
	return d
}

// facingDirection returns the direction the snake is currently facing
// (from body[1] toward body[0], i.e., the head).
func facingDirection(s *logic.SimSnake) (logic.Direction, bool) {
	if len(s.Body) < 2 {
		return logic.Up, false
	}
	head := s.Body[0]
	neck := s.Body[1]
	dx := head.X - neck.X
	dy := head.Y - neck.Y
	switch {
	case dy == 1:
		return logic.Up, true
	case dy == -1:
		return logic.Down, true
	case dx == -1:
		return logic.Left, true
	case dx == 1:
		return logic.Right, true
	}
	// Head and neck overlap (stacked) — no clear facing.
	return logic.Up, false
}

// correctAction returns the action if it's valid, otherwise picks the first
// safe direction. This prevents instant suicide from untrained policies.
func correctAction(g *logic.GameSim, myIdx int, action logic.Direction) logic.Direction {
	mask := computeActionMask(g, myIdx)
	if mask[action] {
		return action
	}
	// Pick first valid action from the mask.
	for d := logic.Direction(0); d < 4; d++ {
		if mask[d] {
			return d
		}
	}
	// No valid actions at all — just return the original.
	return action
}

// computeActionMask returns a mask of valid actions (true = allowed).
// Always masks the 180-degree reversal of the current facing direction,
// and uses isSafeDir for wall+body collision checks.
func computeActionMask(g *logic.GameSim, myIdx int) [4]bool {
	me := &g.Snakes[myIdx]
	if !me.IsAlive() {
		// Dead snake: no valid actions, but return all true to avoid masking everything.
		return [4]bool{true, true, true, true}
	}

	var mask [4]bool
	facing, hasFacing := facingDirection(me)
	var reversal logic.Direction
	if hasFacing {
		reversal = oppositeDir(facing)
	}

	for _, d := range logic.AllDirections {
		// Always mask 180-degree reversal (instant death by self-collision).
		if hasFacing && d == reversal {
			continue
		}
		if logic.IsSafeDir(g, me, d) {
			mask[d] = true
		}
	}

	// If no moves are safe (all masked), allow everything except reversal
	// so the agent can at least choose.
	anyTrue := false
	for _, m := range mask {
		if m {
			anyTrue = true
			break
		}
	}
	if !anyTrue {
		for _, d := range logic.AllDirections {
			if hasFacing && d == reversal {
				continue
			}
			mask[d] = true
		}
	}

	return mask
}
