package logic

import "testing"

func testGameSim() *GameSim {
	return &GameSim{
		Width:  11,
		Height: 11,
		Snakes: []SimSnake{
			{ID: "a", Body: []Coord{{1, 1}, {1, 2}, {1, 3}}, Health: 100, Length: 3},
			{ID: "b", Body: []Coord{{5, 5}, {5, 6}, {5, 7}}, Health: 100, Length: 3},
		},
		Food: []Coord{{3, 3}, {7, 7}},
	}
}

func TestHashConsistency(t *testing.T) {
	g := testGameSim()
	h1 := g.Hash()
	h2 := g.Hash()
	if h1 != h2 {
		t.Fatalf("same state gave different hashes: %x vs %x", h1, h2)
	}
}

func TestHashCloneConsistency(t *testing.T) {
	g := testGameSim()
	c := g.Clone()
	if g.Hash() != c.Hash() {
		t.Fatal("clone has different hash")
	}
}

func TestHashSnakePositionSensitivity(t *testing.T) {
	g1 := testGameSim()
	g2 := testGameSim()
	g2.Snakes[0].Body[0] = Coord{2, 2} // different head
	if g1.Hash() == g2.Hash() {
		t.Fatal("different snake positions should give different hashes")
	}
}

func TestHashFoodSensitivity(t *testing.T) {
	g1 := testGameSim()
	g2 := testGameSim()
	g2.Food[0] = Coord{4, 4}
	if g1.Hash() == g2.Hash() {
		t.Fatal("different food should give different hashes")
	}
}

func TestHashDeadSnakeExcluded(t *testing.T) {
	g1 := testGameSim()
	g2 := testGameSim()
	g2.Snakes[1].EliminatedCause = "wall"
	// Dead snake should be excluded, so hashes differ (one fewer snake hashed).
	if g1.Hash() == g2.Hash() {
		t.Fatal("dead snake exclusion should change hash")
	}
}

func TestHashFoodOrderInvariance(t *testing.T) {
	g1 := testGameSim()
	g2 := testGameSim()
	// Reverse food order.
	g2.Food[0], g2.Food[1] = g2.Food[1], g2.Food[0]
	if g1.Hash() != g2.Hash() {
		t.Fatal("food order should not affect hash (XOR is commutative)")
	}
}

func TestHashNonZero(t *testing.T) {
	g := testGameSim()
	if g.Hash() == 0 {
		t.Fatal("hash should be non-zero for non-empty board")
	}
}
