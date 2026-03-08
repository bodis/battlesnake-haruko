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
	var vr VoronoiResult
	for i := 0; i < b.N; i++ {
		vr = VoronoiTerritory(g, 0, false)
	}
	_ = vr
}

func BenchmarkVoronoiNoBottleneck(b *testing.B) {
	g := standardBenchGame()
	b.ResetTimer()
	var vr VoronoiResult
	for i := 0; i < b.N; i++ {
		vr = VoronoiTerritory(g, 0, true)
	}
	_ = vr
}

func BenchmarkEvaluateLateGame(b *testing.B) {
	// Late-game scenario: long snakes filling >30% of board to trigger Tarjan.
	body0 := make([]Coord, 20)
	body1 := make([]Coord, 20)
	for i := range body0 {
		body0[i] = Coord{i % 11, i / 11}
		body1[i] = Coord{10 - i%11, 10 - i/11}
	}
	g := NewGameSim(11, 11, []SimSnake{
		{ID: "me", Body: body0, Health: 100, Length: 20},
		{ID: "opp", Body: body1, Health: 100, Length: 20},
	}, []Coord{{5, 5}}, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Evaluate(g, 0)
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

func BenchmarkBestMoveIterativeDepth(b *testing.B) {
	g := standardBenchGame()
	b.ResetTimer()
	var totalDepth int
	for i := 0; i < b.N; i++ {
		g.BestMoveIterative("me", 50*time.Millisecond)
		totalDepth += g.LastCompletedDepth
	}
	b.ReportMetric(float64(totalDepth)/float64(b.N), "depth/op")
}

func BenchmarkBestMoveIterativeNodes(b *testing.B) {
	g := standardBenchGame()
	b.ResetTimer()
	var totalNodes int64
	for i := 0; i < b.N; i++ {
		g.BestMoveIterative("me", 50*time.Millisecond)
		totalNodes += g.LastNodeCount
	}
	b.ReportMetric(float64(totalNodes)/float64(b.N), "nodes/op")
}
