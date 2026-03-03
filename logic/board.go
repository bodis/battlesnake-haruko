package logic

// Cell type constants for the FastBoard.
const (
	CellEmpty    uint8 = 0
	CellFood     uint8 = 1
	CellHazard   uint8 = 2
	CellSnake    uint8 = 3 // any opponent body
	CellMySnake  uint8 = 4 // our own body (head is index 0 in Body)
	CellSnakeTail uint8 = 5 // tail segment that will move away next turn
)

// Coord mirrors the API coordinate type to avoid a circular import.
type Coord struct {
	X, Y int
}

// Snake mirrors the minimal snake fields needed for board updates.
type Snake struct {
	ID   string
	Body []Coord
}

// FastBoard represents the game grid as a flat 1-D array for O(1) index
// arithmetic instead of 2-D slice allocations.  Width * Height cells.
type FastBoard struct {
	Cells  []uint8
	Width  int
	Height int
}

// NewFastBoard allocates a FastBoard for the given dimensions.
func NewFastBoard(width, height int) *FastBoard {
	return &FastBoard{
		Cells:  make([]uint8, width*height),
		Width:  width,
		Height: height,
	}
}

// GetIndex converts (x, y) board coordinates to a flat array index.
// Origin (0,0) is the bottom-left corner, matching the Battlesnake API.
func (b *FastBoard) GetIndex(x, y int) int {
	return y*b.Width + x
}

// GetCoords is the inverse of GetIndex.
func (b *FastBoard) GetCoords(index int) (x, y int) {
	return index % b.Width, index / b.Width
}

// InBounds reports whether (x, y) is within the board boundaries.
func (b *FastBoard) InBounds(x, y int) bool {
	return x >= 0 && x < b.Width && y >= 0 && y < b.Height
}

// IsBlocked reports whether the cell at (x, y) is impassable (snake body or
// hazard).  Food and snake tails are not considered blocked.
func (b *FastBoard) IsBlocked(x, y int) bool {
	c := b.Cells[b.GetIndex(x, y)]
	return c == CellSnake || c == CellMySnake || c == CellHazard
}

// Update refreshes the board from the API types passed in from the game state.
// myID is our snake's ID so we can mark our own body separately.
func (b *FastBoard) Update(food, hazards []Coord, snakes []Snake, myID string) {
	// Reset all cells.
	for i := range b.Cells {
		b.Cells[i] = CellEmpty
	}

	for _, f := range food {
		b.Cells[b.GetIndex(f.X, f.Y)] = CellFood
	}

	for _, h := range hazards {
		b.Cells[b.GetIndex(h.X, h.Y)] = CellHazard
	}

	for _, s := range snakes {
		cellType := CellSnake
		if s.ID == myID {
			cellType = CellMySnake
		}
		for _, seg := range s.Body {
			b.Cells[b.GetIndex(seg.X, seg.Y)] = cellType
		}
		// Mark tail as passable if the snake didn't just eat (tail will move
		// away next turn). A stacked tail (last two segments equal) means
		// the snake just ate and the tail won't move.
		n := len(s.Body)
		if n >= 2 {
			tail := s.Body[n-1]
			prev := s.Body[n-2]
			if tail.X != prev.X || tail.Y != prev.Y {
				b.Cells[b.GetIndex(tail.X, tail.Y)] = CellSnakeTail
			}
		}
	}
}
