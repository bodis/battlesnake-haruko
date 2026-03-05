package logic

import (
	"testing"
)

func TestVoronoiSymmetricBoard(t *testing.T) {
	g := &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}, {1, 0}, {2, 0}}, Health: 100, Length: 3},
			{ID: "b", Body: []Coord{{10, 10}, {9, 10}, {8, 10}}, Health: 100, Length: 3},
		},
	}
	my, opp := VoronoiTerritory(g, 0) // "a" is index 0
	if my == 0 || opp == 0 {
		t.Fatalf("expected non-zero territory, got my=%d opp=%d", my, opp)
	}
	diff := my - opp
	if diff < 0 {
		diff = -diff
	}
	if diff > 5 {
		t.Errorf("expected roughly equal territory, got my=%d opp=%d (diff=%d)", my, opp, diff)
	}
}

func TestVoronoiCorneredSnake(t *testing.T) {
	g := &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}, {1, 0}, {2, 0}}, Health: 100, Length: 3},
			{ID: "b", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 100, Length: 3},
		},
	}
	my, opp := VoronoiTerritory(g, 0)
	if my >= opp {
		t.Errorf("cornered snake should have less territory: my=%d opp=%d", my, opp)
	}
}

func TestVoronoiBodyWallPartition(t *testing.T) {
	g := &GameSim{
		Width:  7,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 2}, {0, 1}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{5, 2}, {2, 4}, {2, 3}, {2, 2}, {2, 1}, {2, 0}, {3, 0}}, Health: 100, Length: 7},
		},
	}
	my, opp := VoronoiTerritory(g, 0)
	if my >= opp {
		t.Errorf("wall-partitioned snake should have less territory: my=%d opp=%d", my, opp)
	}
}

func TestVoronoiDeadSnakeIgnored(t *testing.T) {
	g := &GameSim{
		Width:  5,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{4, 4}, {3, 4}}, Health: 0, Length: 2, EliminatedCause: "starvation"},
		},
	}
	my, opp := VoronoiTerritory(g, 0)
	if opp != 0 {
		t.Errorf("dead snake should claim no territory, got opp=%d", opp)
	}
	if my == 0 {
		t.Errorf("alive snake should claim territory, got my=0")
	}
}

func TestVoronoiSingleSnake(t *testing.T) {
	g := &GameSim{
		Width:  5,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{2, 2}, {2, 1}, {2, 0}}, Health: 100, Length: 3},
		},
	}
	my, opp := VoronoiTerritory(g, 0)
	if my != 24 {
		t.Errorf("single snake should claim 24 cells (25 - 1 body), got my=%d", my)
	}
	if opp != 0 {
		t.Errorf("no opponents, expected opp=0, got opp=%d", opp)
	}
}
