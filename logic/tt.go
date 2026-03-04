package logic

const (
	ttExact = iota
	ttLower
	ttUpper

	ttSize = 1 << 20 // ~1M entries
	ttMask = ttSize - 1
)

type ttEntry struct {
	hash       uint64
	score      float64
	bestMove   Direction
	depth      int8
	flag       int8
	generation uint8
}

// TranspositionTable caches search results keyed by Zobrist hash.
type TranspositionTable struct {
	entries    []ttEntry
	generation uint8
}

// NewTranspositionTable allocates a TT with ttSize slots.
func NewTranspositionTable() *TranspositionTable {
	return &TranspositionTable{
		entries: make([]ttEntry, ttSize),
	}
}

var sharedTT *TranspositionTable

// getSharedTT returns a reusable TT singleton to avoid repeated 32MB allocations.
func getSharedTT() *TranspositionTable {
	if sharedTT == nil {
		sharedTT = NewTranspositionTable()
	}
	return sharedTT
}

// NewGeneration bumps the generation counter, logically invalidating all entries.
func (tt *TranspositionTable) NewGeneration() {
	tt.generation++
}

// Probe looks up a position in the TT.
// Returns (score, bestMove, hasTTMove, hit).
// hit=true means the stored result can be used as a cutoff (depth and bounds match).
// hasTTMove=true means the entry matched hash+generation so bestMove is valid for move ordering.
func (tt *TranspositionTable) Probe(hash uint64, depth int, alpha, beta float64) (float64, Direction, bool, bool) {
	idx := hash & ttMask
	e := &tt.entries[idx]

	if e.hash != hash || e.generation != tt.generation {
		return 0, Down, false, false
	}

	hasTTMove := true
	bestMove := e.bestMove

	if int(e.depth) < depth {
		return 0, bestMove, hasTTMove, false
	}

	switch e.flag {
	case ttExact:
		return e.score, bestMove, hasTTMove, true
	case ttLower:
		if e.score >= beta {
			return e.score, bestMove, hasTTMove, true
		}
	case ttUpper:
		if e.score <= alpha {
			return e.score, bestMove, hasTTMove, true
		}
	}

	return 0, bestMove, hasTTMove, false
}

// Store writes a search result into the TT.
// alpha0 is the original alpha before any raising (used to determine flag).
func (tt *TranspositionTable) Store(hash uint64, depth int, score float64, bestMove Direction, alpha0, beta float64) {
	idx := hash & ttMask
	e := &tt.entries[idx]

	// Replacement: always replace stale generation; otherwise replace if deeper or equal depth.
	if e.generation == tt.generation && e.hash != 0 && int(e.depth) > depth {
		return
	}

	var flag int8
	if score <= alpha0 {
		flag = ttUpper
	} else if score >= beta {
		flag = ttLower
	} else {
		flag = ttExact
	}

	e.hash = hash
	e.score = score
	e.bestMove = bestMove
	e.depth = int8(depth)
	e.flag = flag
	e.generation = tt.generation
}
