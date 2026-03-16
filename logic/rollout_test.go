package logic

import (
	"testing"
	"time"
)

func TestMCRolloutSmoke(t *testing.T) {
	// 7x7 board with two snakes.
	g := NewGameSim(7, 7, []SimSnake{
		{ID: "me", Body: []Coord{{3, 3}, {3, 2}, {3, 1}}, Health: 100, Length: 3},
		{ID: "opp", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 100, Length: 3},
	}, []Coord{{1, 1}, {6, 6}}, nil)

	result := MCRolloutTimed(g, 0, 1, 100, 20*time.Millisecond)

	totalRollouts := 0
	for d := Direction(0); d < 4; d++ {
		s := result.Stats[d]
		if result.Valid[d] {
			if s.Total == 0 {
				t.Errorf("direction %d is valid but has 0 rollouts", d)
			}
			if s.SurvivalRate < 0 || s.SurvivalRate > 1 {
				t.Errorf("direction %d survival rate %.2f not in [0,1]", d, s.SurvivalRate)
			}
			if s.Survived > s.Total {
				t.Errorf("direction %d survived (%d) > total (%d)", d, s.Survived, s.Total)
			}
			totalRollouts += s.Total
		}
	}
	if totalRollouts < 10 {
		t.Errorf("expected at least 10 total rollouts, got %d", totalRollouts)
	}
	t.Logf("total rollouts: %d", totalRollouts)
	for d := Direction(0); d < 4; d++ {
		if result.Valid[d] {
			t.Logf("dir %d: %d/%d = %.3f", d, result.Stats[d].Survived, result.Stats[d].Total, result.Stats[d].SurvivalRate)
		}
	}
}

func TestRandomSafeMove(t *testing.T) {
	// Snake in corner — only 2 safe directions (up, right from 0,0).
	g := NewGameSim(7, 7, []SimSnake{
		{ID: "me", Body: []Coord{{0, 0}, {1, 0}, {2, 0}}, Health: 100, Length: 3},
	}, nil, nil)

	rng := xorshift64(12345)
	counts := [4]int{}
	for i := 0; i < 1000; i++ {
		d := randomSafeMove(g, 0, &rng)
		// Only Up (0,1) and Left is wall, Down is wall, Right is body.
		// Actually from (0,0): Up→(0,1) safe, Down→(0,-1) wall, Left→(-1,0) wall, Right→(1,0) body.
		counts[d]++
	}

	// Down and Left should be 0 (wall), Right should be 0 (body at 1,0).
	if counts[Down] > 0 {
		t.Errorf("randomSafeMove returned Down (wall) %d times", counts[Down])
	}
	if counts[Left] > 0 {
		t.Errorf("randomSafeMove returned Left (wall) %d times", counts[Left])
	}
	if counts[Right] > 0 {
		t.Errorf("randomSafeMove returned Right (body) %d times", counts[Right])
	}
	if counts[Up] == 0 {
		t.Error("randomSafeMove never returned Up (only safe direction)")
	}
}

func TestApplyMCBiasNoSignal(t *testing.T) {
	// When survival spread < 0.15, BRS result should be unchanged.
	result := BRSResult{
		Scores:   [4]float64{10, 20, 15, 5},
		HasScore: [4]bool{true, true, true, true},
		BestDir:  Down,
	}
	mc := MCRolloutResult{
		Stats: [4]RolloutStats{
			{Survived: 80, Total: 100, SurvivalRate: 0.80},
			{Survived: 82, Total: 100, SurvivalRate: 0.82},
			{Survived: 79, Total: 100, SurvivalRate: 0.79},
			{Survived: 81, Total: 100, SurvivalRate: 0.81},
		},
		Valid: [4]bool{true, true, true, true},
	}
	dir := applyMCBias(result, mc, 1.0) // late game
	if dir != Down {
		t.Errorf("expected BRS direction Down (unchanged), got %d", dir)
	}
}

func TestApplyMCBiasEarlyGameNoEffect(t *testing.T) {
	// Even with large spread, MC should not override in early game.
	result := BRSResult{
		Scores:   [4]float64{10, 12, 10, 10},
		HasScore: [4]bool{true, true, true, true},
		BestDir:  Down,
	}
	mc := MCRolloutResult{
		Stats: [4]RolloutStats{
			{Survived: 90, Total: 100, SurvivalRate: 0.90},
			{Survived: 30, Total: 100, SurvivalRate: 0.30},
			{Survived: 50, Total: 100, SurvivalRate: 0.50},
			{Survived: 50, Total: 100, SurvivalRate: 0.50},
		},
		Valid: [4]bool{true, true, true, true},
	}
	dir := applyMCBias(result, mc, 0.0) // early game — MC should be ignored
	if dir != Down {
		t.Errorf("expected BRS direction Down (unchanged in early game), got %d", dir)
	}
}

func TestApplyMCBiasOverride(t *testing.T) {
	// When survival spread is large and MC strongly favors a direction,
	// it should override BRS when scores are close (late game).
	result := BRSResult{
		Scores:   [4]float64{10, 12, 10, 10},
		HasScore: [4]bool{true, true, true, true},
		BestDir:  Down, // BRS picks Down (score 12)
	}
	mc := MCRolloutResult{
		Stats: [4]RolloutStats{
			{Survived: 90, Total: 100, SurvivalRate: 0.90}, // Up: high survival
			{Survived: 30, Total: 100, SurvivalRate: 0.30}, // Down: low survival
			{Survived: 50, Total: 100, SurvivalRate: 0.50},
			{Survived: 50, Total: 100, SurvivalRate: 0.50},
		},
		Valid: [4]bool{true, true, true, true},
	}
	dir := applyMCBias(result, mc, 1.0) // late game — MC should override
	if dir != Up {
		t.Errorf("expected MC to override to Up, got %d", dir)
	}
}

func BenchmarkMCRollout(b *testing.B) {
	// 11x11 mid-game position.
	g := NewGameSim(11, 11, []SimSnake{
		{ID: "me", Body: []Coord{{5, 5}, {5, 4}, {5, 3}, {5, 2}, {5, 1}, {4, 1}, {3, 1}}, Health: 80, Length: 7},
		{ID: "opp", Body: []Coord{{8, 8}, {8, 7}, {8, 6}, {8, 5}, {8, 4}, {7, 4}}, Health: 75, Length: 6},
	}, []Coord{{2, 2}, {9, 9}, {0, 5}}, nil)

	// Ensure Zobrist is initialized.
	_ = g.Hash()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MCRolloutTimed(g, 0, 1, 100, 20*time.Millisecond)
	}
}
