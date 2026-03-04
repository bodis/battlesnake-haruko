package logic

import "testing"

// --- Initialization ---

func TestNewGameSimInitialization(t *testing.T) {
	snakes := []SimSnake{
		{ID: "a", Body: []Coord{{1, 1}, {1, 0}}, Health: 100, Length: 2},
	}
	food := []Coord{{3, 3}}
	hazards := []Coord{{4, 4}}

	gs := NewGameSim(11, 11, snakes, food, hazards)

	if gs.Width != 11 || gs.Height != 11 {
		t.Errorf("dimensions = %dx%d, want 11x11", gs.Width, gs.Height)
	}
	if gs.Turn != 0 {
		t.Errorf("Turn = %d, want 0", gs.Turn)
	}
	if len(gs.Snakes) != 1 || gs.Snakes[0].ID != "a" {
		t.Error("snake not initialized correctly")
	}
	if gs.Snakes[0].Health != 100 || gs.Snakes[0].Length != 2 {
		t.Error("snake Health/Length not set")
	}
	if len(gs.Food) != 1 || gs.Food[0] != (Coord{3, 3}) {
		t.Error("food not initialized correctly")
	}
	if len(gs.Hazards) != 1 || gs.Hazards[0] != (Coord{4, 4}) {
		t.Error("hazards not initialized correctly")
	}
}

func TestNewGameSimOwnsSlices(t *testing.T) {
	body := []Coord{{5, 5}, {5, 4}}
	snakes := []SimSnake{{ID: "a", Body: body, Health: 100, Length: 2}}
	food := []Coord{{3, 3}}
	hazards := []Coord{{4, 4}}

	gs := NewGameSim(11, 11, snakes, food, hazards)

	// Mutate inputs
	body[0] = Coord{9, 9}
	snakes[0].ID = "changed"
	food[0] = Coord{0, 0}
	hazards[0] = Coord{0, 0}

	if gs.Snakes[0].ID != "a" {
		t.Error("GameSim snake ID mutated by external change")
	}
	if gs.Snakes[0].Body[0] != (Coord{5, 5}) {
		t.Error("GameSim snake Body mutated by external change")
	}
	if gs.Food[0] != (Coord{3, 3}) {
		t.Error("GameSim Food mutated by external change")
	}
	if gs.Hazards[0] != (Coord{4, 4}) {
		t.Error("GameSim Hazards mutated by external change")
	}
}

// --- Clone ---

func TestCloneIndependence(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}}, Health: 100, Length: 2},
	}, []Coord{{3, 3}}, nil)

	cl := gs.Clone()
	cl.Snakes[0].Health = 50
	cl.Food[0] = Coord{0, 0}
	cl.Turn = 99

	if gs.Snakes[0].Health != 100 {
		t.Error("original snake Health changed after clone mutation")
	}
	if gs.Food[0] != (Coord{3, 3}) {
		t.Error("original Food changed after clone mutation")
	}
	if gs.Turn != 0 {
		t.Error("original Turn changed after clone mutation")
	}
}

func TestCloneDeepCopiesSnakes(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 80, Length: 3},
	}, nil, nil)

	cl := gs.Clone()
	cl.Snakes[0].Body[0] = Coord{9, 9}

	if gs.Snakes[0].Body[0] != (Coord{5, 5}) {
		t.Error("original snake Body changed after clone Body mutation")
	}
}

func TestCloneDeepCopiesFood(t *testing.T) {
	gs := NewGameSim(11, 11, nil, []Coord{{1, 1}, {2, 2}}, nil)

	cl := gs.Clone()
	cl.Food[0] = Coord{9, 9}

	if gs.Food[0] != (Coord{1, 1}) {
		t.Error("original Food changed after clone Food mutation")
	}
}

func TestCloneDeepCopiesHazards(t *testing.T) {
	gs := NewGameSim(11, 11, nil, nil, []Coord{{7, 7}})

	cl := gs.Clone()
	cl.Hazards[0] = Coord{0, 0}

	if gs.Hazards[0] != (Coord{7, 7}) {
		t.Error("original Hazards changed after clone Hazards mutation")
	}
}

func TestClonePreservesScalarFields(t *testing.T) {
	gs := NewGameSim(13, 7, []SimSnake{
		{ID: "a", Body: []Coord{{1, 1}}, Health: 42, Length: 5},
	}, nil, nil)
	gs.Turn = 17

	cl := gs.Clone()

	if cl.Width != 13 || cl.Height != 7 {
		t.Errorf("clone dimensions = %dx%d, want 13x7", cl.Width, cl.Height)
	}
	if cl.Turn != 17 {
		t.Errorf("clone Turn = %d, want 17", cl.Turn)
	}
	if cl.Snakes[0].Health != 42 || cl.Snakes[0].Length != 5 {
		t.Error("clone snake scalars don't match original")
	}
}

// --- Movement ---

func TestMoveSnakesAllDirections(t *testing.T) {
	tests := []struct {
		dir  Direction
		want Coord
	}{
		{Up, Coord{5, 6}},
		{Down, Coord{5, 4}},
		{Left, Coord{4, 5}},
		{Right, Coord{6, 5}},
	}
	for _, tt := range tests {
		gs := NewGameSim(11, 11, []SimSnake{
			{ID: "a", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 100, Length: 3},
		}, nil, nil)
		gs.MoveSnakes(map[string]Direction{"a": tt.dir})
		got := gs.Snakes[0].Body[0]
		if got != tt.want {
			t.Errorf("Move(%d): head = %v, want %v", tt.dir, got, tt.want)
		}
	}
}

func TestMoveSnakesMultiSnake(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{2, 2}, {2, 1}}, Health: 100, Length: 2},
		{ID: "b", Body: []Coord{{8, 8}, {8, 7}}, Health: 100, Length: 2},
	}, nil, nil)
	gs.MoveSnakes(map[string]Direction{"a": Up, "b": Left})

	if gs.Snakes[0].Body[0] != (Coord{2, 3}) {
		t.Errorf("snake a head = %v, want {2,3}", gs.Snakes[0].Body[0])
	}
	if gs.Snakes[1].Body[0] != (Coord{7, 8}) {
		t.Errorf("snake b head = %v, want {7,8}", gs.Snakes[1].Body[0])
	}
}

func TestMoveSnakesTailRemoved(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 100, Length: 3},
	}, nil, nil)
	gs.MoveSnakes(map[string]Direction{"a": Up})

	s := &gs.Snakes[0]
	if len(s.Body) != 3 {
		t.Fatalf("body length = %d, want 3", len(s.Body))
	}
	// New body: {5,6}, {5,5}, {5,4} — old tail {5,3} gone
	expected := []Coord{{5, 6}, {5, 5}, {5, 4}}
	for i, want := range expected {
		if s.Body[i] != want {
			t.Errorf("Body[%d] = %v, want %v", i, s.Body[i], want)
		}
	}
}

func TestMoveSnakesDeadSnakeSkipped(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}}, Health: 0, Length: 2, EliminatedCause: "head-collision"},
	}, nil, nil)
	gs.MoveSnakes(map[string]Direction{"a": Up})

	if gs.Snakes[0].Body[0] != (Coord{5, 5}) {
		t.Error("dead snake should not have moved")
	}
}

func TestMoveSnakesMissingFromMap(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}}, Health: 100, Length: 2},
	}, nil, nil)
	gs.MoveSnakes(map[string]Direction{}) // empty map

	if gs.Snakes[0].Body[0] != (Coord{5, 5}) {
		t.Error("snake not in map should not have moved")
	}
}

func TestMoveSnakesSingleSegment(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{3, 3}}, Health: 100, Length: 1},
	}, nil, nil)
	gs.MoveSnakes(map[string]Direction{"a": Right})

	s := &gs.Snakes[0]
	if len(s.Body) != 1 {
		t.Fatalf("body length = %d, want 1", len(s.Body))
	}
	if s.Body[0] != (Coord{4, 3}) {
		t.Errorf("single-segment head = %v, want {4,3}", s.Body[0])
	}
}

// --- SnakeByID ---

func TestSnakeByIDFound(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{1, 1}}, Health: 100, Length: 1},
		{ID: "b", Body: []Coord{{2, 2}}, Health: 80, Length: 1},
	}, nil, nil)

	s := gs.SnakeByID("b")
	if s == nil || s.ID != "b" {
		t.Error("SnakeByID('b') should return snake b")
	}
}

func TestSnakeByIDNotFound(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{1, 1}}, Health: 100, Length: 1},
	}, nil, nil)

	if gs.SnakeByID("z") != nil {
		t.Error("SnakeByID('z') should return nil")
	}
}

func TestSnakeByIDMutation(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{1, 1}}, Health: 100, Length: 1},
	}, nil, nil)

	s := gs.SnakeByID("a")
	s.Health = 50

	if gs.Snakes[0].Health != 50 {
		t.Error("SnakeByID should return pointer allowing in-place mutation")
	}
}

// --- Accessors ---

func TestSimSnakeHead(t *testing.T) {
	s := SimSnake{Body: []Coord{{3, 4}, {3, 3}, {3, 2}}}
	if s.Head() != (Coord{3, 4}) {
		t.Errorf("Head() = %v, want {3,4}", s.Head())
	}
}

func TestSimSnakeTail(t *testing.T) {
	s := SimSnake{Body: []Coord{{3, 4}, {3, 3}, {3, 2}}}
	if s.Tail() != (Coord{3, 2}) {
		t.Errorf("Tail() = %v, want {3,2}", s.Tail())
	}
}
