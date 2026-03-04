package logic

import (
	"math"
	"testing"
)

func TestTTStoreRetrieveExact(t *testing.T) {
	tt := NewTranspositionTable()
	tt.NewGeneration()

	hash := uint64(0xDEADBEEF)
	tt.Store(hash, 3, 1.5, Up, math.Inf(-1), math.Inf(1))

	score, move, hasTTMove, hit := tt.Probe(hash, 3, math.Inf(-1), math.Inf(1))
	if !hit {
		t.Fatal("expected hit")
	}
	if score != 1.5 {
		t.Fatalf("expected score 1.5, got %f", score)
	}
	if move != Up {
		t.Fatalf("expected move Up, got %d", move)
	}
	if !hasTTMove {
		t.Fatal("expected hasTTMove")
	}
}

func TestTTShallowNoHit(t *testing.T) {
	tt := NewTranspositionTable()
	tt.NewGeneration()

	hash := uint64(0xCAFEBABE)
	tt.Store(hash, 2, 0.5, Right, math.Inf(-1), math.Inf(1))

	// Probe at depth 4 — stored depth 2 is too shallow for cutoff.
	_, move, hasTTMove, hit := tt.Probe(hash, 4, math.Inf(-1), math.Inf(1))
	if hit {
		t.Fatal("should not hit with shallower stored depth")
	}
	// But we should still get the move for ordering.
	if !hasTTMove {
		t.Fatal("expected hasTTMove even without hit")
	}
	if move != Right {
		t.Fatalf("expected move Right, got %d", move)
	}
}

func TestTTGenerationInvalidation(t *testing.T) {
	tt := NewTranspositionTable()
	tt.NewGeneration()

	hash := uint64(0x12345678)
	tt.Store(hash, 3, 2.0, Left, math.Inf(-1), math.Inf(1))

	tt.NewGeneration() // invalidate

	_, _, hasTTMove, hit := tt.Probe(hash, 3, math.Inf(-1), math.Inf(1))
	if hit {
		t.Fatal("should not hit after new generation")
	}
	if hasTTMove {
		t.Fatal("should not have TT move after new generation")
	}
}

func TestTTHashCollisionDetection(t *testing.T) {
	tt := NewTranspositionTable()
	tt.NewGeneration()

	hash1 := uint64(0xAAAA)
	hash2 := hash1 + ttSize // same index, different hash
	tt.Store(hash1, 3, 1.0, Up, math.Inf(-1), math.Inf(1))

	_, _, _, hit := tt.Probe(hash2, 3, math.Inf(-1), math.Inf(1))
	if hit {
		t.Fatal("different hash in same slot should not hit")
	}
}

func TestTTLowerBound(t *testing.T) {
	tt := NewTranspositionTable()
	tt.NewGeneration()

	hash := uint64(0xBBBB)
	// Store with score >= beta → lower bound.
	tt.Store(hash, 3, 5.0, Down, 1.0, 3.0)

	// Probe where stored score (5.0) >= beta (4.0) → should hit.
	score, _, _, hit := tt.Probe(hash, 3, 2.0, 4.0)
	if !hit {
		t.Fatal("lower bound with score >= beta should hit")
	}
	if score != 5.0 {
		t.Fatalf("expected score 5.0, got %f", score)
	}

	// Probe where stored score (5.0) < beta (6.0) → no hit.
	_, _, _, hit = tt.Probe(hash, 3, 2.0, 6.0)
	if hit {
		t.Fatal("lower bound with score < beta should not hit")
	}
}

func TestTTUpperBound(t *testing.T) {
	tt := NewTranspositionTable()
	tt.NewGeneration()

	hash := uint64(0xCCCC)
	// Store with score <= alpha0 → upper bound.
	tt.Store(hash, 3, 1.0, Up, 2.0, 5.0)

	// Probe where stored score (1.0) <= alpha (1.5) → should hit.
	score, _, _, hit := tt.Probe(hash, 3, 1.5, 5.0)
	if !hit {
		t.Fatal("upper bound with score <= alpha should hit")
	}
	if score != 1.0 {
		t.Fatalf("expected score 1.0, got %f", score)
	}

	// Probe where stored score (1.0) > alpha (0.5) → no hit.
	_, _, _, hit = tt.Probe(hash, 3, 0.5, 5.0)
	if hit {
		t.Fatal("upper bound with score > alpha should not hit")
	}
}
