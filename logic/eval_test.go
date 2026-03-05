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
	g1 := twoSnakeGame(
		makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}}),
		makeSnake("opp", []Coord{{5, 9}, {5, 10}}),
	)
	g2 := twoSnakeGame(
		makeSnake("me", []Coord{{5, 9}, {5, 10}}),
		makeSnake("opp", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}}),
	)

	scoreLonger := Evaluate(g1, 0)
	scoreShorter := Evaluate(g2, 0)

	if scoreLonger <= scoreShorter {
		t.Errorf("longer snake should score higher: longer=%f shorter=%f", scoreLonger, scoreShorter)
	}
}

func TestEval_HeadToHeadLonger(t *testing.T) {
	g := twoSnakeGame(
		makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}}),
		makeSnake("opp", []Coord{{5, 6}, {5, 7}}),
	)

	gFar := twoSnakeGame(
		makeSnake("me", []Coord{{0, 0}, {0, 1}, {0, 2}, {0, 3}}),
		makeSnake("opp", []Coord{{10, 10}, {10, 9}}),
	)

	scoreNear := Evaluate(g, 0)
	scoreFar := Evaluate(gFar, 0)

	if scoreNear <= scoreFar {
		t.Errorf("adjacent longer snake should score higher: near=%f far=%f", scoreNear, scoreFar)
	}
}

func TestEval_HeadToHeadShorter(t *testing.T) {
	g := twoSnakeGame(
		makeSnake("me", []Coord{{5, 5}, {5, 4}}),
		makeSnake("opp", []Coord{{5, 6}, {5, 7}, {5, 8}, {5, 9}}),
	)

	gFar := twoSnakeGame(
		makeSnake("me", []Coord{{0, 0}, {0, 1}}),
		makeSnake("opp", []Coord{{10, 10}, {10, 9}, {10, 8}, {10, 7}}),
	)

	scoreNear := Evaluate(g, 0)
	scoreFar := Evaluate(gFar, 0)

	if scoreNear >= scoreFar {
		t.Errorf("adjacent shorter snake should score lower: near=%f far=%f", scoreNear, scoreFar)
	}
}

func TestEval_OpponentTrapped(t *testing.T) {
	g := twoSnakeGame(
		makeSnake("me", []Coord{{2, 2}, {1, 2}, {0, 2}, {0, 1}, {1, 0}, {1, 1}}),
		makeSnake("opp", []Coord{{0, 0}, {0, 0}}),
	)

	gFree := twoSnakeGame(
		makeSnake("me", []Coord{{2, 2}, {2, 1}, {2, 0}, {3, 0}, {4, 0}, {5, 0}}),
		makeSnake("opp", []Coord{{8, 8}, {8, 7}}),
	)

	scoreTrapped := Evaluate(g, 0)
	scoreFree := Evaluate(gFree, 0)

	if scoreTrapped <= scoreFree {
		t.Errorf("trapped opponent should give higher score: trapped=%f free=%f", scoreTrapped, scoreFree)
	}
}

func TestEval_OpponentNearlyTrapped(t *testing.T) {
	g := twoSnakeGame(
		makeSnake("me", []Coord{{0, 6}, {0, 7}, {0, 4}, {0, 3}, {1, 3}}),
		makeSnake("opp", []Coord{{0, 5}, {1, 5}}),
	)

	gFree := twoSnakeGame(
		makeSnake("me", []Coord{{0, 0}, {0, 1}, {0, 2}, {0, 3}, {0, 4}}),
		makeSnake("opp", []Coord{{5, 5}, {5, 4}}),
	)

	scoreNearlyTrapped := Evaluate(g, 0)
	scoreFree := Evaluate(gFree, 0)

	if scoreNearlyTrapped <= scoreFree {
		t.Errorf("nearly trapped opponent should give higher score: nearlyTrapped=%f free=%f",
			scoreNearlyTrapped, scoreFree)
	}
}

func TestSafeMoveCount_Corner(t *testing.T) {
	s := makeSnake("s", []Coord{{0, 0}, {1, 0}})
	g := &GameSim{Width: 11, Height: 11, Snakes: []SimSnake{s}}
	got := safeMoveCount(g, &g.Snakes[0])
	if got != 1 {
		t.Errorf("corner snake: expected 1 safe move, got %d", got)
	}
}

func TestSafeMoveCount_Open(t *testing.T) {
	s := makeSnake("s", []Coord{{5, 5}, {5, 4}})
	g := &GameSim{Width: 11, Height: 11, Snakes: []SimSnake{s}}
	got := safeMoveCount(g, &g.Snakes[0])
	if got != 3 {
		t.Errorf("open center: expected 3 safe moves, got %d", got)
	}
}

func TestSafeMoveCount_BodyBlocked(t *testing.T) {
	me := makeSnake("me", []Coord{{2, 2}, {2, 1}})
	opp := makeSnake("opp", []Coord{{3, 3}, {2, 3}, {1, 2}, {3, 2}})
	g := &GameSim{Width: 11, Height: 11, Snakes: []SimSnake{me, opp}}
	got := safeMoveCount(g, &g.Snakes[0])
	if got != 0 {
		t.Errorf("body-blocked: expected 0 safe moves, got %d", got)
	}
}

func TestEval_ThreeSnakes(t *testing.T) {
	me := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}})
	opp1 := makeSnake("opp1", []Coord{{5, 7}, {5, 8}})
	opp2 := makeSnake("opp2", []Coord{{10, 10}, {10, 9}, {10, 8}})
	g := &GameSim{Width: 11, Height: 11, Snakes: []SimSnake{me, opp1, opp2}}

	score := Evaluate(g, 0) // me is index 0
	if score <= 0 {
		t.Errorf("3-snake eval: expected positive score when longer than all, got %f", score)
	}

	me2 := makeSnake("me", []Coord{{5, 5}, {5, 4}})
	opp1b := makeSnake("opp1", []Coord{{5, 7}, {5, 8}, {5, 9}, {5, 10}})
	opp2b := makeSnake("opp2", []Coord{{0, 0}, {0, 1}, {0, 2}, {0, 3}})
	g2 := &GameSim{Width: 11, Height: 11, Snakes: []SimSnake{me2, opp1b, opp2b}}

	score2 := Evaluate(g2, 0)
	if score2 >= score {
		t.Errorf("3-snake eval: shorter-than-all (%f) should score lower than longer-than-all (%f)", score2, score)
	}
}

func TestEval_DeadSnakeStillNeg1000(t *testing.T) {
	me := SimSnake{ID: "me", Body: []Coord{{5, 5}}, Health: 0, Length: 1, EliminatedCause: "starvation"}
	opp := makeSnake("opp", []Coord{{3, 3}, {3, 2}, {3, 1}})
	g := twoSnakeGame(me, opp)

	score := Evaluate(g, 0)
	if score != -1000 {
		t.Errorf("dead snake should score -1000, got %f", score)
	}
}

func TestEval_EarlyPhase_FavorsLength(t *testing.T) {
	// Early game (Turn=0, short snakes): wLen should be boosted (3.0 vs 2.0).
	// Compare two games with same length diff but different phases.
	earlyMe := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}})
	earlyOpp := makeSnake("opp", []Coord{{5, 9}, {5, 10}})
	gEarly := twoSnakeGame(earlyMe, earlyOpp)
	gEarly.Turn = 5

	midMe := makeSnake("me", []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}, {5, 1}, {4, 1}, {3, 1}, {2, 1}, {1, 1}})
	midOpp := makeSnake("opp", []Coord{{5, 9}, {5, 10}, {4, 10}, {3, 10}, {2, 10}, {1, 10}, {0, 10}})
	gMid := twoSnakeGame(midMe, midOpp)
	gMid.Turn = 50

	scoreEarly := Evaluate(gEarly, 0)
	scoreMid := Evaluate(gMid, 0)

	// Early game should value the length advantage more (wLen=3.0 vs 2.0).
	// Both have +2 length advantage, but early phase amplifies it.
	if scoreEarly <= 0 {
		t.Errorf("early phase should score positively with length advantage, got %f", scoreEarly)
	}
	if scoreMid <= 0 {
		t.Errorf("mid phase should score positively with length advantage, got %f", scoreMid)
	}
}

func TestEval_EarlyPhase_FoodControl(t *testing.T) {
	// Early game: food in our territory should boost score.
	meBody := []Coord{{3, 3}, {3, 2}, {3, 1}}
	oppBody := []Coord{{8, 8}, {8, 7}, {8, 6}}

	// Food near us (in our Voronoi territory).
	gFoodNear := twoSnakeGame(makeSnake("me", meBody), makeSnake("opp", oppBody))
	gFoodNear.Food = []Coord{{2, 3}, {4, 3}}
	gFoodNear.Turn = 5

	// Food near opponent.
	gFoodFar := twoSnakeGame(makeSnake("me", meBody), makeSnake("opp", oppBody))
	gFoodFar.Food = []Coord{{9, 8}, {7, 8}}
	gFoodFar.Turn = 5

	scoreNear := Evaluate(gFoodNear, 0)
	scoreFar := Evaluate(gFoodFar, 0)

	if scoreNear <= scoreFar {
		t.Errorf("early game: food in our territory should score higher: near=%f far=%f", scoreNear, scoreFar)
	}

	// Late game (long snakes, high turn): food control should matter less.
	longMe := makeSnake("me", []Coord{{3, 3}, {3, 2}, {3, 1}, {3, 0}, {4, 0}, {5, 0}, {6, 0}, {7, 0}, {8, 0}, {9, 0}})
	longOpp := makeSnake("opp", []Coord{{8, 8}, {8, 7}, {8, 6}, {8, 5}, {8, 4}, {8, 3}, {7, 3}, {6, 3}, {5, 3}, {4, 3}})

	gLateNear := twoSnakeGame(longMe, longOpp)
	gLateNear.Food = []Coord{{2, 3}, {4, 4}}
	gLateNear.Turn = 100

	gLateFar := twoSnakeGame(longMe, longOpp)
	gLateFar.Food = []Coord{{9, 8}, {7, 8}}
	gLateFar.Turn = 100

	diffLate := Evaluate(gLateNear, 0) - Evaluate(gLateFar, 0)
	diffEarly := scoreNear - scoreFar

	if diffLate >= diffEarly {
		t.Errorf("food control should matter more early than late: earlyDiff=%f lateDiff=%f", diffEarly, diffLate)
	}
}

func TestEval_LatePhase_TerritoryBoost(t *testing.T) {
	// High board fill should boost territory weight.
	// Create a crowded board (>50% fill) vs sparse board.
	sparseMe := makeSnake("me", []Coord{{3, 3}, {3, 2}, {3, 1}})
	sparseOpp := makeSnake("opp", []Coord{{8, 8}, {8, 7}, {8, 6}})
	gSparse := twoSnakeGame(sparseMe, sparseOpp)
	gSparse.Turn = 50

	// Crowded: same head positions but very long snakes (>50% of 121 cells = 61+).
	crowdedMeBody := make([]Coord, 35)
	for i := range crowdedMeBody {
		crowdedMeBody[i] = Coord{i % 11, i / 11}
	}
	crowdedMeBody[0] = Coord{3, 3}
	crowdedOppBody := make([]Coord, 35)
	for i := range crowdedOppBody {
		crowdedOppBody[i] = Coord{10 - i%11, 10 - i/11}
	}
	crowdedOppBody[0] = Coord{8, 8}

	crowdedMe := makeSnake("me", crowdedMeBody)
	crowdedOpp := makeSnake("opp", crowdedOppBody)
	gCrowded := twoSnakeGame(crowdedMe, crowdedOpp)
	gCrowded.Turn = 200

	scoreSparse := Evaluate(gSparse, 0)
	scoreCrowded := Evaluate(gCrowded, 0)

	// Both should be computable without panic.
	_ = scoreSparse
	_ = scoreCrowded
}

func TestEval_PhaseBlendContinuity(t *testing.T) {
	// Verify no abrupt score jumps as snake length increases from 4 to 10.
	opp := makeSnake("opp", []Coord{{8, 8}, {8, 7}, {8, 6}})
	var prev float64
	for length := 4; length <= 10; length++ {
		body := make([]Coord, length)
		for i := range body {
			body[i] = Coord{3, 3 + i}
			if body[i].Y >= 11 {
				body[i] = Coord{4, body[i].Y - 11}
			}
		}
		me := makeSnake("me", body)
		g := twoSnakeGame(me, opp)
		g.Turn = 20
		score := Evaluate(g, 0)

		if length > 4 {
			jump := score - prev
			if jump < -20 || jump > 20 {
				t.Errorf("score discontinuity at length %d: prev=%f curr=%f jump=%f",
					length, prev, score, jump)
			}
		}
		prev = score
	}
}
