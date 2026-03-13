package main

import (
	"math/rand"

	"github.com/bodist/haruko/logic"
)

// maybeSpawnFood implements standard Battlesnake food spawning:
// 15% chance each turn, and always maintain minimum of 1 food on the board.
func maybeSpawnFood(g *logic.GameSim, rng *rand.Rand) {
	shouldSpawn := len(g.Food) < 1 || rng.Float64() < 0.15
	if !shouldSpawn {
		return
	}

	// Build set of occupied cells.
	occupied := make(map[logic.Coord]bool)
	for i := range g.Snakes {
		if !g.Snakes[i].IsAlive() {
			continue
		}
		for _, c := range g.Snakes[i].Body {
			occupied[c] = true
		}
	}
	for _, f := range g.Food {
		occupied[f] = true
	}

	// Collect empty cells.
	var empty []logic.Coord
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			c := logic.Coord{X: x, Y: y}
			if !occupied[c] {
				empty = append(empty, c)
			}
		}
	}

	if len(empty) == 0 {
		return
	}

	// Pick a random empty cell.
	cell := empty[rng.Intn(len(empty))]
	g.Food = append(g.Food, cell)
}
