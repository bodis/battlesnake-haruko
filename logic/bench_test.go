package logic

import (
	"testing"
	"time"
)

// standardBenchGame creates a typical 11x11 2-snake game for benchmarks.
func standardBenchGame() *GameSim {
	return NewGameSim(11, 11, []SimSnake{
		{ID: "me", Body: []Coord{{5, 5}, {5, 4}, {5, 3}}, Health: 100, Length: 3},
		{ID: "opp", Body: []Coord{{3, 5}, {3, 6}, {3, 7}}, Health: 100, Length: 3},
	}, []Coord{{7, 7}, {2, 2}}, nil)
}

func BenchmarkClone(b *testing.B) {
	g := standardBenchGame()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Clone()
	}
}

func BenchmarkCloneFromPool(b *testing.B) {
	g := standardBenchGame()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := g.CloneFromPool()
		c.Release()
	}
}

func BenchmarkStep(b *testing.B) {
	g := standardBenchGame()
	ms := newMoveSet2(0, Up, 1, Left)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := g.CloneFromPool()
		c.Step(ms)
		c.Release()
	}
}

func BenchmarkEvaluate(b *testing.B) {
	g := standardBenchGame()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Evaluate(g, 0)
	}
}

func BenchmarkVoronoi(b *testing.B) {
	g := standardBenchGame()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VoronoiTerritory(g, 0)
	}
}

func BenchmarkBRSNode(b *testing.B) {
	g := standardBenchGame()
	ms := newMoveSet2(0, Up, 1, Left)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := g.CloneFromPool()
		c.Step(ms)
		Evaluate(c, 0)
		c.Release()
	}
}

func BenchmarkBestMoveIterative(b *testing.B) {
	g := standardBenchGame()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.BestMoveIterative("me", 50*time.Millisecond)
	}
}
