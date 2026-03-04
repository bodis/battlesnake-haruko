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

func TestEval_DeadSnakeStillNeg1000(t *testing.T) {
	me := SimSnake{ID: "me", Body: []Coord{{5, 5}}, Health: 0, Length: 1, EliminatedCause: "starvation"}
	opp := makeSnake("opp", []Coord{{3, 3}, {3, 2}, {3, 1}})
	g := twoSnakeGame(me, opp)

	score := Evaluate(g, "me")
	if score != -1000 {
		t.Errorf("dead snake should score -1000, got %f", score)
	}
}
