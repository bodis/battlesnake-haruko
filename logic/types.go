package logic

// Coord mirrors the API coordinate type to avoid a circular import.
type Coord struct {
	X, Y int
}

// Snake mirrors the minimal snake fields needed for board updates.
type Snake struct {
	ID   string
	Body []Coord
}

// Direction represents a cardinal movement direction.
type Direction int

const (
	Up    Direction = 0
	Down  Direction = 1
	Left  Direction = 2
	Right Direction = 3
)

// AllDirections is the set of four cardinal directions.
var AllDirections = [4]Direction{Up, Down, Left, Right}

// DirectionName returns the API string for a direction.
func DirectionName(d Direction) string {
	switch d {
	case Up:
		return "up"
	case Down:
		return "down"
	case Left:
		return "left"
	case Right:
		return "right"
	}
	return ""
}

// Move returns the coordinate one step in the given direction.
func (c Coord) Move(d Direction) Coord {
	switch d {
	case Up:
		return Coord{c.X, c.Y + 1}
	case Down:
		return Coord{c.X, c.Y - 1}
	case Left:
		return Coord{c.X - 1, c.Y}
	case Right:
		return Coord{c.X + 1, c.Y}
	}
	return c
}

// MaxSnakes is the maximum number of snakes supported in a game.
const MaxSnakes = 4

// MoveSet holds a direction for each snake, indexed by snake position in GameSim.Snakes.
type MoveSet struct {
	Dir [MaxSnakes]Direction
	Has [MaxSnakes]bool
}

func newMoveSet1(idx int, dir Direction) MoveSet {
	var ms MoveSet
	ms.Dir[idx] = dir
	ms.Has[idx] = true
	return ms
}

func newMoveSet2(idx1 int, dir1 Direction, idx2 int, dir2 Direction) MoveSet {
	var ms MoveSet
	ms.Dir[idx1] = dir1
	ms.Has[idx1] = true
	ms.Dir[idx2] = dir2
	ms.Has[idx2] = true
	return ms
}
