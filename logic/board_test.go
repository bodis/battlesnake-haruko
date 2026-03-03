package logic

import "testing"

func TestGetIndex(t *testing.T) {
	b := NewFastBoard(11, 11)
	if got := b.GetIndex(0, 0); got != 0 {
		t.Errorf("GetIndex(0,0) = %d, want 0", got)
	}
	if got := b.GetIndex(10, 10); got != 120 {
		t.Errorf("GetIndex(10,10) = %d, want 120", got)
	}
	if got := b.GetIndex(3, 2); got != 25 {
		t.Errorf("GetIndex(3,2) = %d, want 25", got)
	}
}

func TestGetCoords(t *testing.T) {
	b := NewFastBoard(11, 11)
	x, y := b.GetCoords(25)
	if x != 3 || y != 2 {
		t.Errorf("GetCoords(25) = (%d,%d), want (3,2)", x, y)
	}
}

func TestInBounds(t *testing.T) {
	b := NewFastBoard(11, 11)
	tests := []struct {
		x, y int
		want bool
	}{
		{0, 0, true},
		{10, 10, true},
		{-1, 0, false},
		{0, -1, false},
		{11, 0, false},
		{0, 11, false},
		{5, 5, true},
	}
	for _, tt := range tests {
		if got := b.InBounds(tt.x, tt.y); got != tt.want {
			t.Errorf("InBounds(%d,%d) = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
}

func TestIsBlocked(t *testing.T) {
	b := NewFastBoard(11, 11)
	b.Cells[b.GetIndex(1, 1)] = CellSnake
	b.Cells[b.GetIndex(2, 2)] = CellMySnake
	b.Cells[b.GetIndex(3, 3)] = CellHazard
	b.Cells[b.GetIndex(4, 4)] = CellFood
	b.Cells[b.GetIndex(5, 5)] = CellSnakeTail

	if !b.IsBlocked(1, 1) {
		t.Error("CellSnake should be blocked")
	}
	if !b.IsBlocked(2, 2) {
		t.Error("CellMySnake should be blocked")
	}
	if !b.IsBlocked(3, 3) {
		t.Error("CellHazard should be blocked")
	}
	if b.IsBlocked(4, 4) {
		t.Error("CellFood should not be blocked")
	}
	if b.IsBlocked(5, 5) {
		t.Error("CellSnakeTail should not be blocked")
	}
	if b.IsBlocked(0, 0) {
		t.Error("CellEmpty should not be blocked")
	}
}

func TestUpdateTailAwareness(t *testing.T) {
	b := NewFastBoard(5, 5)
	snakes := []Snake{
		{
			ID:   "me",
			Body: []Coord{{2, 2}, {2, 1}, {2, 0}}, // tail at (2,0), not stacked
		},
	}
	b.Update(nil, nil, snakes, "me")

	if b.Cells[b.GetIndex(2, 2)] != CellMySnake {
		t.Error("head should be CellMySnake")
	}
	if b.Cells[b.GetIndex(2, 0)] != CellSnakeTail {
		t.Error("non-stacked tail should be CellSnakeTail")
	}

	// Stacked tail (just ate)
	snakes[0].Body = []Coord{{2, 2}, {2, 1}, {2, 0}, {2, 0}}
	b.Update(nil, nil, snakes, "me")
	if b.Cells[b.GetIndex(2, 0)] != CellMySnake {
		t.Error("stacked tail should remain CellMySnake")
	}
}
