package logic

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

// MoveSnakes applies the given moves to each alive snake.
// For each alive snake in the map, the head advances one step in the given
// direction and the tail is dropped (in-place shift, no allocation).
// Dead snakes and snakes not in the map are skipped.
func (gs *GameSim) MoveSnakes(moves map[string]Direction) {
	for i := range gs.Snakes {
		s := &gs.Snakes[i]
		if s.EliminatedCause != "" {
			continue
		}
		dir, ok := moves[s.ID]
		if !ok {
			continue
		}
		newHead := s.Body[0].Move(dir)
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
// moves maps snake ID → direction for each alive snake that should move.
func (gs *GameSim) Step(moves map[string]Direction) {
	n := len(gs.Snakes)

	// Phase 0 — Save tails before movement (for growth).
	savedTails := make([]Coord, n)
	for i := range gs.Snakes {
		s := &gs.Snakes[i]
		if s.IsAlive() {
			if _, ok := moves[s.ID]; ok {
				savedTails[i] = s.Tail()
			}
		}
	}

	// Phase 1 — Move snakes.
	gs.MoveSnakes(moves)

	// Phase 2 — Reduce health by 1.
	for i := range gs.Snakes {
		s := &gs.Snakes[i]
		if s.IsAlive() {
			if _, ok := moves[s.ID]; ok {
				s.Health--
			}
		}
	}

	// Phase 3 — Hazard damage.
	for i := range gs.Snakes {
		s := &gs.Snakes[i]
		if !s.IsAlive() {
			continue
		}
		if _, ok := moves[s.ID]; !ok {
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
	eaten := make([]bool, len(gs.Food))
	for i := range gs.Snakes {
		s := &gs.Snakes[i]
		if !s.IsAlive() {
			continue
		}
		if _, ok := moves[s.ID]; !ok {
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
	for fi := len(eaten) - 1; fi >= 0; fi-- {
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
	var elims []elimination

	for i := range gs.Snakes {
		s := &gs.Snakes[i]
		if !s.IsAlive() {
			continue
		}
		if _, ok := moves[s.ID]; !ok {
			continue
		}

		// 5a: Starvation.
		if s.Health <= 0 {
			elims = append(elims, elimination{i, "starvation"})
			continue
		}

		head := s.Head()

		// 5b: Wall collision (out of bounds).
		if head.X < 0 || head.X >= gs.Width || head.Y < 0 || head.Y >= gs.Height {
			elims = append(elims, elimination{i, "wall"})
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
			elims = append(elims, elimination{i, "body-collision"})
			continue
		}
	}

	// Apply 5a-5c eliminations before head-to-head check.
	for _, e := range elims {
		gs.Snakes[e.index].EliminatedCause = e.cause
	}

	// 5d: Head-to-head collisions (only among snakes still alive after 5a-5c).
	// Group alive snake heads by coordinate.
	type headGroup struct {
		indices []int
	}
	headMap := make(map[Coord]*headGroup)
	for i := range gs.Snakes {
		s := &gs.Snakes[i]
		if !s.IsAlive() {
			continue
		}
		if _, ok := moves[s.ID]; !ok {
			continue
		}
		head := s.Head()
		if g, ok := headMap[head]; ok {
			g.indices = append(g.indices, i)
		} else {
			headMap[head] = &headGroup{indices: []int{i}}
		}
	}
	for _, g := range headMap {
		if len(g.indices) < 2 {
			continue
		}
		// Find max length.
		maxLen := 0
		for _, idx := range g.indices {
			if gs.Snakes[idx].Length > maxLen {
				maxLen = gs.Snakes[idx].Length
			}
		}
		// Count how many share max length.
		maxCount := 0
		for _, idx := range g.indices {
			if gs.Snakes[idx].Length == maxLen {
				maxCount++
			}
		}
		for _, idx := range g.indices {
			if gs.Snakes[idx].Length < maxLen {
				gs.Snakes[idx].EliminatedCause = "head-collision"
			} else if maxCount > 1 {
				gs.Snakes[idx].EliminatedCause = "head-collision"
			}
		}
	}

	// Phase 6 — Increment turn.
	gs.Turn++
}
