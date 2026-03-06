package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
)

type record struct {
	Type string `json:"type"`

	// Header
	GameID    string `json:"game_id"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	MySnakeID string `json:"my_snake_id"`

	// Turn
	Turn      *int    `json:"turn"`
	Move      string  `json:"move"`
	Eval      float64 `json:"eval"`
	Depth     int     `json:"depth"`
	MyHealth  int     `json:"my_health"`
	MyLen     int     `json:"my_len"`
	OppLen    int     `json:"opp_len"`
	MyTerritory  int  `json:"my_territory"`
	OppTerritory int  `json:"opp_territory"`
	IsPartitioned bool `json:"is_partitioned"`
	FoodCount int    `json:"food_count"`

	Territory       float64 `json:"territory"`
	LenAdvantage    float64 `json:"len_advantage"`
	H2H             float64 `json:"h2h"`
	OppConfinement  float64 `json:"opp_confinement"`
	SelfConfinement float64 `json:"self_confinement"`
	FoodUrgency     float64 `json:"food_urgency"`
	FoodCluster     float64 `json:"food_cluster"`
	FoodReach       float64 `json:"food_reach"`
	FoodDenial      float64 `json:"food_denial"`
	StarvationRisk  float64 `json:"starvation_risk"`
	GrowthUrgency   float64 `json:"growth_urgency"`
	TailChase       float64 `json:"tail_chase"`
	Bottleneck      float64 `json:"bottleneck"`
	MyThreatened    int     `json:"my_threatened"`
	OppThreatened   int     `json:"opp_threatened"`
	EarlyBlend      float64 `json:"early_blend"`
	LateBlend       float64 `json:"late_blend"`

	// Footer
	Result     string `json:"result"`
	DeathCause string `json:"death_cause"`
	TotalTurns int    `json:"total_turns"`
}

type game struct {
	header  record
	turns   []record
	footer  record
}

func loadGames(files []string) []game {
	var games []game
	for _, fname := range files {
		f, err := os.Open(fname)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: cannot open %s: %v\n", fname, err)
			continue
		}
		var g game
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			var r record
			if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
				continue
			}
			switch r.Type {
			case "header":
				g.header = r
			case "turn":
				g.turns = append(g.turns, r)
			case "footer":
				g.footer = r
			}
		}
		f.Close()
		if g.header.Type == "header" {
			games = append(games, g)
		}
	}
	return games
}

func modeSummary(games []game) {
	wins, losses, draws := 0, 0, 0
	deathCounts := map[string]int{}
	var winTurns, lossTurns []int

	for _, g := range games {
		switch g.footer.Result {
		case "win":
			wins++
			winTurns = append(winTurns, g.footer.TotalTurns)
		case "loss":
			losses++
			lossTurns = append(lossTurns, g.footer.TotalTurns)
			if g.footer.DeathCause != "" {
				deathCounts[g.footer.DeathCause]++
			}
		case "draw":
			draws++
		}

		// Find biggest eval swing.
		var maxSwing float64
		swingTurn := 0
		for i := 1; i < len(g.turns); i++ {
			swing := g.turns[i].Eval - g.turns[i-1].Eval
			if math.Abs(swing) > math.Abs(maxSwing) {
				maxSwing = swing
				swingTurn = *g.turns[i].Turn
			}
		}

		peakEval := 0.0
		peakTurn := 0
		for _, t := range g.turns {
			if t.Eval > peakEval {
				peakEval = t.Eval
				peakTurn = *t.Turn
			}
		}

		prefix := g.header.GameID
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
		switch g.footer.Result {
		case "loss":
			fmt.Printf("Game %s: LOSS (%s) turn %d, swing %.1f at turn %d\n",
				prefix, g.footer.DeathCause, g.footer.TotalTurns, maxSwing, swingTurn)
		case "win":
			fmt.Printf("Game %s: WIN  turn %d, peak eval %.1f at turn %d\n",
				prefix, g.footer.TotalTurns, peakEval, peakTurn)
		default:
			fmt.Printf("Game %s: DRAW turn %d\n", prefix, g.footer.TotalTurns)
		}
	}

	fmt.Println("---")
	fmt.Printf("%d games: %d wins, %d losses, %d draws\n", len(games), wins, losses, draws)

	if len(deathCounts) > 0 {
		fmt.Printf("Deaths:")
		for cause, count := range deathCounts {
			fmt.Printf(" %s=%d", cause, count)
		}
		fmt.Println()
	}

	avg := func(vals []int) float64 {
		if len(vals) == 0 {
			return 0
		}
		sum := 0
		for _, v := range vals {
			sum += v
		}
		return float64(sum) / float64(len(vals))
	}
	if len(winTurns) > 0 || len(lossTurns) > 0 {
		fmt.Printf("Avg game length: wins=%.0f losses=%.0f\n", avg(winTurns), avg(lossTurns))
	}
}

func modeTurningPoints(games []game, threshold float64, top int) {
	type turningPoint struct {
		gameID     string
		turn       int
		prevEval   float64
		currEval   float64
		swing      float64
		biggestSig string
		biggestDrop float64
		secondSig  string
		secondDrop float64
		deathTurn  int
		deathCause string
	}

	var points []turningPoint
	for _, g := range games {
		for i := 1; i < len(g.turns); i++ {
			swing := g.turns[i].Eval - g.turns[i-1].Eval
			if math.Abs(swing) < threshold {
				continue
			}
			// Find the two biggest signal changes.
			type sigDelta struct {
				name  string
				delta float64
			}
			deltas := []sigDelta{
				{"Territory", g.turns[i].Territory - g.turns[i-1].Territory},
				{"LenAdvantage", g.turns[i].LenAdvantage - g.turns[i-1].LenAdvantage},
				{"H2H", g.turns[i].H2H - g.turns[i-1].H2H},
				{"OppConfinement", g.turns[i].OppConfinement - g.turns[i-1].OppConfinement},
				{"SelfConfinement", g.turns[i].SelfConfinement - g.turns[i-1].SelfConfinement},
				{"FoodUrgency", g.turns[i].FoodUrgency - g.turns[i-1].FoodUrgency},
				{"FoodCluster", g.turns[i].FoodCluster - g.turns[i-1].FoodCluster},
				{"FoodReach", g.turns[i].FoodReach - g.turns[i-1].FoodReach},
				{"FoodDenial", g.turns[i].FoodDenial - g.turns[i-1].FoodDenial},
				{"StarvationRisk", g.turns[i].StarvationRisk - g.turns[i-1].StarvationRisk},
				{"GrowthUrgency", g.turns[i].GrowthUrgency - g.turns[i-1].GrowthUrgency},
				{"TailChase", g.turns[i].TailChase - g.turns[i-1].TailChase},
				{"Bottleneck", g.turns[i].Bottleneck - g.turns[i-1].Bottleneck},
			}
			sort.Slice(deltas, func(a, b int) bool {
				return math.Abs(deltas[a].delta) > math.Abs(deltas[b].delta)
			})

			tp := turningPoint{
				gameID:   g.header.GameID,
				turn:     *g.turns[i].Turn,
				prevEval: g.turns[i-1].Eval,
				currEval: g.turns[i].Eval,
				swing:    swing,
			}
			if len(deltas) > 0 {
				tp.biggestSig = deltas[0].name
				tp.biggestDrop = deltas[0].delta
			}
			if len(deltas) > 1 {
				tp.secondSig = deltas[1].name
				tp.secondDrop = deltas[1].delta
			}
			if g.footer.Result == "loss" {
				tp.deathTurn = g.footer.TotalTurns
				tp.deathCause = g.footer.DeathCause
			}
			points = append(points, tp)
		}
	}

	sort.Slice(points, func(i, j int) bool {
		return math.Abs(points[i].swing) > math.Abs(points[j].swing)
	})

	if top > len(points) {
		top = len(points)
	}
	for _, tp := range points[:top] {
		prefix := tp.gameID
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
		fmt.Printf("Game %s turn %d: eval %+.1f → %+.1f (swing: %+.1f)\n",
			prefix, tp.turn, tp.prevEval, tp.currEval, tp.swing)
		fmt.Printf("  %-16s %+.1f  (biggest)\n", tp.biggestSig+":", tp.biggestDrop)
		if tp.secondSig != "" {
			fmt.Printf("  %-16s %+.1f  (second)\n", tp.secondSig+":", tp.secondDrop)
		}
		if tp.deathTurn > 0 {
			fmt.Printf("  Died %d turns later at turn %d (%s)\n",
				tp.deathTurn-tp.turn, tp.deathTurn, tp.deathCause)
		}
		fmt.Println()
	}
}

func modeDeaths(games []game, top int) {
	count := 0
	for _, g := range games {
		if g.footer.Result != "loss" || len(g.turns) == 0 {
			continue
		}
		count++
		if count > top {
			break
		}

		prefix := g.header.GameID
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
		fmt.Printf("LOSS: %s at turn %d (game %s)\n",
			g.footer.DeathCause, g.footer.TotalTurns, prefix)

		// Show last 10 turns.
		start := len(g.turns) - 10
		if start < 0 {
			start = 0
		}

		// Track largest signal drop over the window.
		type sigAccum struct {
			name      string
			firstVal  float64
			lastVal   float64
		}
		sigs := []sigAccum{
			{"territory", g.turns[start].Territory, 0},
			{"len_adv", g.turns[start].LenAdvantage, 0},
			{"h2h", g.turns[start].H2H, 0},
			{"confine", g.turns[start].OppConfinement + g.turns[start].SelfConfinement, 0},
			{"food", g.turns[start].FoodUrgency + g.turns[start].FoodCluster + g.turns[start].FoodReach, 0},
		}

		for _, t := range g.turns[start:] {
			fmt.Printf("  Turn %3d: eval %+7.1f  territory=%+.0f  h2h=%+.0f  confine=%+.0f  depth=%d\n",
				*t.Turn, t.Eval, t.Territory, t.H2H,
				t.OppConfinement+t.SelfConfinement, t.Depth)
		}

		last := g.turns[len(g.turns)-1]
		sigs[0].lastVal = last.Territory
		sigs[1].lastVal = last.LenAdvantage
		sigs[2].lastVal = last.H2H
		sigs[3].lastVal = last.OppConfinement + last.SelfConfinement
		sigs[4].lastVal = last.FoodUrgency + last.FoodCluster + last.FoodReach

		// Find biggest drop.
		biggestName := ""
		biggestDrop := 0.0
		for _, s := range sigs {
			drop := s.lastVal - s.firstVal
			if math.Abs(drop) > math.Abs(biggestDrop) {
				biggestDrop = drop
				biggestName = s.name
			}
		}
		if biggestName != "" {
			fmt.Printf("  Largest drop: %s (%+.1f over last %d turns)\n",
				biggestName, biggestDrop, len(g.turns)-start)
		}
		fmt.Println()
	}
	if count == 0 {
		fmt.Println("No losses found.")
	}
}

func modeSignals(games []game) {
	type signalStats struct {
		name     string
		winSum   float64
		winCount int
		lossSum  float64
		lossCount int
	}

	sigNames := []string{
		"Territory", "LenAdvantage", "H2H", "OppConfinement",
		"SelfConfinement", "FoodUrgency", "FoodCluster", "FoodReach",
		"FoodDenial", "StarvationRisk", "GrowthUrgency", "TailChase",
		"Bottleneck",
	}
	stats := make(map[string]*signalStats)
	for _, name := range sigNames {
		stats[name] = &signalStats{name: name}
	}

	getSig := func(r record, name string) float64 {
		switch name {
		case "Territory":
			return r.Territory
		case "LenAdvantage":
			return r.LenAdvantage
		case "H2H":
			return r.H2H
		case "OppConfinement":
			return r.OppConfinement
		case "SelfConfinement":
			return r.SelfConfinement
		case "FoodUrgency":
			return r.FoodUrgency
		case "FoodCluster":
			return r.FoodCluster
		case "FoodReach":
			return r.FoodReach
		case "FoodDenial":
			return r.FoodDenial
		case "StarvationRisk":
			return r.StarvationRisk
		case "GrowthUrgency":
			return r.GrowthUrgency
		case "TailChase":
			return r.TailChase
		case "Bottleneck":
			return r.Bottleneck
		}
		return 0
	}

	for _, g := range games {
		isWin := g.footer.Result == "win"
		isLoss := g.footer.Result == "loss"
		if !isWin && !isLoss {
			continue
		}
		for _, t := range g.turns {
			for _, name := range sigNames {
				val := getSig(t, name)
				s := stats[name]
				if isWin {
					s.winSum += val
					s.winCount++
				} else {
					s.lossSum += val
					s.lossCount++
				}
			}
		}
	}

	fmt.Println("Signal avg contribution (wins vs losses):")
	fmt.Printf("  %-18s %8s  %8s  %s\n", "Signal", "Wins", "Losses", "Note")
	fmt.Printf("  %-18s %8s  %8s  %s\n", "------", "----", "------", "----")
	for _, name := range sigNames {
		s := stats[name]
		winAvg := 0.0
		if s.winCount > 0 {
			winAvg = s.winSum / float64(s.winCount)
		}
		lossAvg := 0.0
		if s.lossCount > 0 {
			lossAvg = s.lossSum / float64(s.lossCount)
		}
		note := ""
		if math.Abs(lossAvg) > math.Abs(winAvg)*1.5 && math.Abs(lossAvg) > 0.5 {
			note = "<- strong in losses"
		} else if math.Abs(winAvg) > math.Abs(lossAvg)*1.5 && math.Abs(winAvg) > 0.5 {
			note = "<- strong in wins"
		}
		fmt.Printf("  %-18s %+8.2f  %+8.2f  %s\n", name, winAvg, lossAvg, note)
	}
}

func main() {
	mode := flag.String("mode", "summary", "summary | turning-points | deaths | signals")
	top := flag.Int("top", 10, "number of items to show")
	threshold := flag.Float64("threshold", 15.0, "eval swing threshold for turning points")
	flag.Parse()

	files := flag.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: go run ./cmd/analyze [flags] traces/*.jsonl")
		os.Exit(1)
	}

	games := loadGames(files)
	if len(games) == 0 {
		fmt.Fprintln(os.Stderr, "No valid trace files found.")
		os.Exit(1)
	}

	switch *mode {
	case "summary":
		modeSummary(games)
	case "turning-points":
		modeTurningPoints(games, *threshold, *top)
	case "deaths":
		modeDeaths(games, *top)
	case "signals":
		modeSignals(games)
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}
