package logic

import "math/rand"

const (
	maxBoardCells = 121 // 11×11
	maxBodyLen    = 121
)

var (
	zobristBody [MaxSnakes][maxBodyLen][maxBoardCells]uint64
	zobristFood [maxBoardCells]uint64
)

func init() {
	rng := rand.New(rand.NewSource(0x4861_7275_6B30))
	for s := 0; s < MaxSnakes; s++ {
		for seg := 0; seg < maxBodyLen; seg++ {
			for c := 0; c < maxBoardCells; c++ {
				zobristBody[s][seg][c] = rng.Uint64()
			}
		}
	}
	for c := 0; c < maxBoardCells; c++ {
		zobristFood[c] = rng.Uint64()
	}
}

// Hash computes a Zobrist hash of the game state.
// It hashes alive snake bodies (by snake index and segment index) and food positions.
// Health, hazards, and turn are excluded by design.
func (g *GameSim) Hash() uint64 {
	cells := g.Width * g.Height
	if cells > maxBoardCells {
		return 0
	}

	var h uint64
	for i := range g.Snakes {
		s := &g.Snakes[i]
		if !s.IsAlive() {
			continue
		}
		if i >= MaxSnakes {
			break
		}
		for seg := 0; seg < len(s.Body) && seg < maxBodyLen; seg++ {
			cell := s.Body[seg].Y*g.Width + s.Body[seg].X
			if cell >= 0 && cell < cells {
				h ^= zobristBody[i][seg][cell]
			}
		}
	}

	for _, f := range g.Food {
		cell := f.Y*g.Width + f.X
		if cell >= 0 && cell < cells {
			h ^= zobristFood[cell]
		}
	}

	return h
}
