package logic

import (
	"math"
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
	vr := VoronoiTerritory(g, 0)
	if vr.MyTerritory == 0 || vr.OppTerritory == 0 {
		t.Fatalf("expected non-zero territory, got my=%d opp=%d", vr.MyTerritory, vr.OppTerritory)
	}
	diff := vr.MyTerritory - vr.OppTerritory
	if diff < 0 {
		diff = -diff
	}
	if diff > 5 {
		t.Errorf("expected roughly equal territory, got my=%d opp=%d (diff=%d)", vr.MyTerritory, vr.OppTerritory, diff)
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
	vr := VoronoiTerritory(g, 0)
	if vr.MyTerritory >= vr.OppTerritory {
		t.Errorf("cornered snake should have less territory: my=%d opp=%d", vr.MyTerritory, vr.OppTerritory)
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
	vr := VoronoiTerritory(g, 0)
	if vr.MyTerritory >= vr.OppTerritory {
		t.Errorf("wall-partitioned snake should have less territory: my=%d opp=%d", vr.MyTerritory, vr.OppTerritory)
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
	vr := VoronoiTerritory(g, 0)
	if vr.OppTerritory != 0 {
		t.Errorf("dead snake should claim no territory, got opp=%d", vr.OppTerritory)
	}
	if vr.MyTerritory == 0 {
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
	vr := VoronoiTerritory(g, 0)
	if vr.MyTerritory != 24 {
		t.Errorf("single snake should claim 24 cells (25 - 1 body), got my=%d", vr.MyTerritory)
	}
	if vr.OppTerritory != 0 {
		t.Errorf("no opponents, expected opp=0, got opp=%d", vr.OppTerritory)
	}
}

func TestVoronoiResult_FoodInMyTerritory(t *testing.T) {
	g := &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{1, 1}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{9, 9}, {9, 10}}, Health: 100, Length: 2},
		},
		Food: []Coord{{0, 0}, {2, 2}}, // close to snake a
	}
	vr := VoronoiTerritory(g, 0)
	if vr.MyFood != 2 {
		t.Errorf("expected MyFood=2, got %d", vr.MyFood)
	}
	if vr.OppFood != 0 {
		t.Errorf("expected OppFood=0, got %d", vr.OppFood)
	}
}

func TestVoronoiResult_FoodInOppTerritory(t *testing.T) {
	g := &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{1, 1}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{9, 9}, {9, 10}}, Health: 100, Length: 2},
		},
		Food: []Coord{{10, 10}, {8, 8}}, // close to snake b
	}
	vr := VoronoiTerritory(g, 0)
	if vr.MyFood != 0 {
		t.Errorf("expected MyFood=0, got %d", vr.MyFood)
	}
	if vr.OppFood != 2 {
		t.Errorf("expected OppFood=2, got %d", vr.OppFood)
	}
}

func TestVoronoiResult_FoodOnFrontier(t *testing.T) {
	// Place food equidistant from both snakes on a small board.
	g := &GameSim{
		Width:  5,
		Height: 1,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}}, Health: 100, Length: 1},
			{ID: "b", Body: []Coord{{4, 0}}, Health: 100, Length: 1},
		},
		Food: []Coord{{2, 0}}, // equidistant → tied → neither claims
	}
	vr := VoronoiTerritory(g, 0)
	if vr.MyFood != 0 {
		t.Errorf("frontier food should not count as ours: MyFood=%d", vr.MyFood)
	}
	if vr.OppFood != 0 {
		t.Errorf("frontier food should not count as opponent's: OppFood=%d", vr.OppFood)
	}
}

func TestVoronoiResult_Partitioned(t *testing.T) {
	// Body wall completely separates the two snakes on a 5x3 board.
	// Snake b's interior body (indices 1..4) blocks column 2 fully.
	g := &GameSim{
		Width:  5,
		Height: 3,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 1}}, Health: 100, Length: 1},
			{ID: "b", Body: []Coord{{4, 1}, {2, 0}, {2, 1}, {2, 2}, {3, 2}, {4, 2}}, Health: 100, Length: 6},
		},
	}
	vr := VoronoiTerritory(g, 0)
	if !vr.IsPartitioned {
		t.Error("expected IsPartitioned=true when body wall separates snakes")
	}
}

func TestVoronoiResult_NotPartitioned(t *testing.T) {
	g := &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{10, 10}, {9, 10}}, Health: 100, Length: 2},
		},
	}
	vr := VoronoiTerritory(g, 0)
	if vr.IsPartitioned {
		t.Error("expected IsPartitioned=false on open board")
	}
}

func TestVoronoiResult_SingleSnakeNotPartitioned(t *testing.T) {
	g := &GameSim{
		Width:  5,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{2, 2}}, Health: 100, Length: 1},
		},
	}
	vr := VoronoiTerritory(g, 0)
	if vr.IsPartitioned {
		t.Error("expected IsPartitioned=false with single snake")
	}
}

func TestVoronoiResult_FoodDistances(t *testing.T) {
	// Snake a at (0,0), snake b at (10,10). Food at (2,0) dist=2 for a, (8,10) dist=2 for b.
	g := &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{10, 10}, {9, 10}}, Health: 100, Length: 2},
		},
		Food: []Coord{{2, 0}, {8, 10}},
	}
	vr := VoronoiTerritory(g, 0)
	if vr.MyClosestFoodDist != 2 {
		t.Errorf("expected MyClosestFoodDist=2, got %d", vr.MyClosestFoodDist)
	}
	if vr.OppClosestFoodDist != 2 {
		t.Errorf("expected OppClosestFoodDist=2, got %d", vr.OppClosestFoodDist)
	}
	// MyFoodValue = 1/2 = 0.5
	if math.Abs(vr.MyFoodValue-0.5) > 0.01 {
		t.Errorf("expected MyFoodValue≈0.5, got %f", vr.MyFoodValue)
	}
}

func TestVoronoiResult_FoodValueMultiple(t *testing.T) {
	// Snake a at (0,0), snake b at (10,10). Two foods close to a: dist=1 and dist=3.
	g := &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{10, 10}, {9, 10}}, Health: 100, Length: 2},
		},
		Food: []Coord{{0, 1}, {3, 0}}, // dist 1 and dist 3 from head (0,0)
	}
	vr := VoronoiTerritory(g, 0)
	if vr.MyFood != 2 {
		t.Errorf("expected MyFood=2, got %d", vr.MyFood)
	}
	// MyFoodValue = 1/1 + 1/3 ≈ 1.333
	expected := 1.0 + 1.0/3.0
	if math.Abs(vr.MyFoodValue-expected) > 0.01 {
		t.Errorf("expected MyFoodValue≈%.3f, got %f", expected, vr.MyFoodValue)
	}
	if vr.MyClosestFoodDist != 1 {
		t.Errorf("expected MyClosestFoodDist=1, got %d", vr.MyClosestFoodDist)
	}
}

func TestVoronoiResult_NoFoodZeroValues(t *testing.T) {
	g := &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{10, 10}, {9, 10}}, Health: 100, Length: 2},
		},
	}
	vr := VoronoiTerritory(g, 0)
	if vr.MyClosestFoodDist != 0 {
		t.Errorf("expected MyClosestFoodDist=0, got %d", vr.MyClosestFoodDist)
	}
	if vr.OppClosestFoodDist != 0 {
		t.Errorf("expected OppClosestFoodDist=0, got %d", vr.OppClosestFoodDist)
	}
	if vr.MyFoodValue != 0 {
		t.Errorf("expected MyFoodValue=0, got %f", vr.MyFoodValue)
	}
}

func TestVoronoiResult_TerritoryDepth(t *testing.T) {
	g := &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{10, 10}, {9, 10}}, Health: 100, Length: 2},
		},
	}
	vr := VoronoiTerritory(g, 0)
	if vr.MyTerritoryDepth <= 0 {
		t.Errorf("expected MyTerritoryDepth > 0, got %d", vr.MyTerritoryDepth)
	}
}

func TestVoronoiResult_CentroidSymmetric(t *testing.T) {
	// Symmetric board: snakes at opposite corners.
	g := &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{10, 10}, {9, 10}}, Health: 100, Length: 2},
		},
	}
	vr := VoronoiTerritory(g, 0)
	// Centroids should be roughly mirror images across the center (5,5).
	sumX := vr.MyCenterX + vr.OppCenterX
	sumY := vr.MyCenterY + vr.OppCenterY
	if math.Abs(sumX-10.0) > 2.0 {
		t.Errorf("centroid X not symmetric: my=%.1f opp=%.1f sum=%.1f", vr.MyCenterX, vr.OppCenterX, sumX)
	}
	if math.Abs(sumY-10.0) > 2.0 {
		t.Errorf("centroid Y not symmetric: my=%.1f opp=%.1f sum=%.1f", vr.MyCenterY, vr.OppCenterY, sumY)
	}
}

func TestVoronoiResult_TailReachable(t *testing.T) {
	// Short snake — tail is nearby and in own territory.
	g := &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{1, 1}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{9, 9}, {9, 10}}, Health: 100, Length: 2},
		},
	}
	vr := VoronoiTerritory(g, 0)
	if !vr.MyTailReachable {
		t.Error("expected MyTailReachable=true when tail is in own territory")
	}
}

func TestVoronoiResult_TailNotReachable(t *testing.T) {
	// Snake a's tail is deep in opponent territory.
	g := &GameSim{
		Width:  5,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}, {4, 4}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{3, 3}, {3, 2}}, Health: 100, Length: 2},
		},
	}
	vr := VoronoiTerritory(g, 0)
	if vr.MyTailReachable {
		t.Error("expected MyTailReachable=false when tail is in opponent territory")
	}
}

func TestVoronoiResult_SingleSnakeDepthAndCentroid(t *testing.T) {
	g := &GameSim{
		Width:  5,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{2, 2}, {2, 1}, {2, 0}}, Health: 100, Length: 3},
		},
	}
	vr := VoronoiTerritory(g, 0)
	// Max depth from center: corners are at Manhattan dist 4 (e.g., (0,0) from (2,2)).
	if vr.MyTerritoryDepth < 4 {
		t.Errorf("expected MyTerritoryDepth >= 4, got %d", vr.MyTerritoryDepth)
	}
	// Centroid should be near board center (accounting for body blocked cells).
	if math.Abs(vr.MyCenterX-2.0) > 0.5 {
		t.Errorf("expected MyCenterX≈2.0, got %.2f", vr.MyCenterX)
	}
	if math.Abs(vr.MyCenterY-2.0) > 0.5 {
		t.Errorf("expected MyCenterY≈2.0, got %.2f", vr.MyCenterY)
	}
}
