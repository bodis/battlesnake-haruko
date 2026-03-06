package logic

import "sync"

// HazardDamage is the health penalty for standing on a hazard cell (standard rules).
const HazardDamage = 14

// SimSnake represents a snake in the game simulator with full state.
type SimSnake struct {
	ID              string
	Body            []Coord // head-first: Body[0] = head
	Health          int
	Length          int
	EliminatedCause string // "" = alive
}

// IsAlive reports whether the snake has not been eliminated.
func (s *SimSnake) IsAlive() bool { return s.EliminatedCause == "" }

// Head returns the snake's head coordinate.
func (s *SimSnake) Head() Coord { return s.Body[0] }

// Tail returns the snake's tail coordinate.
func (s *SimSnake) Tail() Coord { return s.Body[len(s.Body)-1] }

// GameSim holds the full game state for simulation and cloning.
type GameSim struct {
	Width, Height int
	Snakes        []SimSnake
	Food          []Coord
	Hazards       []Coord
	Turn          int
	poolRef       *pooledGameSim // back-reference for Release; nil if not pooled

	LastCompletedDepth int // set by BestMoveIterative after search
}

// cloneSnakes deep-copies a SimSnake slice so no body backing arrays are shared.
func cloneSnakes(src []SimSnake) []SimSnake {
	dst := make([]SimSnake, len(src))
	for i, s := range src {
		body := make([]Coord, len(s.Body))
		copy(body, s.Body)
		dst[i] = SimSnake{
			ID:              s.ID,
			Body:            body,
			Health:          s.Health,
			Length:          s.Length,
			EliminatedCause: s.EliminatedCause,
		}
	}
	return dst
}

// NewGameSim creates a GameSim with deep-copied slices.
func NewGameSim(width, height int, snakes []SimSnake, food, hazards []Coord) *GameSim {
	f := make([]Coord, len(food))
	copy(f, food)
	h := make([]Coord, len(hazards))
	copy(h, hazards)
	return &GameSim{
		Width:   width,
		Height:  height,
		Snakes:  cloneSnakes(snakes),
		Food:    f,
		Hazards: h,
	}
}

// Clone returns a deep copy of the GameSim with no shared backing arrays.
func (gs *GameSim) Clone() *GameSim {
	f := make([]Coord, len(gs.Food))
	copy(f, gs.Food)
	h := make([]Coord, len(gs.Hazards))
	copy(h, gs.Hazards)
	return &GameSim{
		Width:   gs.Width,
		Height:  gs.Height,
		Turn:    gs.Turn,
		Snakes:  cloneSnakes(gs.Snakes),
		Food:    f,
		Hazards: h,
	}
}

// --- sync.Pool for GameSim clones ---

const maxPoolBodyLen = 128 // covers typical snake lengths on boards up to 19x19

type pooledGameSim struct {
	gs      GameSim
	snakes  [MaxSnakes]SimSnake
	bodies  [MaxSnakes][maxPoolBodyLen]Coord
	food    [32]Coord
	hazards [maxBoardCells]Coord
}

var gameSimPool = sync.Pool{
	New: func() any { return &pooledGameSim{} },
}

// CloneFromPool returns a deep copy using pooled backing arrays.
// The caller MUST call Release() when done.
func (gs *GameSim) CloneFromPool() *GameSim {
	p := gameSimPool.Get().(*pooledGameSim)
	dst := &p.gs
	dst.Width = gs.Width
	dst.Height = gs.Height
	dst.Turn = gs.Turn
	dst.poolRef = p

	// Snakes
	n := len(gs.Snakes)
	if n > MaxSnakes {
		n = MaxSnakes
	}
	dst.Snakes = p.snakes[:n]
	for i := 0; i < n; i++ {
		src := &gs.Snakes[i]
		bodyLen := len(src.Body)
		var body []Coord
		if bodyLen <= maxPoolBodyLen {
			copy(p.bodies[i][:bodyLen], src.Body)
			body = p.bodies[i][:bodyLen]
		} else {
			body = make([]Coord, bodyLen)
			copy(body, src.Body)
		}
		dst.Snakes[i] = SimSnake{
			ID:              src.ID,
			Body:            body,
			Health:          src.Health,
			Length:          src.Length,
			EliminatedCause: src.EliminatedCause,
		}
	}

	// Food
	nFood := len(gs.Food)
	if nFood <= len(p.food) {
		copy(p.food[:nFood], gs.Food)
		dst.Food = p.food[:nFood]
	} else {
		dst.Food = make([]Coord, nFood)
		copy(dst.Food, gs.Food)
	}

	// Hazards
	nHaz := len(gs.Hazards)
	if nHaz <= len(p.hazards) {
		copy(p.hazards[:nHaz], gs.Hazards)
		dst.Hazards = p.hazards[:nHaz]
	} else {
		dst.Hazards = make([]Coord, nHaz)
		copy(dst.Hazards, gs.Hazards)
	}

	return dst
}

// Release returns a pooled GameSim to the pool. Safe to call on non-pooled instances (no-op).
func (gs *GameSim) Release() {
	if gs.poolRef != nil {
		gameSimPool.Put(gs.poolRef)
		gs.poolRef = nil
	}
}

// MoveSnakes applies the given moves to each alive snake.
// For each alive snake in the MoveSet, the head advances one step in the given
// direction and the tail is dropped (in-place shift, no allocation).
// Dead snakes and snakes not in the MoveSet are skipped.
func (gs *GameSim) MoveSnakes(moves MoveSet) {
	for i := range gs.Snakes {
		if i >= MaxSnakes {
			break
		}
		s := &gs.Snakes[i]
		if s.EliminatedCause != "" || !moves.Has[i] {
			continue
		}
		newHead := s.Body[0].Move(moves.Dir[i])
		copy(s.Body[1:], s.Body[:len(s.Body)-1])
		s.Body[0] = newHead
	}
}

// SnakeByID returns a pointer to the snake with the given ID, or nil.
func (gs *GameSim) SnakeByID(id string) *SimSnake {
	for i := range gs.Snakes {
		if gs.Snakes[i].ID == id {
			return &gs.Snakes[i]
		}
	}
	return nil
}

// IsOver reports whether the game is finished (fewer than 2 snakes alive).
func (gs *GameSim) IsOver() bool {
	alive := 0
	for i := range gs.Snakes {
		if gs.Snakes[i].IsAlive() {
			alive++
			if alive >= 2 {
				return false
			}
		}
	}
	return true
}

// Step executes a full turn following Battlesnake Standard rules.
// moves maps snake index → direction for each alive snake that should move.
func (gs *GameSim) Step(moves MoveSet) {
	n := len(gs.Snakes)
	if n > MaxSnakes {
		n = MaxSnakes
	}

	// Phase 0 — Save tails before movement (for growth).
	var savedTails [MaxSnakes]Coord
	for i := 0; i < n; i++ {
		s := &gs.Snakes[i]
		if s.IsAlive() && moves.Has[i] {
			savedTails[i] = s.Tail()
		}
	}

	// Phase 1 — Move snakes.
	gs.MoveSnakes(moves)

	// Phase 2 — Reduce health by 1.
	for i := 0; i < n; i++ {
		s := &gs.Snakes[i]
		if s.IsAlive() && moves.Has[i] {
			s.Health--
		}
	}

	// Phase 3 — Hazard damage.
	for i := 0; i < n; i++ {
		s := &gs.Snakes[i]
		if !s.IsAlive() || !moves.Has[i] {
			continue
		}
		head := s.Head()
		for _, h := range gs.Hazards {
			if head == h {
				s.Health -= HazardDamage
				break
			}
		}
	}

	// Phase 4 — Feed snakes.
	var eaten [maxBoardCells]bool
	for i := 0; i < n; i++ {
		s := &gs.Snakes[i]
		if !s.IsAlive() || !moves.Has[i] {
			continue
		}
		head := s.Head()
		for fi, f := range gs.Food {
			if head == f {
				s.Health = 100
				s.Length++
				s.Body = append(s.Body, savedTails[i])
				eaten[fi] = true
				break
			}
		}
	}
	// Remove eaten food (swap-and-truncate from back).
	for fi := len(gs.Food) - 1; fi >= 0; fi-- {
		if eaten[fi] {
			gs.Food[fi] = gs.Food[len(gs.Food)-1]
			gs.Food = gs.Food[:len(gs.Food)-1]
		}
	}

	// Phase 5 — Eliminate snakes (simultaneous).
	type elimination struct {
		index int
		cause string
	}
	var elims [MaxSnakes]elimination
	nElims := 0

	for i := 0; i < n; i++ {
		s := &gs.Snakes[i]
		if !s.IsAlive() || !moves.Has[i] {
			continue
		}

		// 5a: Starvation.
		if s.Health <= 0 {
			elims[nElims] = elimination{i, "starvation"}
			nElims++
			continue
		}

		head := s.Head()

		// 5b: Wall collision (out of bounds).
		if head.X < 0 || head.X >= gs.Width || head.Y < 0 || head.Y >= gs.Height {
			elims[nElims] = elimination{i, "wall"}
			nElims++
			continue
		}

		// 5c: Body collision (head on any body segment index > 0, including self).
		bodyHit := false
		for j := range gs.Snakes {
			other := &gs.Snakes[j]
			if !other.IsAlive() {
				continue
			}
			for seg := 1; seg < len(other.Body); seg++ {
				if head == other.Body[seg] {
					bodyHit = true
					break
				}
			}
			if bodyHit {
				break
			}
		}
		if bodyHit {
			elims[nElims] = elimination{i, "body-collision"}
			nElims++
			continue
		}
	}

	// Apply 5a-5c eliminations before head-to-head check.
	for e := 0; e < nElims; e++ {
		gs.Snakes[elims[e].index].EliminatedCause = elims[e].cause
	}

	// 5d: Head-to-head collisions (only among snakes still alive after 5a-5c).
	// Collect moved alive heads, then pairwise comparison (max 4 snakes = 6 pairs).
	type headInfo struct {
		idx  int
		head Coord
	}
	var heads [MaxSnakes]headInfo
	nHeads := 0
	for i := 0; i < n; i++ {
		s := &gs.Snakes[i]
		if !s.IsAlive() || !moves.Has[i] {
			continue
		}
		heads[nHeads] = headInfo{i, s.Head()}
		nHeads++
	}

	for i := 0; i < nHeads; i++ {
		// Count snakes at same position and find max length.
		sameCount := 0
		maxLen := 0
		for j := 0; j < nHeads; j++ {
			if heads[j].head == heads[i].head {
				sameCount++
				if gs.Snakes[heads[j].idx].Length > maxLen {
					maxLen = gs.Snakes[heads[j].idx].Length
				}
			}
		}
		if sameCount < 2 {
			continue
		}

		myLen := gs.Snakes[heads[i].idx].Length
		if myLen < maxLen {
			gs.Snakes[heads[i].idx].EliminatedCause = "head-collision"
		} else {
			// Count how many share max length at this position.
			maxCount := 0
			for j := 0; j < nHeads; j++ {
				if heads[j].head == heads[i].head && gs.Snakes[heads[j].idx].Length == maxLen {
					maxCount++
				}
			}
			if maxCount > 1 {
				gs.Snakes[heads[i].idx].EliminatedCause = "head-collision"
			}
		}
	}

	// Phase 6 — Increment turn.
	gs.Turn++
}
