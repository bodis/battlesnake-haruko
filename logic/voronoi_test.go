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
	vr := VoronoiTerritory(g, 0, false)
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
	vr := VoronoiTerritory(g, 0, false)
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
	vr := VoronoiTerritory(g, 0, false)
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
	vr := VoronoiTerritory(g, 0, false)
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
	vr := VoronoiTerritory(g, 0, false)
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
	vr := VoronoiTerritory(g, 0, false)
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
	vr := VoronoiTerritory(g, 0, false)
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
	vr := VoronoiTerritory(g, 0, false)
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
	vr := VoronoiTerritory(g, 0, false)
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
	vr := VoronoiTerritory(g, 0, false)
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
	vr := VoronoiTerritory(g, 0, false)
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
	vr := VoronoiTerritory(g, 0, false)
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
	vr := VoronoiTerritory(g, 0, false)
	if vr.MyFood != 2 {
		t.Errorf("expected MyFood=2, got %d", vr.MyFood)
	}
	// MyFoodValue = 1/1 + 1/3 ≈ 1.333
	expected := 1.0 + 1.0/3.0
	if math.Abs(vr.MyFoodValue-expected) > 0.01 {
		t.Errorf("expected MyFoodValue≈%.3f, got %f", expected, vr.MyFoodValue)
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
	vr := VoronoiTerritory(g, 0, false)
	if vr.MyFoodValue != 0 {
		t.Errorf("expected MyFoodValue=0, got %f", vr.MyFoodValue)
	}
}

func TestVoronoiResult_BottleneckCorridor(t *testing.T) {
	// 7x5 board. Snake A's territory is a corridor through a 1-cell gap.
	// Snake B's body creates a wall with a gap that A must pass through.
	//
	// Layout (7x5, y=0 bottom):
	//   y=4: . . . . . . .
	//   y=3: . . B B B B .
	//   y=2: A . . . . B .
	//   y=1: . . B B B B .
	//   y=0: . . . . . . .
	//
	// Snake A at (0,2), snake B head at (5,3), body blocks a corridor.
	// A's territory extends through the gap at (1,2) which is an AP.
	g := &GameSim{
		Width:  7,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 2}, {0, 1}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{5, 3}, {2, 3}, {3, 3}, {4, 3}, {5, 1}, {4, 1}, {3, 1}, {2, 1}, {5, 2}}, Health: 100, Length: 9},
		},
	}
	vr := VoronoiTerritory(g, 0, false)
	if vr.MyThreatenedTerritory == 0 {
		t.Error("expected MyThreatenedTerritory > 0 for corridor territory")
	}
}

func TestVoronoiResult_BottleneckCompact(t *testing.T) {
	// Snakes at opposite corners of 11x11 — compact territories, no bottlenecks.
	g := &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{1, 1}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{9, 9}, {9, 10}}, Health: 100, Length: 2},
		},
	}
	vr := VoronoiTerritory(g, 0, false)
	if vr.MyThreatenedTerritory != 0 {
		t.Errorf("expected MyThreatenedTerritory=0 for compact territory, got %d", vr.MyThreatenedTerritory)
	}
}

func TestVoronoiResult_BottleneckOpponent(t *testing.T) {
	// Mirror: opponent has corridor territory, we should detect OppThreatenedTerritory.
	g := &GameSim{
		Width:  7,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{5, 3}, {2, 3}, {3, 3}, {4, 3}, {5, 1}, {4, 1}, {3, 1}, {2, 1}, {5, 2}}, Health: 100, Length: 9},
			{ID: "b", Body: []Coord{{0, 2}, {0, 1}}, Health: 100, Length: 2},
		},
	}
	// myIdx=0 is the snake with the body wall, opponent (idx=1) has corridor
	vr := VoronoiTerritory(g, 0, false)
	if vr.OppThreatenedTerritory == 0 {
		t.Error("expected OppThreatenedTerritory > 0 for opponent corridor")
	}
}

func TestVoronoiResult_BottleneckInternalAPIgnored(t *testing.T) {
	// Single snake owns entire small board — any internal APs should be
	// ignored because they have no non-owned neighbors.
	g := &GameSim{
		Width:  5,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{2, 2}, {2, 1}}, Health: 100, Length: 2},
		},
	}
	vr := VoronoiTerritory(g, 0, false)
	if vr.MyThreatenedTerritory != 0 {
		t.Errorf("expected MyThreatenedTerritory=0 for single snake (no live APs), got %d",
			vr.MyThreatenedTerritory)
	}
}

func TestVoronoiResult_BottleneckSmallTerritory(t *testing.T) {
	// Territory < 8 cells should skip bottleneck detection entirely.
	g := &GameSim{
		Width:  5,
		Height: 3,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 1}}, Health: 100, Length: 1},
			{ID: "b", Body: []Coord{{4, 1}, {2, 0}, {2, 1}, {2, 2}, {3, 2}, {4, 2}}, Health: 100, Length: 6},
		},
	}
	vr := VoronoiTerritory(g, 0, false)
	// Snake A's territory is ~5 cells (left side of partition) — too small for bottleneck.
	if vr.MyThreatenedTerritory != 0 {
		t.Errorf("expected MyThreatenedTerritory=0 for small territory (<8), got %d",
			vr.MyThreatenedTerritory)
	}
}

func TestVoronoiResult_19x19Board(t *testing.T) {
	// Verify the engine works on a 19x19 board without panicking.
	g := &GameSim{
		Width:  19,
		Height: 19,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{1, 1}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{17, 17}, {17, 18}}, Health: 100, Length: 2},
		},
		Food: []Coord{{9, 9}},
	}
	vr := VoronoiTerritory(g, 0, false)
	if vr.MyTerritory == 0 || vr.OppTerritory == 0 {
		t.Errorf("expected non-zero territory on 19x19, got my=%d opp=%d", vr.MyTerritory, vr.OppTerritory)
	}
	total := vr.MyTerritory + vr.OppTerritory
	// 19*19=361, minus 2 body segments, minus some tied cells
	if total < 300 {
		t.Errorf("expected substantial territory on 19x19, got total=%d", total)
	}
}

func TestVoronoiResult_7x7Board(t *testing.T) {
	g := &GameSim{
		Width:  7,
		Height: 7,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{1, 1}, {1, 0}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{5, 5}, {5, 6}}, Health: 100, Length: 2},
		},
	}
	vr := VoronoiTerritory(g, 0, false)
	if vr.MyTerritory == 0 || vr.OppTerritory == 0 {
		t.Errorf("expected non-zero territory on 7x7, got my=%d opp=%d", vr.MyTerritory, vr.OppTerritory)
	}
}

// --- BFSTailDist tests ---

func TestBFSTailDist_SimpleOpenPath(t *testing.T) {
	// 5x5 board, snake at (0,0)→(1,0)→(2,0). Territory connects head to tail.
	g := &GameSim{
		Width:  5,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}, {1, 0}, {2, 0}}, Health: 100, Length: 3},
		},
	}
	var ownerBuf [MaxBoardCells]int8
	VoronoiTerritoryWithOwner(g, 0, true, ownerBuf[:])
	dist, reachable := BFSTailDist(g, 0, ownerBuf[:], g.Width, g.Height)
	if !reachable {
		t.Fatal("expected tail to be reachable")
	}
	if dist != 2 {
		t.Errorf("expected dist=2, got %d", dist)
	}
}

func TestBFSTailDist_BlockedPath(t *testing.T) {
	// 5x3 board. Snake A owns left side, tail is on right side (unreachable).
	// Snake B's body wall blocks the path.
	g := &GameSim{
		Width:  5,
		Height: 3,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 1}, {4, 1}}, Health: 100, Length: 2},
			{ID: "b", Body: []Coord{{4, 2}, {2, 0}, {2, 1}, {2, 2}, {3, 2}}, Health: 100, Length: 5},
		},
	}
	var ownerBuf [MaxBoardCells]int8
	VoronoiTerritoryWithOwner(g, 0, true, ownerBuf[:])
	_, reachable := BFSTailDist(g, 0, ownerBuf[:], g.Width, g.Height)
	if reachable {
		t.Error("expected tail to be unreachable when partitioned by opponent body")
	}
}

func TestBFSTailDist_PathThroughBody(t *testing.T) {
	// Snake's body bridges a gap in its territory.
	// 3x3 board, snake wraps: head (0,0), body (1,0), (2,0), (2,1), tail (2,2).
	// Only one other snake far away; territory alone may not connect, but body does.
	g := &GameSim{
		Width:  3,
		Height: 3,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{0, 0}, {1, 0}, {2, 0}, {2, 1}, {2, 2}}, Health: 100, Length: 5},
		},
	}
	var ownerBuf [MaxBoardCells]int8
	VoronoiTerritoryWithOwner(g, 0, true, ownerBuf[:])
	dist, reachable := BFSTailDist(g, 0, ownerBuf[:], g.Width, g.Height)
	if !reachable {
		t.Fatal("expected tail reachable through body cells")
	}
	// Shortest path: (0,0)→(1,0)→(2,0)→(2,1)→(2,2) = 4 steps
	if dist != 4 {
		t.Errorf("expected dist=4, got %d", dist)
	}
}

func TestBFSTailDist_LengthOne(t *testing.T) {
	g := &GameSim{
		Width:  5,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{2, 2}}, Health: 100, Length: 1},
		},
	}
	var ownerBuf [MaxBoardCells]int8
	VoronoiTerritoryWithOwner(g, 0, true, ownerBuf[:])
	dist, reachable := BFSTailDist(g, 0, ownerBuf[:], g.Width, g.Height)
	if !reachable {
		t.Fatal("expected reachable for length-1 snake")
	}
	if dist != 0 {
		t.Errorf("expected dist=0 for length-1 snake, got %d", dist)
	}
}

func TestBFSTailDist_DeadSnake(t *testing.T) {
	g := &GameSim{
		Width:  5,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{2, 2}, {2, 1}}, Health: 0, Length: 2, EliminatedCause: "starvation"},
		},
	}
	var ownerBuf [MaxBoardCells]int8
	dist, reachable := BFSTailDist(g, 0, ownerBuf[:], g.Width, g.Height)
	if reachable {
		t.Error("expected unreachable for dead snake")
	}
	if dist != -1 {
		t.Errorf("expected dist=-1 for dead snake, got %d", dist)
	}
}

func TestVoronoiResult_SingleSnakeTerritory(t *testing.T) {
	g := &GameSim{
		Width:  5,
		Height: 5,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{2, 2}, {2, 1}, {2, 0}}, Health: 100, Length: 3},
		},
	}
	vr := VoronoiTerritory(g, 0, false)
	// Single snake should own all non-body territory.
	if vr.MyTerritory < 20 {
		t.Errorf("expected MyTerritory >= 20, got %d", vr.MyTerritory)
	}
}
