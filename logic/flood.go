package logic

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

// FloodFill counts the number of reachable cells from start using BFS.
// A cell is passable if it is empty, food, or a snake tail.
// Snake bodies and hazards are impassable. Out-of-bounds start returns 0.
func (b *FastBoard) FloodFill(start Coord) int {
	if !b.InBounds(start.X, start.Y) {
		return 0
	}
	startCell := b.Cells[b.GetIndex(start.X, start.Y)]
	if startCell == CellSnake || startCell == CellMySnake || startCell == CellHazard {
		return 0
	}

	visited := make([]bool, len(b.Cells))
	queue := make([]Coord, 0, len(b.Cells)/2)
	idx := b.GetIndex(start.X, start.Y)
	visited[idx] = true
	queue = append(queue, start)
	count := 0

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		count++

		for _, d := range AllDirections {
			next := cur.Move(d)
			if !b.InBounds(next.X, next.Y) {
				continue
			}
			ni := b.GetIndex(next.X, next.Y)
			if visited[ni] {
				continue
			}
			c := b.Cells[ni]
			if c == CellSnake || c == CellMySnake || c == CellHazard {
				continue
			}
			visited[ni] = true
			queue = append(queue, next)
		}
	}
	return count
}
