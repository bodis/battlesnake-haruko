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
	// 5x5 board. Our snake head at (0,2), neck at (0,1).
	// Left is wall (x=-1). Down (0,1) is our neck.
	// Up (0,3) and Right (1,2) are open, but we build a wall of opponent
	// body that blocks up except for a 1-cell gap, making Right clearly better.
	//
	// Simpler setup: head at (0,4) neck at (0,3). Only Up is wall, Left is wall.
	// Down (0,3) is neck — illegal. Right (1,4) is open wide.
	// We wall off everything except Right and a tiny pocket at (0,4) going Up...
	// Actually let's use a clean, easy scenario:
	//
	// 3x3 board, our snake occupies the left column going up:
	//   head=(0,2), body=(0,1),(0,0)
	// We have a wall of opponent body forming a barrier:
	//   opp body at (1,2),(1,1),(1,0) — entire middle column blocked.
	// Safe moves from (0,2): Up → (0,3) out of bounds, Down → (0,1) own body,
	//   Left → (-1,2) wall, Right → (1,2) blocked by opp body.
	// That's fully trapped — not a good test. Let's do 5x3 instead.
	//
	// 5x3 board. Our head=(0,1), neck=(0,0).
	// Opponent body column at x=1: (1,0),(1,1),(1,2).
	// From (0,1): Up→(0,2) open (but only connects to (0,2) since right is blocked at x=1)
	//              Down→(0,0) own body (blocked)
	//              Left→(-1,1) wall
	//              Right→(1,1) opp body blocked
	// So Up reaches just 1 cell. That's still trapped.
	//
	// Use a 7x3 board with opp body forming a partial wall.
	// Our snake: head=(0,1), body=(0,0). (length 2)
	// Opp snake: head=(2,2), body=(2,1),(2,0). (length 3, blocks x=2 column)
	// From (0,1): Up→(0,2): reachable cells = (0,2),(1,2) only 2 cells
	//             Right→(1,1): reachable (1,1),(1,0),(1,2) = 3 cells but opp head at (2,2)
	//             Actually let me think more carefully.
	//
	// Simplest reliable test: large board, our snake near a wall with one clearly dominant direction.
	// 11x11 board. Our head=(0,5), neck=(0,4).
	// Opponent forms a barrier at x=1 column (cells 1..9 in y), blocking Right.
	// Up→(0,6): open flood to top half  ~5 cells
	// Down→(0,4): own body blocked
	// Left→(-1,5): wall
	// Right→(1,5): opp interior body blocked
	// Up wins with more space than the tiny left/down corner. But actually
	// Up and Left/Down situation is: up opens the top portion of x=0 column only (cells 6..10) = 5 cells,
	// but the opp body at x=1 means the entire x=0 strip is accessible (y=0..10 minus own body neck).
	// Let me just do a concrete, simple scenario without over-thinking it.

	// Simple: 5x5 board. Our head at (2,2) center, neck at (2,1).
	// Opp snake head at (4,2), body filling right side: (4,2),(4,1),(4,0),(3,0),(3,1),(3,2),(3,3),(3,4),(4,4),(4,3) (forms a pocket on the right).
	// Down→(2,1) own body blocked.
	// Left→(1,2) large open area.
	// Right→(3,2) — enters the walled pocket. Only cells reachable: (3,2) then blocked by rest of opp body. Very small.
	// Up→(2,3) large open area.
	// Best moves: Up or Left (both open wide). Right should be worst.

	me := makeSnake("me", []Coord{{2, 2}, {2, 1}})
	// Opp body forms a partial cage on the right side interior.
	opp := makeSnake("opp", []Coord{
		{4, 2}, {4, 1}, {4, 0}, {3, 0}, {3, 1}, {3, 2}, {3, 3}, {3, 4}, {4, 4}, {4, 3},
	})
	g := NewGameSim(5, 5, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMove("me", 3)
	// Right leads into the pocket — must not pick Right.
	if dir == Right {
		t.Errorf("BestMove picked Right (dead-end pocket), expected Up or Left")
	}
}

// TestBestMove_NeckTrap: opponent body walls force a single escape.
func TestBestMove_NeckTrap(t *testing.T) {
	// 5x5 board. Our head=(2,2), neck=(2,3).
	// Opp body fills: (1,2),(1,1),(2,1),(3,1),(3,2) — walls below and sides.
	// Up→(2,3) is own neck (body[1]), blocked.
	// Down→(2,1) blocked by opp body.
	// Left→(1,2) blocked by opp body.
	// Right→(3,2) blocked by opp body.
	// Hmm that's fully trapped again. Let me leave one escape open.
	//
	// Opp body: (1,2),(1,1),(2,1),(3,1) — right side open.
	// Up→(2,3) own neck body[1] — after step, that cell will be vacated (tail moves).
	// Actually in BestMove we call Step which moves snakes. The body is still there at step time.
	// Let's just leave Right clearly open and everything else blocked.

	me := makeSnake("me", []Coord{{2, 2}, {2, 3}, {2, 4}}) // head at (2,2), neck at (2,3)
	opp := makeSnake("opp", []Coord{{1, 2}, {1, 1}, {2, 1}, {3, 1}, {3, 2}, {3, 3}})
	// Opp body blocks Left(1,2), Down(2,1), and Right(3,2).
	// Up→(2,3) is our own body — blocked.
	// This is fully trapped (all 4 blocked). Not ideal.
	// Instead: opp only blocks Left and Down, leaving Right and Up.
	// But we want only one escape. Let's block Up too with our own body length.

	me2 := makeSnake("me", []Coord{{2, 0}, {2, 1}, {2, 2}, {2, 3}}) // head at bottom (2,0), body going up
	opp2 := makeSnake("opp", []Coord{{1, 0}, {0, 0}, {0, 1}, {0, 2}, {0, 3}})
	// From head (2,0):
	//   Down→(2,-1) wall
	//   Up→(2,1) own body blocked
	//   Left→(1,0) opp head — blocked in floodFill (enemy head)
	//   Right→(3,0) OPEN
	// Right should be the only good move.
	g2 := NewGameSim(5, 5, []SimSnake{me2, opp2}, nil, nil)
	dir2 := g2.BestMove("me", 3)
	_ = me
	_ = opp
	if dir2 != Right {
		t.Errorf("BestMove expected Right (only escape), got %v", DirectionName(dir2))
	}
}

// TestBestMove_HeadToHeadKill: we're longer, moving to a square adjacent to
// opponent's head enables a head-to-head kill — prefer it.
func TestBestMove_HeadToHeadKill(t *testing.T) {
	// 11x11 board. Our snake length 4, opp length 2.
	// If we move to same cell as opp head next turn, we win (we're longer).
	// Our head=(5,5), opp head=(5,7). If opp moves Down→(5,6) and we move Up→(5,6),
	// head-to-head occurs: we're length 4, opp length 2 → opp dies.
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}})  // length 4
	opp := makeSnake("opp", []Coord{{5, 7}, {5, 8}})                 // length 2
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	dir := g.BestMove("me", 3)
	// Moving Up puts us at (5,6). If opp moves Down to (5,6), we win.
	// The minimax should prefer Up because in the worst case (opp moves Down) we still win.
	if dir != Up {
		t.Errorf("BestMove expected Up (head-to-head kill opportunity), got %v", DirectionName(dir))
	}
}

// TestBestMove_FoodReachable: among equally-spaced moves, prefer food.
// We skip explicit food scoring in Evaluate (it's pure flood fill),
// so this test just verifies BestMove doesn't panic and returns a valid direction.
func TestBestMove_FoodReachable(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}})
	opp := makeSnake("opp", []Coord{{0, 0}, {0, 1}})
	food := []Coord{{5, 6}} // directly above our head
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
	// 3x3 board, our snake fills the entire board — all moves blocked.
	me := makeSnake("me", []Coord{
		{1, 1}, // head center
		{0, 1}, {0, 0}, {1, 0}, {2, 0}, {2, 1}, {2, 2}, {1, 2}, {0, 2},
	})
	g := NewGameSim(3, 3, []SimSnake{me}, nil, nil)

	// Should not panic.
	dir := g.BestMove("me", 3)
	valid := dir == Up || dir == Down || dir == Left || dir == Right
	if !valid {
		t.Errorf("BestMove returned invalid direction %v", dir)
	}
}

// TestBestMove_NoOpponents: with no opponents, picks the move with most space.
func TestBestMove_NoOpponents(t *testing.T) {
	// 5x5 board. Our head=(0,2), neck=(0,1).
	// Up→(0,3): connects to full top-left region (many cells).
	// Down→(0,1) own body blocked.
	// Left→(-1,2) wall.
	// Right→(1,2) open but smaller than going up along left wall.
	// With just us on board, Up and Right both open the whole remaining board.
	// Just verify no panic and a valid direction.
	me := makeSnake("me", []Coord{{0, 2}, {0, 1}, {0, 0}})
	g := NewGameSim(5, 5, []SimSnake{me}, nil, nil)

	dir := g.BestMove("me", 3)
	valid := dir == Up || dir == Down || dir == Left || dir == Right
	if !valid {
		t.Errorf("BestMove returned invalid direction %v", dir)
	}
	// Down and Left are clearly bad (wall/body); Up or Right should win.
	if dir == Down || dir == Left {
		t.Errorf("BestMove picked a clearly bad direction %v", DirectionName(dir))
	}
}

// TestBestMove_DepthComparison: deeper search should not panic and should
// return valid moves at any depth.
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

// TestBestMoveIterative_DeadEndAvoidance: same scenario as TestBestMove_DeadEndAvoidance,
// verify iterative deepening also avoids the dead-end pocket.
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

// TestEvaluate_MeDead: returns -1000 when our snake is eliminated.
func TestEvaluate_MeDead(t *testing.T) {
	me := SimSnake{ID: "me", Body: []Coord{{0, 0}}, Health: 0, Length: 1, EliminatedCause: "starvation"}
	opp := makeSnake("opp", []Coord{{5, 5}})
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	score := Evaluate(g, "me")
	if score != -1000 {
		t.Errorf("expected -1000 when me is dead, got %v", score)
	}
}

// TestEvaluate_AllOppsDead: returns +1000 when all opponents are eliminated.
func TestEvaluate_AllOppsDead(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}})
	opp := SimSnake{ID: "opp", Body: []Coord{{0, 0}}, Health: 0, Length: 1, EliminatedCause: "wall"}
	g := NewGameSim(11, 11, []SimSnake{me, opp}, nil, nil)

	score := Evaluate(g, "me")
	if score != 1000 {
		t.Errorf("expected +1000 when all opponents dead, got %v", score)
	}
}
