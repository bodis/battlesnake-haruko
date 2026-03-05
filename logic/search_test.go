package logic

import (
	"testing"
	"time"
)

// makeSnake is a helper to build a SimSnake with full health.
func makeSnake(id string, body []Coord) SimSnake {
	return SimSnake{ID: id, Body: body, Health: 100, Length: len(body)}
}

// TestBestMove_DeadEndAvoidance: one direction leads to a 1-cell pocket,
// others open — should not pick the pocket direction.
func TestBestMove_DeadEndAvoidance(t *testing.T) {
	me := makeSnake("me", []Coord{{2, 2}, {2, 1}})
	opp := makeSnake("opp", []Coord{
		{4, 2}, {4, 1}, {4, 0}, {3, 0}, {3, 1}, {3, 2}, {3, 3}, {3, 4}, {4, 4}, {4, 3},
	})
	g := NewGameSim(5, 5, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMove("me", 3)
	if dir == Right {
		t.Errorf("BestMove picked Right (dead-end pocket), expected Up or Left")
	}
}

// TestBestMove_NeckTrap: opponent body walls force a single escape.
func TestBestMove_NeckTrap(t *testing.T) {
	me2 := makeSnake("me", []Coord{{2, 0}, {2, 1}, {2, 2}, {2, 3}})
	opp2 := makeSnake("opp", []Coord{{1, 0}, {0, 0}, {0, 1}, {0, 2}, {0, 3}})
	g2 := NewGameSim(5, 5, []SimSnake{me2, opp2}, nil, nil)
	dir2 := g2.BestMove("me", 3)
	if dir2 != Right {
		t.Errorf("BestMove expected Right (only escape), got %v", DirectionName(dir2))
	}
}

// TestBestMove_HeadToHeadKill: we're longer, moving to a square adjacent to
// opponent's head enables a head-to-head kill — prefer it.
func TestBestMove_HeadToHeadKill(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}})
	opp := makeSnake("opp", []Coord{{5, 7}, {5, 8}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMove("me", 3)
	if dir != Up {
		t.Errorf("BestMove expected Up (head-to-head kill opportunity), got %v", DirectionName(dir))
	}
}

// TestBestMove_FoodReachable: verify BestMove doesn't panic and returns a valid direction.
func TestBestMove_FoodReachable(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}})
	opp := makeSnake("opp", []Coord{{0, 0}, {0, 1}})
	food := []Coord{{5, 6}}
	g := NewGameSim(11, 11, []SimSnake{me, opp}, food, nil)

	dir := g.BestMove("me", 3)
	valid := dir == Up || dir == Down || dir == Left || dir == Right
	if !valid {
		t.Errorf("BestMove returned invalid direction %v", dir)
	}
}

// TestBestMove_AlreadyDead: if all moves lead to death, BestMove returns
// a valid Direction without panicking.
func TestBestMove_AlreadyDead(t *testing.T) {
	me := makeSnake("me", []Coord{
		{1, 1},
		{0, 1}, {0, 0}, {1, 0}, {2, 0}, {2, 1}, {2, 2}, {1, 2}, {0, 2},
	})
	g := NewGameSim(3, 3, []SimSnake{me}, nil, nil)

	dir := g.BestMove("me", 3)
	valid := dir == Up || dir == Down || dir == Left || dir == Right
	if !valid {
		t.Errorf("BestMove returned invalid direction %v", dir)
	}
}

// TestBestMove_NoOpponents: with no opponents, picks the move with most space.
func TestBestMove_NoOpponents(t *testing.T) {
	me := makeSnake("me", []Coord{{0, 2}, {0, 1}, {0, 0}})
	g := NewGameSim(5, 5, []SimSnake{me}, nil, nil)

	dir := g.BestMove("me", 3)
	valid := dir == Up || dir == Down || dir == Left || dir == Right
	if !valid {
		t.Errorf("BestMove returned invalid direction %v", dir)
	}
	if dir == Down || dir == Left {
		t.Errorf("BestMove picked a clearly bad direction %v", DirectionName(dir))
	}
}

// TestBestMove_DepthComparison: deeper search should not panic.
func TestBestMove_DepthComparison(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}})
	opp := makeSnake("opp", []Coord{{3, 5}, {3, 6}, {3, 7}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	for _, depth := range []int{1, 2, 3} {
		d := g.BestMove("me", depth)
		valid := d == Up || d == Down || d == Left || d == Right
		if !valid {
			t.Errorf("BestMove(depth=%d) returned invalid direction %v", depth, d)
		}
	}
}

// TestBestMoveIterative_ReturnsValidMove: generous budget, verify valid direction.
func TestBestMoveIterative_ReturnsValidMove(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}})
	opp := makeSnake("opp", []Coord{{3, 5}, {3, 6}, {3, 7}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMoveIterative("me", 500*time.Millisecond)
	valid := dir == Up || dir == Down || dir == Left || dir == Right
	if !valid {
		t.Errorf("BestMoveIterative returned invalid direction %v", dir)
	}
}

// TestBestMoveIterative_TinyBudget: 1ms budget, still returns a valid move.
func TestBestMoveIterative_TinyBudget(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}})
	opp := makeSnake("opp", []Coord{{3, 5}, {3, 6}, {3, 7}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMoveIterative("me", 1*time.Millisecond)
	valid := dir == Up || dir == Down || dir == Left || dir == Right
	if !valid {
		t.Errorf("BestMoveIterative with tiny budget returned invalid direction %v", dir)
	}
}

// TestBestMoveIterative_DeadEndAvoidance: same scenario as TestBestMove_DeadEndAvoidance.
func TestBestMoveIterative_DeadEndAvoidance(t *testing.T) {
	me := makeSnake("me", []Coord{{2, 2}, {2, 1}})
	opp := makeSnake("opp", []Coord{
		{4, 2}, {4, 1}, {4, 0}, {3, 0}, {3, 1}, {3, 2}, {3, 3}, {3, 4}, {4, 4}, {4, 3},
	})
	g := NewGameSim(5, 5, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMoveIterative("me", 300*time.Millisecond)
	if dir == Right {
		t.Errorf("BestMoveIterative picked Right (dead-end pocket), expected Up or Left")
	}
}

// TestOrderedMoves_PVFirst: PV move is always first in the ordering.
func TestOrderedMoves_PVFirst(t *testing.T) {
	moves := orderedMoves(Right, true, [2]Direction{}, [2]bool{})
	if moves[0] != Right {
		t.Errorf("expected PV move Right first, got %v", DirectionName(moves[0]))
	}
	seen := map[Direction]bool{}
	for _, d := range moves {
		seen[d] = true
	}
	if len(seen) != 4 {
		t.Errorf("expected 4 unique directions, got %d", len(seen))
	}
}

// TestOrderedMoves_KillersBeforeDefault: killer moves come before default order.
func TestOrderedMoves_KillersBeforeDefault(t *testing.T) {
	moves := orderedMoves(Up, false, [2]Direction{Right, Left}, [2]bool{true, true})
	if moves[0] != Right {
		t.Errorf("expected killer[0] Right first, got %v", DirectionName(moves[0]))
	}
	if moves[1] != Left {
		t.Errorf("expected killer[1] Left second, got %v", DirectionName(moves[1]))
	}
}

// TestOrderedMoves_PVAndKillers: PV first, then killers, then rest.
func TestOrderedMoves_PVAndKillers(t *testing.T) {
	moves := orderedMoves(Left, true, [2]Direction{Right, Down}, [2]bool{true, true})
	if moves[0] != Left {
		t.Errorf("expected PV Left first, got %v", DirectionName(moves[0]))
	}
	if moves[1] != Right {
		t.Errorf("expected killer Right second, got %v", DirectionName(moves[1]))
	}
	if moves[2] != Down {
		t.Errorf("expected killer Down third, got %v", DirectionName(moves[2]))
	}
	if moves[3] != Up {
		t.Errorf("expected Up last, got %v", DirectionName(moves[3]))
	}
}

// TestOrderedMoves_NoPVNoKillers: falls back to default AllDirections order.
func TestOrderedMoves_NoPVNoKillers(t *testing.T) {
	moves := orderedMoves(Up, false, [2]Direction{}, [2]bool{})
	if moves != AllDirections {
		t.Errorf("expected default AllDirections order, got %v", moves)
	}
}

// TestBestMoveIterative_PVOrdering: iterative deepening with PV ordering.
func TestBestMoveIterative_PVOrdering(t *testing.T) {
	me := makeSnake("me", []Coord{{2, 0}, {2, 1}, {2, 2}, {2, 3}})
	opp := makeSnake("opp", []Coord{{1, 0}, {0, 0}, {0, 1}, {0, 2}, {0, 3}})
	g := NewGameSim(5, 5, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMoveIterative("me", 300*time.Millisecond)
	if dir != Right {
		t.Errorf("BestMoveIterative with PV ordering expected Right, got %v", DirectionName(dir))
	}
}

// --- BRS Tests ---

func TestBRS_DeadEndAvoidance(t *testing.T) {
	me := makeSnake("me", []Coord{{2, 2}, {2, 1}})
	opp := makeSnake("opp", []Coord{
		{4, 2}, {4, 1}, {4, 0}, {3, 0}, {3, 1}, {3, 2}, {3, 3}, {3, 4}, {4, 4}, {4, 3},
	})
	g := NewGameSim(5, 5, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMoveIterative("me", 300*time.Millisecond)
	if dir == Right {
		t.Errorf("BRS picked Right (dead-end pocket), expected Up or Left")
	}
}

func TestBRS_HeadToHeadKill(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}})
	opp := makeSnake("opp", []Coord{{5, 7}, {5, 8}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMoveIterative("me", 300*time.Millisecond)
	if dir != Up {
		t.Errorf("BRS expected Up (head-to-head kill), got %v", DirectionName(dir))
	}
}

func TestBRS_NoOpponent(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}})
	g := NewGameSim(11, 11, []SimSnake{me}, nil, nil)

	dir := g.BestMoveIterative("me", 200*time.Millisecond)
	valid := dir == Up || dir == Down || dir == Left || dir == Right
	if !valid {
		t.Errorf("BRS no-opponent returned invalid direction %v", dir)
	}
}

func TestBRS_TinyBudget(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}})
	opp := makeSnake("opp", []Coord{{3, 5}, {3, 6}, {3, 7}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMoveIterative("me", 1*time.Millisecond)
	valid := dir == Up || dir == Down || dir == Left || dir == Right
	if !valid {
		t.Errorf("BRS tiny budget returned invalid direction %v", dir)
	}
}

func TestBRS_DepthComparison(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}})
	opp := makeSnake("opp", []Coord{{3, 5}, {3, 6}, {3, 7}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	for _, budget := range []time.Duration{
		5 * time.Millisecond,
		50 * time.Millisecond,
		200 * time.Millisecond,
		500 * time.Millisecond,
	} {
		dir := g.BestMoveIterative("me", budget)
		valid := dir == Up || dir == Down || dir == Left || dir == Right
		if !valid {
			t.Errorf("BRS budget=%v returned invalid direction %v", budget, dir)
		}
	}
}

func TestBRS_ReturnsValidMove(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}})
	opp := makeSnake("opp", []Coord{{3, 5}, {3, 6}, {3, 7}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMoveIterative("me", 500*time.Millisecond)
	valid := dir == Up || dir == Down || dir == Left || dir == Right
	if !valid {
		t.Errorf("BRS returned invalid direction %v", dir)
	}
}

// --- Quiescence Search Tests ---

func TestIsQuiet_Volatile(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}})
	opp := makeSnake("opp", []Coord{{5, 6}, {5, 7}, {5, 8}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	if isQuiet(g, 0, 1) {
		t.Error("expected volatile (heads dist=1), got quiet")
	}
}

func TestIsQuiet_Calm(t *testing.T) {
	me := makeSnake("me", []Coord{{1, 1}, {1, 0}})
	opp := makeSnake("opp", []Coord{{9, 9}, {9, 8}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	if !isQuiet(g, 0, 1) {
		t.Error("expected quiet (heads far apart), got volatile")
	}
}

func TestIsQuiet_TrappedSnake(t *testing.T) {
	me := makeSnake("me", []Coord{{9, 9}, {9, 8}})
	opp := makeSnake("opp", []Coord{{0, 0}, {0, 1}, {1, 0}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	if isQuiet(g, 0, 1) {
		t.Error("expected volatile (opp has 0 safe moves), got quiet")
	}
}

func TestQS_HeadToHeadKill(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}})
	opp := makeSnake("opp", []Coord{{5, 7}, {5, 8}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMoveIterative("me", 300*time.Millisecond)
	if dir != Up {
		t.Errorf("QS tactical test: expected Up (h2h kill), got %v", DirectionName(dir))
	}
}

// TestEvaluate_MeDead: returns -1000 when our snake is eliminated.
func TestEvaluate_MeDead(t *testing.T) {
	me := SimSnake{ID: "me", Body: []Coord{{0, 0}}, Health: 0, Length: 1, EliminatedCause: "starvation"}
	opp := makeSnake("opp", []Coord{{5, 5}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	score := Evaluate(g, 0)
	if score != -1000 {
		t.Errorf("expected -1000 when me is dead, got %v", score)
	}
}

// TestEvaluate_AllOppsDead: returns +1000 when all opponents are eliminated.
func TestEvaluate_AllOppsDead(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}})
	opp := SimSnake{ID: "opp", Body: []Coord{{0, 0}}, Health: 0, Length: 1, EliminatedCause: "wall"}
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	score := Evaluate(g, 0)
	if score != 1000 {
		t.Errorf("expected +1000 when all opponents dead, got %v", score)
	}
}
