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
	GrowthUrgency   float64 `json:"growth_urgency"`
	TailChase       float64 `json:"tail_chase"`
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

func classifyWin(g game) string {
	// Check self-destruct: avg eval over last 5 turns < 5.0
	if len(g.turns) > 0 {
		start := len(g.turns) - 5
		if start < 0 {
			start = 0
		}
		sum := 0.0
		for _, t := range g.turns[start:] {
			sum += t.Eval
		}
		avg := sum / float64(len(g.turns[start:]))
		if avg < 5.0 {
			return "self-destruct"
		}
	}

	// Find dominant signal group at final turn.
	if len(g.turns) == 0 {
		return "unknown"
	}
	last := g.turns[len(g.turns)-1]

	territoryGroup := math.Abs(last.Territory) + math.Abs(last.OppConfinement)
	h2hGroup := math.Abs(last.H2H)

	if territoryGroup >= h2hGroup {
		return "territory-squeeze"
	}
	return "h2h-kill"
}

func modeSummary(games []game) {
	wins, losses, draws := 0, 0, 0
	deathCounts := map[string]int{}
	var winTurns, lossTurns []int
	earlyDeaths, midDeaths, lateDeaths := 0, 0, 0
	winClassCounts := map[string]int{}

	for _, g := range games {
		switch g.footer.Result {
		case "win":
			wins++
			winTurns = append(winTurns, g.footer.TotalTurns)
			winClassCounts[classifyWin(g)]++
		case "loss":
			losses++
			lossTurns = append(lossTurns, g.footer.TotalTurns)
			if g.footer.DeathCause != "" {
				deathCounts[g.footer.DeathCause]++
			}
			switch {
			case g.footer.TotalTurns < 50:
				earlyDeaths++
			case g.footer.TotalTurns <= 200:
				midDeaths++
			default:
				lateDeaths++
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

	if losses > 0 {
		fmt.Printf("Death phase: early(<50)=%d  mid(50-200)=%d  late(>200)=%d\n",
			earlyDeaths, midDeaths, lateDeaths)
	}

	if wins > 0 {
		fmt.Printf("Win types:")
		for class, count := range winClassCounts {
			fmt.Printf(" %s=%d", class, count)
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
				{"GrowthUrgency", g.turns[i].GrowthUrgency - g.turns[i-1].GrowthUrgency},
				{"TailChase", g.turns[i].TailChase - g.turns[i-1].TailChase},
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
			{"food", g.turns[start].FoodUrgency + g.turns[start].FoodCluster, 0},
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
		sigs[4].lastVal = last.FoodUrgency + last.FoodCluster

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

func modeWins(games []game, top int) {
	classCounts := map[string]int{}
	count := 0
	for _, g := range games {
		if g.footer.Result != "win" || len(g.turns) == 0 {
			continue
		}
		count++
		class := classifyWin(g)
		classCounts[class]++
		if count > top {
			continue // still count classifications but skip detail output
		}

		prefix := g.header.GameID
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
		fmt.Printf("WIN: %s at turn %d (game %s)\n", class, g.footer.TotalTurns, prefix)

		// Show last 10 turns.
		start := len(g.turns) - 10
		if start < 0 {
			start = 0
		}

		// Track largest signal gain over the window.
		type sigAccum struct {
			name     string
			firstVal float64
			lastVal  float64
		}
		sigs := []sigAccum{
			{"territory", g.turns[start].Territory, 0},
			{"len_adv", g.turns[start].LenAdvantage, 0},
			{"h2h", g.turns[start].H2H, 0},
			{"confine", g.turns[start].OppConfinement + g.turns[start].SelfConfinement, 0},
			{"food", g.turns[start].FoodUrgency + g.turns[start].FoodCluster, 0},
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
		sigs[4].lastVal = last.FoodUrgency + last.FoodCluster

		// Find biggest gain.
		biggestName := ""
		biggestGain := 0.0
		for _, s := range sigs {
			gain := s.lastVal - s.firstVal
			if math.Abs(gain) > math.Abs(biggestGain) {
				biggestGain = gain
				biggestName = s.name
			}
		}
		if biggestName != "" {
			fmt.Printf("  Largest gain: %s (%+.1f over last %d turns)\n",
				biggestName, biggestGain, len(g.turns)-start)
		}
		fmt.Println()
	}

	if count == 0 {
		fmt.Println("No wins found.")
		return
	}

	// Classification summary.
	fmt.Println("---")
	fmt.Printf("Win classification (%d wins):\n", count)
	for class, n := range classCounts {
		fmt.Printf("  %-20s %d (%.0f%%)\n", class, n, 100*float64(n)/float64(count))
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
		"SelfConfinement", "FoodUrgency", "FoodCluster", "GrowthUrgency",
		"TailChase",
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
		case "GrowthUrgency":
			return r.GrowthUrgency
		case "TailChase":
			return r.TailChase
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

func modeTrajectories(games []game) {
	// Phase boundaries.
	const earlyEnd = 100
	const midEnd = 250

	type phaseStats struct {
		avgTerritory    float64
		avgSelfConfine  float64
		avgOppConfine   float64
		avgEval         float64
		avgLenAdv       float64
		avgH2H          float64
		avgPartitioned  float64
		territoryTrend  float64 // slope: last - first territory in phase
		confineEvents   int     // turns with self-confinement < -10
		count           int
	}

	computePhase := func(turns []record, start, end int) phaseStats {
		var ps phaseStats
		var firstTerr, lastTerr float64
		first := true
		for _, t := range turns {
			if t.Turn == nil {
				continue
			}
			turn := *t.Turn
			if turn < start || turn >= end {
				continue
			}
			ps.avgTerritory += t.Territory
			ps.avgSelfConfine += t.SelfConfinement
			ps.avgOppConfine += t.OppConfinement
			ps.avgEval += t.Eval
			ps.avgLenAdv += t.LenAdvantage
			ps.avgH2H += t.H2H
			if t.IsPartitioned {
				ps.avgPartitioned++
			}
			if t.SelfConfinement < -10 {
				ps.confineEvents++
			}
			if first {
				firstTerr = t.Territory
				first = true
			}
			lastTerr = t.Territory
			first = false
			ps.count++
		}
		if ps.count > 0 {
			n := float64(ps.count)
			ps.avgTerritory /= n
			ps.avgSelfConfine /= n
			ps.avgOppConfine /= n
			ps.avgEval /= n
			ps.avgLenAdv /= n
			ps.avgH2H /= n
			ps.avgPartitioned /= n
			ps.territoryTrend = lastTerr - firstTerr
		}
		return ps
	}

	// Aggregate per outcome.
	type outcomeAgg struct {
		early, mid, late []phaseStats
	}
	winAgg := outcomeAgg{}
	lossAgg := outcomeAgg{}

	for _, g := range games {
		if len(g.turns) == 0 {
			continue
		}
		totalTurns := 0
		if g.footer.TotalTurns > 0 {
			totalTurns = g.footer.TotalTurns
		}

		early := computePhase(g.turns, 0, earlyEnd)
		mid := computePhase(g.turns, earlyEnd, midEnd)
		late := computePhase(g.turns, midEnd, totalTurns+1)

		prefix := g.header.GameID
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}

		if g.footer.Result == "loss" || g.footer.Result == "win" {
			fmt.Printf("%-4s %s turn=%d: ", g.footer.Result, prefix, totalTurns)
			if g.footer.DeathCause != "" {
				fmt.Printf("(%s) ", g.footer.DeathCause)
			}
			fmt.Println()

			for _, phase := range []struct {
				name string
				ps   phaseStats
			}{{"early", early}, {"mid", mid}, {"late", late}} {
				if phase.ps.count == 0 {
					continue
				}
				fmt.Printf("  %-5s (%3d turns): eval=%+6.1f  terr=%+5.1f  trend=%+5.1f  lenAdv=%+5.1f  h2h=%+4.1f  oppConf=%+4.1f  selfConf=%+5.1f  confEvents=%d  partitioned=%.0f%%\n",
					phase.name, phase.ps.count,
					phase.ps.avgEval, phase.ps.avgTerritory, phase.ps.territoryTrend,
					phase.ps.avgLenAdv, phase.ps.avgH2H,
					phase.ps.avgOppConfine, phase.ps.avgSelfConfine,
					phase.ps.confineEvents,
					phase.ps.avgPartitioned*100)
			}
			fmt.Println()
		}

		switch g.footer.Result {
		case "win":
			winAgg.early = append(winAgg.early, early)
			winAgg.mid = append(winAgg.mid, mid)
			winAgg.late = append(winAgg.late, late)
		case "loss":
			lossAgg.early = append(lossAgg.early, early)
			lossAgg.mid = append(lossAgg.mid, mid)
			lossAgg.late = append(lossAgg.late, late)
		}
	}

	// Print aggregate comparison.
	avgPhase := func(phases []phaseStats) phaseStats {
		var agg phaseStats
		if len(phases) == 0 {
			return agg
		}
		for _, ps := range phases {
			agg.avgTerritory += ps.avgTerritory
			agg.avgSelfConfine += ps.avgSelfConfine
			agg.avgOppConfine += ps.avgOppConfine
			agg.avgEval += ps.avgEval
			agg.avgLenAdv += ps.avgLenAdv
			agg.avgH2H += ps.avgH2H
			agg.avgPartitioned += ps.avgPartitioned
			agg.territoryTrend += ps.territoryTrend
			agg.confineEvents += ps.confineEvents
			agg.count += ps.count
		}
		n := float64(len(phases))
		agg.avgTerritory /= n
		agg.avgSelfConfine /= n
		agg.avgOppConfine /= n
		agg.avgEval /= n
		agg.avgLenAdv /= n
		agg.avgH2H /= n
		agg.avgPartitioned /= n
		agg.territoryTrend /= n
		return agg
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("AGGREGATE: Wins vs Losses by game phase")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	for _, phase := range []struct {
		name string
		win  []phaseStats
		loss []phaseStats
	}{
		{"EARLY (0-100)", winAgg.early, lossAgg.early},
		{"MID (100-250)", winAgg.mid, lossAgg.mid},
		{"LATE (250+)", winAgg.late, lossAgg.late},
	} {
		w := avgPhase(phase.win)
		l := avgPhase(phase.loss)
		fmt.Printf("\n%s  (wins=%d, losses=%d)\n", phase.name, len(phase.win), len(phase.loss))
		fmt.Printf("  %-14s  %8s  %8s  %8s\n", "", "Wins", "Losses", "Delta")
		fmt.Printf("  %-14s  %+8.1f  %+8.1f  %+8.1f\n", "eval", w.avgEval, l.avgEval, w.avgEval-l.avgEval)
		fmt.Printf("  %-14s  %+8.1f  %+8.1f  %+8.1f\n", "territory", w.avgTerritory, l.avgTerritory, w.avgTerritory-l.avgTerritory)
		fmt.Printf("  %-14s  %+8.1f  %+8.1f  %+8.1f\n", "terr trend", w.territoryTrend, l.territoryTrend, w.territoryTrend-l.territoryTrend)
		fmt.Printf("  %-14s  %+8.1f  %+8.1f  %+8.1f\n", "lenAdv", w.avgLenAdv, l.avgLenAdv, w.avgLenAdv-l.avgLenAdv)
		fmt.Printf("  %-14s  %+8.1f  %+8.1f  %+8.1f\n", "h2h", w.avgH2H, l.avgH2H, w.avgH2H-l.avgH2H)
		fmt.Printf("  %-14s  %+8.1f  %+8.1f  %+8.1f\n", "oppConfine", w.avgOppConfine, l.avgOppConfine, w.avgOppConfine-l.avgOppConfine)
		fmt.Printf("  %-14s  %+8.1f  %+8.1f  %+8.1f\n", "selfConfine", w.avgSelfConfine, l.avgSelfConfine, w.avgSelfConfine-l.avgSelfConfine)
		fmt.Printf("  %-14s  %8.0f%%  %8.0f%%\n", "partitioned", w.avgPartitioned*100, l.avgPartitioned*100)

		wEvents := 0
		for _, ps := range phase.win {
			wEvents += ps.confineEvents
		}
		lEvents := 0
		for _, ps := range phase.loss {
			lEvents += ps.confineEvents
		}
		wPerGame := 0.0
		if len(phase.win) > 0 {
			wPerGame = float64(wEvents) / float64(len(phase.win))
		}
		lPerGame := 0.0
		if len(phase.loss) > 0 {
			lPerGame = float64(lEvents) / float64(len(phase.loss))
		}
		fmt.Printf("  %-14s  %8.1f  %8.1f  (per game)\n", "confineEvents", wPerGame, lPerGame)
	}
}

func main() {
	mode := flag.String("mode", "summary", "summary | turning-points | deaths | wins | signals")
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
	case "wins":
		modeWins(games, *top)
	case "signals":
		modeSignals(games)
	case "trajectories":
		modeTrajectories(games)
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}
