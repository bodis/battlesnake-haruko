package logic

import "testing"

// helper: build a minimal 2-snake GameSim on an 11x11 board.
func twoSnakeGame(me, opp SimSnake) *GameSim {
	return &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{me, opp},
	}
}

func TestEval_LengthAdvantage(t *testing.T) {
	// Symmetric positions, only difference is length.
	g1 := twoSnakeGame(
		makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}}),
		makeSnake("opp", []Coord{{5, 9}, {5, 10}}),
	)
	g2 := twoSnakeGame(
		makeSnake("me", []Coord{{5, 9}, {5, 10}}),
		makeSnake("opp", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}}),
	)

	scoreLonger := Evaluate(g1, "me")
	scoreShorter := Evaluate(g2, "me")

	if scoreLonger <= scoreShorter {
		t.Errorf("longer snake should score higher: longer=%f shorter=%f", scoreLonger, scoreShorter)
	}
}

func TestEval_HeadToHeadLonger(t *testing.T) {
	// We're longer and adjacent to opponent head (dist=1).
	g := twoSnakeGame(
		makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}}),
		makeSnake("opp", []Coord{{5, 6}, {5, 7}}),
	)

	// Same but far apart.
	gFar := twoSnakeGame(
		makeSnake("me", []Coord{{0, 0}, {0, 1}, {0, 2}, {0, 3}}),
		makeSnake("opp", []Coord{{10, 10}, {10, 9}}),
	)

	scoreNear := Evaluate(g, "me")
	scoreFar := Evaluate(gFar, "me")

	if scoreNear <= scoreFar {
		t.Errorf("adjacent longer snake should score higher: near=%f far=%f", scoreNear, scoreFar)
	}
}

func TestEval_HeadToHeadShorter(t *testing.T) {
	// We're shorter and adjacent to opponent head (dist=1).
	g := twoSnakeGame(
		makeSnake("me", []Coord{{5, 5}, {5, 4}}),
		makeSnake("opp", []Coord{{5, 6}, {5, 7}, {5, 8}, {5, 9}}),
	)

	// Same but far apart.
	gFar := twoSnakeGame(
		makeSnake("me", []Coord{{0, 0}, {0, 1}}),
		makeSnake("opp", []Coord{{10, 10}, {10, 9}, {10, 8}, {10, 7}}),
	)

	scoreNear := Evaluate(g, "me")
	scoreFar := Evaluate(gFar, "me")

	if scoreNear >= scoreFar {
		t.Errorf("adjacent shorter snake should score lower: near=%f far=%f", scoreNear, scoreFar)
	}
}

func TestEval_OpponentTrapped(t *testing.T) {
	// Opponent in corner (0,0) completely surrounded by our body.
	g := twoSnakeGame(
		makeSnake("me", []Coord{{2, 2}, {1, 2}, {0, 2}, {0, 1}, {1, 0}, {1, 1}}),
		makeSnake("opp", []Coord{{0, 0}, {0, 0}}), // stacked tail
	)

	// Compare to opponent with freedom (center of board).
	gFree := twoSnakeGame(
		makeSnake("me", []Coord{{2, 2}, {2, 1}, {2, 0}, {3, 0}, {4, 0}, {5, 0}}),
		makeSnake("opp", []Coord{{8, 8}, {8, 7}}),
	)

	scoreTrapped := Evaluate(g, "me")
	scoreFree := Evaluate(gFree, "me")

	if scoreTrapped <= scoreFree {
		t.Errorf("trapped opponent should give higher score: trapped=%f free=%f", scoreTrapped, scoreFree)
	}
}

func TestEval_OpponentNearlyTrapped(t *testing.T) {
	// Opponent at wall with only 1 safe move.
	// Opp head at (0,5), body goes right. Wall blocks left, our body blocks up and down.
	g := twoSnakeGame(
		makeSnake("me", []Coord{{0, 6}, {0, 7}, {0, 4}, {0, 3}, {1, 3}}),
		makeSnake("opp", []Coord{{0, 5}, {1, 5}}),
	)

	// Opponent with full freedom.
	gFree := twoSnakeGame(
		makeSnake("me", []Coord{{0, 0}, {0, 1}, {0, 2}, {0, 3}, {0, 4}}),
		makeSnake("opp", []Coord{{5, 5}, {5, 4}}),
	)

	scoreNearlyTrapped := Evaluate(g, "me")
	scoreFree := Evaluate(gFree, "me")

	if scoreNearlyTrapped <= scoreFree {
		t.Errorf("nearly trapped opponent should give higher score: nearlyTrapped=%f free=%f",
			scoreNearlyTrapped, scoreFree)
	}
}

func TestSafeMoveCount_Corner(t *testing.T) {
	// Snake head in corner (0,0) — only Up and Right are in-bounds.
	s := makeSnake("s", []Coord{{0, 0}, {1, 0}})
	g := &GameSim{Width: 11, Height: 11, Snakes: []SimSnake{s}}
	got := safeMoveCount(g, &g.Snakes[0])
	// Down (-1 Y) and Left (-1 X) are walls; Right (1,0) is own body seg 1.
	// Only Up (0,1) is safe.
	if got != 1 {
		t.Errorf("corner snake: expected 1 safe move, got %d", got)
	}
}

func TestSafeMoveCount_Open(t *testing.T) {
	// Snake head in center, no obstacles.
	s := makeSnake("s", []Coord{{5, 5}, {5, 4}})
	g := &GameSim{Width: 11, Height: 11, Snakes: []SimSnake{s}}
	got := safeMoveCount(g, &g.Snakes[0])
	// Up, Left, Right are open. Down (5,4) is body seg 1.
	if got != 3 {
		t.Errorf("open center: expected 3 safe moves, got %d", got)
	}
}

func TestSafeMoveCount_BodyBlocked(t *testing.T) {
	// Snake surrounded by another snake's body.
	me := makeSnake("me", []Coord{{2, 2}, {2, 1}})
	opp := makeSnake("opp", []Coord{{3, 3}, {2, 3}, {1, 2}, {3, 2}})
	g := &GameSim{Width: 11, Height: 11, Snakes: []SimSnake{me, opp}}
	got := safeMoveCount(g, &g.Snakes[0])
	// Up (2,3) blocked by opp seg 1; Left (1,2) blocked by opp seg 2;
	// Right (3,2) blocked by opp seg 3; Down (2,1) own body seg 1.
	if got != 0 {
		t.Errorf("body-blocked: expected 0 safe moves, got %d", got)
	}
}

func TestEval_ThreeSnakes(t *testing.T) {
	// 3-snake game: me is longer than both opponents, near one.
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}})
	opp1 := makeSnake("opp1", []Coord{{5, 7}, {5, 8}})              // near, shorter
	opp2 := makeSnake("opp2", []Coord{{10, 10}, {10, 9}, {10, 8}}) // far, shorter
	g := &GameSim{Width: 11, Height: 11, Snakes: []SimSnake{me, opp1, opp2}}

	score := Evaluate(g, "me")
	// Should be positive: length advantage over both + H2H bonus vs opp1
	if score <= 0 {
		t.Errorf("3-snake eval: expected positive score when longer than all, got %f", score)
	}

	// Now make me shorter than both.
	me2 := makeSnake("me", []Coord{{5, 5}, {5, 4}})
	opp1b := makeSnake("opp1", []Coord{{5, 7}, {5, 8}, {5, 9}, {5, 10}})
	opp2b := makeSnake("opp2", []Coord{{0, 0}, {0, 1}, {0, 2}, {0, 3}})
	g2 := &GameSim{Width: 11, Height: 11, Snakes: []SimSnake{me2, opp1b, opp2b}}

	score2 := Evaluate(g2, "me")
	if score2 >= score {
		t.Errorf("3-snake eval: shorter-than-all (%f) should score lower than longer-than-all (%f)", score2, score)
	}
}

func TestEval_DeadSnakeStillNeg1000(t *testing.T) {
	me := SimSnake{ID: "me", Body: []Coord{{5, 5}}, Health: 0, Length: 1, EliminatedCause: "starvation"}
	opp := makeSnake("opp", []Coord{{3, 3}, {3, 2}, {3, 1}})
	g := twoSnakeGame(me, opp)

	score := Evaluate(g, "me")
	if score != -1000 {
		t.Errorf("dead snake should score -1000, got %f", score)
	}
}
