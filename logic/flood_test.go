package logic

import "testing"

func TestFloodFillOpenBoard(t *testing.T) {
	b := NewFastBoard(11, 11)
	got := b.FloodFill(Coord{5, 5})
	if got != 121 {
		t.Errorf("open 11x11 board: FloodFill = %d, want 121", got)
	}
}

func TestFloodFillFromCorner(t *testing.T) {
	b := NewFastBoard(11, 11)
	got := b.FloodFill(Coord{0, 0})
	if got != 121 {
		t.Errorf("open board from corner: FloodFill = %d, want 121", got)
	}
}

func TestFloodFillTrapped(t *testing.T) {
	b := NewFastBoard(5, 5)
	// Surround (0,0) with snake cells
	b.Cells[b.GetIndex(1, 0)] = CellSnake
	b.Cells[b.GetIndex(0, 1)] = CellSnake
	got := b.FloodFill(Coord{0, 0})
	if got != 1 {
		t.Errorf("trapped cell: FloodFill = %d, want 1", got)
	}
}

func TestFloodFillPartitioned(t *testing.T) {
	// 5x5 board with a snake wall at x=2, splitting into left (2 cols) and right (2 cols)
	b := NewFastBoard(5, 5)
	for y := 0; y < 5; y++ {
		b.Cells[b.GetIndex(2, y)] = CellSnake
	}
	got := b.FloodFill(Coord{0, 0})
	if got != 10 { // 2 columns * 5 rows
		t.Errorf("partitioned board left side: FloodFill = %d, want 10", got)
	}
	got = b.FloodFill(Coord{4, 4})
	if got != 10 {
		t.Errorf("partitioned board right side: FloodFill = %d, want 10", got)
	}
}

func TestFloodFillFoodPassable(t *testing.T) {
	b := NewFastBoard(3, 3)
	// Fill middle row with food
	b.Cells[b.GetIndex(0, 1)] = CellFood
	b.Cells[b.GetIndex(1, 1)] = CellFood
	b.Cells[b.GetIndex(2, 1)] = CellFood
	got := b.FloodFill(Coord{0, 0})
	if got != 9 {
		t.Errorf("food should be passable: FloodFill = %d, want 9", got)
	}
}

func TestFloodFillSnakeTailPassable(t *testing.T) {
	b := NewFastBoard(3, 3)
	b.Cells[b.GetIndex(1, 1)] = CellSnakeTail
	got := b.FloodFill(Coord{0, 0})
	if got != 9 {
		t.Errorf("snake tail should be passable: FloodFill = %d, want 9", got)
	}
}

func TestFloodFillOutOfBounds(t *testing.T) {
	b := NewFastBoard(5, 5)
	got := b.FloodFill(Coord{-1, 0})
	if got != 0 {
		t.Errorf("out-of-bounds start: FloodFill = %d, want 0", got)
	}
}

func TestFloodFillBlockedStart(t *testing.T) {
	b := NewFastBoard(5, 5)
	b.Cells[b.GetIndex(2, 2)] = CellSnake
	got := b.FloodFill(Coord{2, 2})
	if got != 0 {
		t.Errorf("blocked start: FloodFill = %d, want 0", got)
	}
}

func TestFloodFillHazardBlocks(t *testing.T) {
	b := NewFastBoard(5, 5)
	// Hazard wall at x=2
	for y := 0; y < 5; y++ {
		b.Cells[b.GetIndex(2, y)] = CellHazard
	}
	got := b.FloodFill(Coord{0, 0})
	if got != 10 {
		t.Errorf("hazard wall should block: FloodFill = %d, want 10", got)
	}
}

func TestDirectionName(t *testing.T) {
	tests := []struct {
		d    Direction
		want string
	}{
		{Up, "up"},
		{Down, "down"},
		{Left, "left"},
		{Right, "right"},
	}
	for _, tt := range tests {
		if got := DirectionName(tt.d); got != tt.want {
			t.Errorf("DirectionName(%d) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestCoordMove(t *testing.T) {
	c := Coord{5, 5}
	tests := []struct {
		d    Direction
		want Coord
	}{
		{Up, Coord{5, 6}},
		{Down, Coord{5, 4}},
		{Left, Coord{4, 5}},
		{Right, Coord{6, 5}},
	}
	for _, tt := range tests {
		if got := c.Move(tt.d); got != tt.want {
			t.Errorf("Coord{5,5}.Move(%d) = %v, want %v", tt.d, got, tt.want)
		}
	}
}
