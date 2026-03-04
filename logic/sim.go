package logic

// SimSnake represents a snake in the game simulator with full state.
type SimSnake struct {
	ID              string
	Body            []Coord // head-first: Body[0] = head
	Health          int
	Length          int
	EliminatedCause string // "" = alive
}

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

// NewGameSim creates a GameSim with deep-copied slices.
func NewGameSim(width, height int, snakes []SimSnake, food, hazards []Coord) *GameSim {
	gs := &GameSim{
		Width:  width,
		Height: height,
		Turn:   0,
	}

	gs.Snakes = make([]SimSnake, len(snakes))
	for i, s := range snakes {
		body := make([]Coord, len(s.Body))
		copy(body, s.Body)
		gs.Snakes[i] = SimSnake{
			ID:              s.ID,
			Body:            body,
			Health:          s.Health,
			Length:          s.Length,
			EliminatedCause: s.EliminatedCause,
		}
	}

	gs.Food = make([]Coord, len(food))
	copy(gs.Food, food)

	gs.Hazards = make([]Coord, len(hazards))
	copy(gs.Hazards, hazards)

	return gs
}

// Clone returns a deep copy of the GameSim with no shared backing arrays.
func (gs *GameSim) Clone() *GameSim {
	c := &GameSim{
		Width:  gs.Width,
		Height: gs.Height,
		Turn:   gs.Turn,
	}

	c.Snakes = make([]SimSnake, len(gs.Snakes))
	for i, s := range gs.Snakes {
		body := make([]Coord, len(s.Body))
		copy(body, s.Body)
		c.Snakes[i] = SimSnake{
			ID:              s.ID,
			Body:            body,
			Health:          s.Health,
			Length:          s.Length,
			EliminatedCause: s.EliminatedCause,
		}
	}

	c.Food = make([]Coord, len(gs.Food))
	copy(c.Food, gs.Food)

	c.Hazards = make([]Coord, len(gs.Hazards))
	copy(c.Hazards, gs.Hazards)

	return c
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
