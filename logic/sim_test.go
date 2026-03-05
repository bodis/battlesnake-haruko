package logic

import "testing"

// testMoves converts a map[string]Direction to a MoveSet using snake indices in the game.
func testMoves(g *GameSim, moves map[string]Direction) MoveSet {
	var ms MoveSet
	for id, dir := range moves {
		for i := range g.Snakes {
			if g.Snakes[i].ID == id {
				ms.Dir[i] = dir
				ms.Has[i] = true
				break
			}
		}
	}
	return ms
}

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

// --- CloneFromPool ---

func TestCloneFromPoolIndependence(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}}, Health: 100, Length: 2},
	}, []Coord{{3, 3}}, nil)

	cl := gs.CloneFromPool()
	cl.Snakes[0].Health = 50
	cl.Food[0] = Coord{0, 0}
	cl.Turn = 99

	if gs.Snakes[0].Health != 100 {
		t.Error("original snake Health changed after pooled clone mutation")
	}
	if gs.Food[0] != (Coord{3, 3}) {
		t.Error("original Food changed after pooled clone mutation")
	}
	if gs.Turn != 0 {
		t.Error("original Turn changed after pooled clone mutation")
	}
	cl.Release()
}

func TestCloneFromPoolReleaseCycle(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 80, Length: 3},
	}, nil, nil)

	// Clone, use, release multiple times to test pool reuse.
	for i := 0; i < 10; i++ {
		cl := gs.CloneFromPool()
		if cl.Snakes[0].Health != 80 {
			t.Errorf("iter %d: Health = %d, want 80", i, cl.Snakes[0].Health)
		}
		cl.Snakes[0].Health = 0 // mutate
		cl.Release()
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
		gs.MoveSnakes(testMoves(gs, map[string]Direction{"a": tt.dir}))
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
	gs.MoveSnakes(testMoves(gs, map[string]Direction{"a": Up, "b": Left}))

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
	gs.MoveSnakes(testMoves(gs, map[string]Direction{"a": Up}))

	s := &gs.Snakes[0]
	if len(s.Body) != 3 {
		t.Fatalf("body length = %d, want 3", len(s.Body))
	}
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
	gs.MoveSnakes(testMoves(gs, map[string]Direction{"a": Up}))

	if gs.Snakes[0].Body[0] != (Coord{5, 5}) {
		t.Error("dead snake should not have moved")
	}
}

func TestMoveSnakesMissingFromMap(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}}, Health: 100, Length: 2},
	}, nil, nil)
	gs.MoveSnakes(MoveSet{}) // empty MoveSet

	if gs.Snakes[0].Body[0] != (Coord{5, 5}) {
		t.Error("snake not in MoveSet should not have moved")
	}
}

func TestMoveSnakesSingleSegment(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{3, 3}}, Health: 100, Length: 1},
	}, nil, nil)
	gs.MoveSnakes(testMoves(gs, map[string]Direction{"a": Right}))

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

// --- IsAlive ---

func TestIsAlive(t *testing.T) {
	alive := SimSnake{ID: "a", Body: []Coord{{1, 1}}, Health: 100, Length: 1}
	if !alive.IsAlive() {
		t.Error("snake with no EliminatedCause should be alive")
	}

	dead := SimSnake{ID: "b", Body: []Coord{{1, 1}}, Health: 0, Length: 1, EliminatedCause: "wall"}
	if dead.IsAlive() {
		t.Error("snake with EliminatedCause should not be alive")
	}
}

// --- IsOver ---

func TestIsOverTwoAlive(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{1, 1}}, Health: 100, Length: 1},
		{ID: "b", Body: []Coord{{5, 5}}, Health: 100, Length: 1},
	}, nil, nil)
	if gs.IsOver() {
		t.Error("game with 2 alive snakes should not be over")
	}
}

func TestIsOverOneAlive(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{1, 1}}, Health: 100, Length: 1},
		{ID: "b", Body: []Coord{{5, 5}}, Health: 0, Length: 1, EliminatedCause: "wall"},
	}, nil, nil)
	if !gs.IsOver() {
		t.Error("game with 1 alive snake should be over")
	}
}

func TestIsOverAllDead(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{1, 1}}, Health: 0, Length: 1, EliminatedCause: "wall"},
		{ID: "b", Body: []Coord{{5, 5}}, Health: 0, Length: 1, EliminatedCause: "starvation"},
	}, nil, nil)
	if !gs.IsOver() {
		t.Error("game with 0 alive snakes should be over")
	}
}

// --- Step ---

func TestStepHealthDecrement(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 100, Length: 3},
	}, nil, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Up}))
	if gs.Snakes[0].Health != 99 {
		t.Errorf("Health = %d, want 99", gs.Snakes[0].Health)
	}
}

func TestStepStarvation(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 1, Length: 3},
	}, nil, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Up}))
	s := &gs.Snakes[0]
	if s.IsAlive() {
		t.Error("snake with health 1 should starve after step")
	}
	if s.EliminatedCause != "starvation" {
		t.Errorf("EliminatedCause = %q, want \"starvation\"", s.EliminatedCause)
	}
}

func TestStepWallCollision(t *testing.T) {
	tests := []struct {
		name string
		head Coord
		dir  Direction
	}{
		{"top wall", Coord{5, 10}, Up},
		{"bottom wall", Coord{5, 0}, Down},
		{"left wall", Coord{0, 5}, Left},
		{"right wall", Coord{10, 5}, Right},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := NewGameSim(11, 11, []SimSnake{
				{ID: "a", Body: []Coord{tt.head, {5, 5}}, Health: 100, Length: 2},
			}, nil, nil)
			gs.Step(testMoves(gs, map[string]Direction{"a": tt.dir}))
			s := &gs.Snakes[0]
			if s.IsAlive() {
				t.Error("snake should be eliminated by wall")
			}
			if s.EliminatedCause != "wall" {
				t.Errorf("EliminatedCause = %q, want \"wall\"", s.EliminatedCause)
			}
		})
	}
}

func TestStepBodyCollisionOther(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{3, 3}, {2, 3}}, Health: 100, Length: 2},
		{ID: "b", Body: []Coord{{5, 3}, {4, 3}, {4, 2}}, Health: 100, Length: 3},
	}, nil, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Right, "b": Up}))
	if gs.Snakes[0].EliminatedCause != "body-collision" {
		t.Errorf("snake a cause = %q, want \"body-collision\"", gs.Snakes[0].EliminatedCause)
	}
	if !gs.Snakes[1].IsAlive() {
		t.Error("snake b should still be alive")
	}
}

func TestStepSelfCollision(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{3, 3}, {4, 3}, {4, 2}, {3, 2}, {2, 2}, {2, 3}, {2, 4}}, Health: 100, Length: 7},
	}, nil, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Left}))
	s := &gs.Snakes[0]
	if s.IsAlive() {
		t.Error("snake should be eliminated by self-collision")
	}
	if s.EliminatedCause != "body-collision" {
		t.Errorf("EliminatedCause = %q, want \"body-collision\"", s.EliminatedCause)
	}
}

func TestStepEatFood(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 50, Length: 3},
	}, []Coord{{5, 6}}, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Up}))
	s := &gs.Snakes[0]
	if s.Health != 100 {
		t.Errorf("Health = %d, want 100", s.Health)
	}
	if s.Length != 4 {
		t.Errorf("Length = %d, want 4", s.Length)
	}
	if len(s.Body) != 4 {
		t.Errorf("len(Body) = %d, want 4", len(s.Body))
	}
	if len(gs.Food) != 0 {
		t.Errorf("Food count = %d, want 0", len(gs.Food))
	}
}

func TestStepEatFoodTailPosition(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 50, Length: 3},
	}, []Coord{{5, 6}}, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Up}))
	s := &gs.Snakes[0]
	if s.Tail() != (Coord{5, 3}) {
		t.Errorf("grown tail = %v, want {5,3}", s.Tail())
	}
}

func TestStepTwoSnakesEatSameFood(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{4, 5}, {3, 5}}, Health: 50, Length: 2},
		{ID: "b", Body: []Coord{{6, 5}, {7, 5}}, Health: 50, Length: 2},
	}, []Coord{{5, 5}}, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Right, "b": Left}))
	a := &gs.Snakes[0]
	b := &gs.Snakes[1]
	if a.Length != 3 {
		t.Errorf("snake a Length = %d, want 3", a.Length)
	}
	if b.Length != 3 {
		t.Errorf("snake b Length = %d, want 3", b.Length)
	}
	if len(gs.Food) != 0 {
		t.Errorf("Food count = %d, want 0", len(gs.Food))
	}
}

func TestStepHeadToHeadShorterDies(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{4, 5}, {3, 5}, {2, 5}}, Health: 100, Length: 3},
		{ID: "b", Body: []Coord{{6, 5}, {7, 5}}, Health: 100, Length: 2},
	}, nil, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Right, "b": Left}))
	a := &gs.Snakes[0]
	b := &gs.Snakes[1]
	if !a.IsAlive() {
		t.Error("longer snake a should survive head-to-head")
	}
	if b.IsAlive() {
		t.Error("shorter snake b should be eliminated")
	}
	if b.EliminatedCause != "head-collision" {
		t.Errorf("snake b cause = %q, want \"head-collision\"", b.EliminatedCause)
	}
}

func TestStepHeadToHeadEqualBothDie(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{4, 5}, {3, 5}}, Health: 100, Length: 2},
		{ID: "b", Body: []Coord{{6, 5}, {7, 5}}, Health: 100, Length: 2},
	}, nil, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Right, "b": Left}))
	a := &gs.Snakes[0]
	b := &gs.Snakes[1]
	if a.IsAlive() || b.IsAlive() {
		t.Error("equal length head-to-head should eliminate both")
	}
	if a.EliminatedCause != "head-collision" || b.EliminatedCause != "head-collision" {
		t.Errorf("causes = %q, %q; want both \"head-collision\"", a.EliminatedCause, b.EliminatedCause)
	}
}

func TestStepHeadToHeadThreeSnakes(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{4, 5}, {3, 5}, {2, 5}}, Health: 100, Length: 3},
		{ID: "b", Body: []Coord{{5, 4}, {5, 3}}, Health: 100, Length: 2},
		{ID: "c", Body: []Coord{{6, 5}, {7, 5}, {8, 5}}, Health: 100, Length: 3},
	}, nil, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Right, "b": Up, "c": Left}))
	for _, s := range gs.Snakes {
		if s.IsAlive() {
			t.Errorf("snake %s should be eliminated in 3-way head-to-head", s.ID)
		}
		if s.EliminatedCause != "head-collision" {
			t.Errorf("snake %s cause = %q, want \"head-collision\"", s.ID, s.EliminatedCause)
		}
	}
}

func TestStepHazardDamage(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 100, Length: 3},
	}, nil, []Coord{{5, 6}})
	gs.Step(testMoves(gs, map[string]Direction{"a": Up}))
	if gs.Snakes[0].Health != 85 {
		t.Errorf("Health = %d, want 85", gs.Snakes[0].Health)
	}
}

func TestStepEatFoodOnHazard(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 100, Length: 3},
	}, []Coord{{5, 6}}, []Coord{{5, 6}})
	gs.Step(testMoves(gs, map[string]Direction{"a": Up}))
	s := &gs.Snakes[0]
	if s.Health != 100 {
		t.Errorf("Health = %d, want 100 (food restores after hazard)", s.Health)
	}
	if !s.IsAlive() {
		t.Error("snake should be alive after eating on hazard")
	}
}

func TestStepAlreadyEliminatedSkipped(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}}, Health: 0, Length: 2, EliminatedCause: "wall"},
		{ID: "b", Body: []Coord{{3, 3}, {3, 2}}, Health: 100, Length: 2},
	}, nil, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Up, "b": Up}))
	a := &gs.Snakes[0]
	if a.Body[0] != (Coord{5, 5}) {
		t.Error("dead snake should not have moved")
	}
	if a.Health != 0 {
		t.Error("dead snake health should not change")
	}
}

func TestStepTurnIncrement(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{5, 5}, {5, 4}}, Health: 100, Length: 2},
	}, nil, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Up}))
	if gs.Turn != 1 {
		t.Errorf("Turn = %d, want 1", gs.Turn)
	}
	gs.Step(testMoves(gs, map[string]Direction{"a": Up}))
	if gs.Turn != 2 {
		t.Errorf("Turn = %d, want 2", gs.Turn)
	}
}

func TestStepSimultaneousBodyCollision(t *testing.T) {
	gs := NewGameSim(11, 11, []SimSnake{
		{ID: "a", Body: []Coord{{3, 5}, {3, 4}, {3, 3}}, Health: 100, Length: 3},
		{ID: "b", Body: []Coord{{5, 5}, {4, 5}, {4, 4}}, Health: 100, Length: 3},
	}, nil, nil)
	gs.Step(testMoves(gs, map[string]Direction{"a": Right, "b": Up}))
	a := &gs.Snakes[0]
	b := &gs.Snakes[1]
	if a.IsAlive() {
		t.Error("snake a should be eliminated by body collision with b")
	}
	if a.EliminatedCause != "body-collision" {
		t.Errorf("snake a cause = %q, want \"body-collision\"", a.EliminatedCause)
	}
	if !b.IsAlive() {
		t.Error("snake b should be alive")
	}
}
