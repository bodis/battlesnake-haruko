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
	Connectivity    float64 `json:"connectivity"`
	MyConnectivity  float64 `json:"my_connectivity"`
	OppConnectivity float64 `json:"opp_connectivity"`
	EarlyBlend      float64 `json:"early_blend"`
	LateBlend       float64 `json:"late_blend"`

	// Diagnostic: territory depth profile
	MyNearTerritory  int `json:"my_near_territory"`
	MyFarTerritory   int `json:"my_far_territory"`
	OppNearTerritory int `json:"opp_near_territory"`
	OppFarTerritory  int `json:"opp_far_territory"`

	// Diagnostic: territory shape
	MyCorridorCells  int `json:"my_corridor_cells"`
	OppCorridorCells int `json:"opp_corridor_cells"`

	// Diagnostic: escape reachability
	MyEscapeRoutes  int `json:"my_escape_routes"`
	OppEscapeRoutes int `json:"opp_escape_routes"`

	// Diagnostic: tail reachability and loopability
	MyTailReachable  bool `json:"my_tail_reachable"`
	MyTailBFSDist    int  `json:"my_tail_bfs_dist"`
	OppTailReachable bool `json:"opp_tail_reachable"`
	OppTailBFSDist   int  `json:"opp_tail_bfs_dist"`
	MyLoopable       bool `json:"my_loopable"`
	OppLoopable      bool `json:"opp_loopable"`
	TurnsToDeathEst  int  `json:"turns_to_death_est"`

	// Diagnostic: derived ratios
	MyFunnelRatio    float64 `json:"my_funnel_ratio"`
	MyCorridorRatio  float64 `json:"my_corridor_ratio"`
	MyEscapeRatio    float64 `json:"my_escape_ratio"`
	OppFunnelRatio   float64 `json:"opp_funnel_ratio"`
	OppCorridorRatio float64 `json:"opp_corridor_ratio"`
	OppEscapeRatio   float64 `json:"opp_escape_ratio"`

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

// modeCorrelation analyzes diagnostic metrics for correlation with game outcome.
// Computes per-game averages, win vs loss comparison, pre-decision-point trends,
// and Pearson correlation for each diagnostic metric.
func modeCorrelation(games []game) {
	type metricDef struct {
		name string
		get  func(r record) float64
	}
	metrics := []metricDef{
		// Raw counts
		{"MyNearTerritory", func(r record) float64 { return float64(r.MyNearTerritory) }},
		{"MyFarTerritory", func(r record) float64 { return float64(r.MyFarTerritory) }},
		{"OppNearTerritory", func(r record) float64 { return float64(r.OppNearTerritory) }},
		{"OppFarTerritory", func(r record) float64 { return float64(r.OppFarTerritory) }},
		{"MyCorridorCells", func(r record) float64 { return float64(r.MyCorridorCells) }},
		{"OppCorridorCells", func(r record) float64 { return float64(r.OppCorridorCells) }},
		{"MyEscapeRoutes", func(r record) float64 { return float64(r.MyEscapeRoutes) }},
		{"OppEscapeRoutes", func(r record) float64 { return float64(r.OppEscapeRoutes) }},
		// Ratios
		{"MyFunnelRatio", func(r record) float64 { return r.MyFunnelRatio }},
		{"MyCorridorRatio", func(r record) float64 { return r.MyCorridorRatio }},
		{"MyEscapeRatio", func(r record) float64 { return r.MyEscapeRatio }},
		{"OppFunnelRatio", func(r record) float64 { return r.OppFunnelRatio }},
		{"OppCorridorRatio", func(r record) float64 { return r.OppCorridorRatio }},
		{"OppEscapeRatio", func(r record) float64 { return r.OppEscapeRatio }},
		// Existing signals for comparison
		{"MyConnectivity", func(r record) float64 { return r.MyConnectivity }},
		{"OppConnectivity", func(r record) float64 { return r.OppConnectivity }},
		{"MyTerritory", func(r record) float64 { return float64(r.MyTerritory) }},
		{"OppTerritory", func(r record) float64 { return float64(r.OppTerritory) }},
	}

	// --- Section 1: Per-game average comparison (wins vs losses) ---
	type metricAccum struct {
		winSum, lossSum     float64
		winCount, lossCount int
	}
	avgStats := make([]metricAccum, len(metrics))

	// --- Section 2: Pearson correlation with outcome ---
	// outcome: 1.0 for win, 0.0 for loss
	type corrData struct {
		xs []float64 // per-game metric average
		ys []float64 // outcome (1=win, 0=loss)
	}
	corrStats := make([]corrData, len(metrics))

	// --- Section 3: Pre-death/pre-kill trends (last 20 turns) ---
	type trendAccum struct {
		winDeltaSum, lossDeltaSum     float64
		winDeltaCount, lossDeltaCount int
	}
	trendStats := make([]trendAccum, len(metrics))

	// --- Section 4: Late-game (turn 250+) comparison ---
	lateStats := make([]metricAccum, len(metrics))

	for _, g := range games {
		if g.footer.Result != "win" && g.footer.Result != "loss" {
			continue
		}
		isWin := g.footer.Result == "win"
		outcome := 0.0
		if isWin {
			outcome = 1.0
		}

		if len(g.turns) == 0 {
			continue
		}

		// Per-game average for each metric.
		for mi, m := range metrics {
			sum := 0.0
			n := 0
			for _, t := range g.turns {
				sum += m.get(t)
				n++
			}
			if n == 0 {
				continue
			}
			avg := sum / float64(n)
			if isWin {
				avgStats[mi].winSum += avg
				avgStats[mi].winCount++
			} else {
				avgStats[mi].lossSum += avg
				avgStats[mi].lossCount++
			}
			corrStats[mi].xs = append(corrStats[mi].xs, avg)
			corrStats[mi].ys = append(corrStats[mi].ys, outcome)
		}

		// Pre-decision trend: last 20 turns slope.
		window := 20
		if len(g.turns) >= window {
			startTurns := g.turns[len(g.turns)-window:]
			for mi, m := range metrics {
				first := m.get(startTurns[0])
				last := m.get(startTurns[window-1])
				delta := last - first
				if isWin {
					trendStats[mi].winDeltaSum += delta
					trendStats[mi].winDeltaCount++
				} else {
					trendStats[mi].lossDeltaSum += delta
					trendStats[mi].lossDeltaCount++
				}
			}
		}

		// Late-game stats (turn 250+).
		for mi, m := range metrics {
			sum := 0.0
			n := 0
			for _, t := range g.turns {
				if t.Turn != nil && *t.Turn >= 250 {
					sum += m.get(t)
					n++
				}
			}
			if n > 0 {
				avg := sum / float64(n)
				if isWin {
					lateStats[mi].winSum += avg
					lateStats[mi].winCount++
				} else {
					lateStats[mi].lossSum += avg
					lateStats[mi].lossCount++
				}
			}
		}
	}

	// Pearson correlation.
	pearson := func(xs, ys []float64) float64 {
		n := len(xs)
		if n < 3 {
			return 0
		}
		var sumX, sumY, sumXY, sumX2, sumY2 float64
		for i := 0; i < n; i++ {
			sumX += xs[i]
			sumY += ys[i]
			sumXY += xs[i] * ys[i]
			sumX2 += xs[i] * xs[i]
			sumY2 += ys[i] * ys[i]
		}
		fn := float64(n)
		num := fn*sumXY - sumX*sumY
		den := math.Sqrt((fn*sumX2 - sumX*sumX) * (fn*sumY2 - sumY*sumY))
		if den == 0 {
			return 0
		}
		return num / den
	}

	// --- Output ---
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════")
	fmt.Println("DIAGNOSTIC METRIC CORRELATION ANALYSIS")
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════")

	// Section 1: Game averages
	fmt.Println("\n1. GAME-AVERAGE COMPARISON (wins vs losses)")
	fmt.Printf("  %-22s  %8s  %8s  %8s  %6s\n", "Metric", "Wins", "Losses", "Delta", "Corr")
	fmt.Printf("  %-22s  %8s  %8s  %8s  %6s\n", "------", "----", "------", "-----", "----")
	for mi, m := range metrics {
		winAvg := 0.0
		if avgStats[mi].winCount > 0 {
			winAvg = avgStats[mi].winSum / float64(avgStats[mi].winCount)
		}
		lossAvg := 0.0
		if avgStats[mi].lossCount > 0 {
			lossAvg = avgStats[mi].lossSum / float64(avgStats[mi].lossCount)
		}
		corr := pearson(corrStats[mi].xs, corrStats[mi].ys)

		marker := "  "
		if math.Abs(corr) >= 0.3 {
			marker = "**"
		} else if math.Abs(corr) >= 0.15 {
			marker = "* "
		}
		_ = m
		fmt.Printf("  %-22s  %8.2f  %8.2f  %+8.2f  %+.3f %s\n",
			m.name, winAvg, lossAvg, winAvg-lossAvg, corr, marker)
	}
	fmt.Println("  (* = r≥0.15, ** = r≥0.30)")

	// Section 2: Pre-death/pre-kill trends
	fmt.Println("\n2. PRE-OUTCOME TREND (last 20 turns: end - start)")
	fmt.Printf("  %-22s  %8s  %8s  %8s\n", "Metric", "PreWin", "PreLoss", "Delta")
	fmt.Printf("  %-22s  %8s  %8s  %8s\n", "------", "------", "-------", "-----")
	for mi, m := range metrics {
		winDelta := 0.0
		if trendStats[mi].winDeltaCount > 0 {
			winDelta = trendStats[mi].winDeltaSum / float64(trendStats[mi].winDeltaCount)
		}
		lossDelta := 0.0
		if trendStats[mi].lossDeltaCount > 0 {
			lossDelta = trendStats[mi].lossDeltaSum / float64(trendStats[mi].lossDeltaCount)
		}
		if math.Abs(winDelta)+math.Abs(lossDelta) < 0.01 {
			continue // skip metrics with no movement
		}
		_ = m
		fmt.Printf("  %-22s  %+8.2f  %+8.2f  %+8.2f\n",
			m.name, winDelta, lossDelta, winDelta-lossDelta)
	}

	// Section 3: Late-game comparison
	fmt.Println("\n3. LATE-GAME AVERAGES (turn 250+)")
	fmt.Printf("  %-22s  %8s  %8s  %8s\n", "Metric", "Wins", "Losses", "Delta")
	fmt.Printf("  %-22s  %8s  %8s  %8s\n", "------", "----", "------", "-----")
	for mi, m := range metrics {
		winAvg := 0.0
		if lateStats[mi].winCount > 0 {
			winAvg = lateStats[mi].winSum / float64(lateStats[mi].winCount)
		}
		lossAvg := 0.0
		if lateStats[mi].lossCount > 0 {
			lossAvg = lateStats[mi].lossSum / float64(lateStats[mi].lossCount)
		}
		if winAvg == 0 && lossAvg == 0 {
			continue
		}
		_ = m
		fmt.Printf("  %-22s  %8.2f  %8.2f  %+8.2f\n",
			m.name, winAvg, lossAvg, winAvg-lossAvg)
	}

	// Section 4: Per-game detail — show worst metric trajectories for losses
	fmt.Println("\n4. LOSS TRAJECTORIES (diagnostic metrics, sampled every 50 turns)")
	lossCount := 0
	for _, g := range games {
		if g.footer.Result != "loss" || len(g.turns) < 10 {
			continue
		}
		lossCount++
		if lossCount > 5 {
			break
		}
		prefix := g.header.GameID
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
		fmt.Printf("\n  LOSS %s at turn %d (%s)\n", prefix, g.footer.TotalTurns, g.footer.DeathCause)
		fmt.Printf("  %5s  %5s %5s  %5s %5s  %5s %5s  %5s %5s  %6s %6s  %6s %6s\n",
			"Turn", "MyTer", "OpTer", "MyNr", "MyFar", "MyCor", "OpCor", "MyEsc", "OpEsc",
			"MyFnl", "MyCorR", "MyEscR", "OpEscR")
		for _, t := range g.turns {
			if t.Turn == nil {
				continue
			}
			turn := *t.Turn
			if turn%50 != 0 && turn != g.footer.TotalTurns-1 {
				continue
			}
			fmt.Printf("  %5d  %5d %5d  %5d %5d  %5d %5d  %5d %5d  %6.2f %6.2f  %6.2f %6.2f\n",
				turn, t.MyTerritory, t.OppTerritory,
				t.MyNearTerritory, t.MyFarTerritory,
				t.MyCorridorCells, t.OppCorridorCells,
				t.MyEscapeRoutes, t.OppEscapeRoutes,
				t.MyFunnelRatio, t.MyCorridorRatio,
				t.MyEscapeRatio, t.OppEscapeRatio)
		}
	}
}

// modeDecisionPoints traces backward from death/kill to find:
// - When each diagnostic metric started deteriorating (inflection point)
// - How many turns before death that was (vs 7-turn search horizon)
// - Whether the winner's metrics showed attack opportunity at the same time
// - Whether there was a "last chance" where the loser still had options
func modeDecisionPoints(games []game, top int) {
	const horizon = 7 // full rounds of lookahead (14 plies / 2)

	type metricDef struct {
		name      string
		get       func(r record) float64
		declining bool // true = metric drops before loss, false = rises
	}
	metrics := []metricDef{
		{"MyTerritory", func(r record) float64 { return float64(r.MyTerritory) }, true},
		{"MyFarTerritory", func(r record) float64 { return float64(r.MyFarTerritory) }, true},
		{"MyEscapeRoutes", func(r record) float64 { return float64(r.MyEscapeRoutes) }, true},
		{"MyConnectivity", func(r record) float64 { return r.MyConnectivity }, true},
		{"MyCorridorRatio", func(r record) float64 { return r.MyCorridorRatio }, false},
		{"MyFunnelRatio", func(r record) float64 { return r.MyFunnelRatio }, false},
		{"MyEscapeRatio", func(r record) float64 { return r.MyEscapeRatio }, false},
	}

	// For attack analysis: mirror metrics from opponent side
	attackMetrics := []metricDef{
		{"OppTerritory", func(r record) float64 { return float64(r.OppTerritory) }, true},
		{"OppFarTerritory", func(r record) float64 { return float64(r.OppFarTerritory) }, true},
		{"OppEscapeRoutes", func(r record) float64 { return float64(r.OppEscapeRoutes) }, true},
		{"OppConnectivity", func(r record) float64 { return r.OppConnectivity }, true},
		{"OppCorridorRatio", func(r record) float64 { return r.OppCorridorRatio }, false},
		{"OppEscapeRatio", func(r record) float64 { return r.OppEscapeRatio }, false},
	}

	// findInflection walks backward from the end to find where the metric
	// started its sustained decline (or rise for inverted metrics).
	// Uses a smoothed window of 5 turns to avoid noise.
	// Returns the turn index where decline began, or -1 if no clear inflection.
	findInflection := func(turns []record, m metricDef) int {
		n := len(turns)
		if n < 10 {
			return -1
		}

		// Compute smoothed values (5-turn moving average).
		smooth := make([]float64, n)
		for i := 0; i < n; i++ {
			sum := 0.0
			count := 0
			for j := i - 2; j <= i+2; j++ {
				if j >= 0 && j < n {
					sum += m.get(turns[j])
					count++
				}
			}
			smooth[i] = sum / float64(count)
		}

		// Walk backward from end to find where the trend reversed.
		// Look for the peak (for declining metrics) or trough (for rising metrics).
		endVal := smooth[n-1]
		bestIdx := n - 1
		bestVal := endVal

		for i := n - 2; i >= 0; i-- {
			if m.declining {
				// For declining metrics, find the peak before the drop
				if smooth[i] > bestVal {
					bestVal = smooth[i]
					bestIdx = i
				}
				// Stop if we've found a significant peak and it's well above end value
				if bestVal-endVal > 5 && smooth[i] < bestVal*0.95 {
					break
				}
			} else {
				// For rising metrics (corridor ratio etc), find the trough before the rise
				if smooth[i] < bestVal {
					bestVal = smooth[i]
					bestIdx = i
				}
				if endVal-bestVal > 0.05 && smooth[i] > bestVal*1.05+0.01 {
					break
				}
			}
		}

		// Only report if there was a meaningful change
		if m.declining && bestVal-endVal < 3 {
			return -1
		}
		if !m.declining && endVal-bestVal < 0.03 {
			return -1
		}
		return bestIdx
	}

	// --- Aggregate stats ---
	type inflectionStat struct {
		turnsBeforeDeath []int
		withinHorizon    int
		beyondHorizon    int
		total            int
	}
	defenseStats := make([]inflectionStat, len(metrics))
	attackStats := make([]inflectionStat, len(attackMetrics))

	// Track "last chance" — the last turn where escape routes were still > 20
	type lastChance struct {
		turn             int
		turnsBeforeDeath int
		escapeRoutes     int
		territory        int
	}
	var lastChances []lastChance

	// --- Per-game analysis ---
	lossDetail := 0
	winDetail := 0
	for _, g := range games {
		if len(g.turns) < 10 {
			continue
		}

		prefix := g.header.GameID
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}

		if g.footer.Result == "loss" {
			deathTurn := g.footer.TotalTurns
			showDetail := lossDetail < top
			if showDetail {
				lossDetail++
				fmt.Printf("═══ LOSS %s at turn %d (%s) ═══\n", prefix, deathTurn, g.footer.DeathCause)
			}

			// Find inflection for each defense metric
			if showDetail {
				fmt.Printf("  DEFENSE (when did our metrics start deteriorating?):\n")
				fmt.Printf("  %-20s  %6s  %8s  %8s  %8s  %s\n",
					"Metric", "Peak@", "PeakVal", "EndVal", "Drop", "Horizon?")
			}
			for mi, m := range metrics {
				idx := findInflection(g.turns, m)
				if idx < 0 {
					continue
				}
				inflTurn := 0
				if g.turns[idx].Turn != nil {
					inflTurn = *g.turns[idx].Turn
				}
				turnsBeforeDeath := deathTurn - inflTurn
				peakVal := m.get(g.turns[idx])
				endVal := m.get(g.turns[len(g.turns)-1])

				defenseStats[mi].total++
				defenseStats[mi].turnsBeforeDeath = append(defenseStats[mi].turnsBeforeDeath, turnsBeforeDeath)
				if turnsBeforeDeath <= horizon {
					defenseStats[mi].withinHorizon++
				} else {
					defenseStats[mi].beyondHorizon++
				}

				if showDetail {
					horizonLabel := "WITHIN"
					if turnsBeforeDeath > horizon {
						horizonLabel = fmt.Sprintf("BEYOND (%d turns too late)", turnsBeforeDeath-horizon)
					}
					drop := peakVal - endVal
					if !m.declining {
						drop = endVal - peakVal
					}
					fmt.Printf("  %-20s  %6d  %8.1f  %8.1f  %+8.1f  %s\n",
						m.name, inflTurn, peakVal, endVal, drop, horizonLabel)
				}
			}

			// Find "last chance" — last turn with reasonable escape routes
			for i := len(g.turns) - 1; i >= 0; i-- {
				t := g.turns[i]
				if t.MyEscapeRoutes >= 20 && t.MyTerritory >= 20 {
					turn := 0
					if t.Turn != nil {
						turn = *t.Turn
					}
					lc := lastChance{
						turn:             turn,
						turnsBeforeDeath: deathTurn - turn,
						escapeRoutes:     t.MyEscapeRoutes,
						territory:        t.MyTerritory,
					}
					lastChances = append(lastChances, lc)
					if showDetail {
						fmt.Printf("\n  LAST CHANCE: turn %d (%d turns before death)\n",
							lc.turn, lc.turnsBeforeDeath)
						fmt.Printf("    EscapeRoutes=%d  Territory=%d\n", lc.escapeRoutes, lc.territory)
					}
					break
				}
			}

			// Show the critical 15-turn window around inflection
			if showDetail {
				// Find earliest inflection
				earliestInflection := len(g.turns) - 1
				for _, m := range metrics {
					idx := findInflection(g.turns, m)
					if idx >= 0 && idx < earliestInflection {
						earliestInflection = idx
					}
				}
				windowStart := earliestInflection - 5
				if windowStart < 0 {
					windowStart = 0
				}

				fmt.Printf("\n  CRITICAL WINDOW (5 turns before earliest inflection → death):\n")
				fmt.Printf("  %5s  %5s %5s  %5s %5s  %5s  %6s %6s  %+7s  %5s\n",
					"Turn", "MyTer", "OpTer", "MyFar", "MyEsc", "OpEsc",
					"MyCorR", "MyFnlR", "Eval", "Depth")
				for i := windowStart; i < len(g.turns); i++ {
					t := g.turns[i]
					if t.Turn == nil {
						continue
					}
					turn := *t.Turn
					// Show every turn in the first 20, then every 5th
					offset := i - windowStart
					if offset >= 20 && offset%5 != 0 && i != len(g.turns)-1 {
						continue
					}
					fmt.Printf("  %5d  %5d %5d  %5d %5d  %5d  %6.2f %6.2f  %+7.1f  %5d\n",
						turn, t.MyTerritory, t.OppTerritory,
						t.MyFarTerritory, t.MyEscapeRoutes, t.OppEscapeRoutes,
						t.MyCorridorRatio, t.MyFunnelRatio, t.Eval, t.Depth)
				}
				fmt.Println()
			}
		}

		// Attack analysis: for wins, find when opponent metrics started collapsing
		if g.footer.Result == "win" {
			killTurn := g.footer.TotalTurns
			showDetail := winDetail < top/2 // show fewer win details
			if showDetail {
				winDetail++
				fmt.Printf("═══ WIN  %s at turn %d ═══\n", prefix, killTurn)
				fmt.Printf("  ATTACK (when did opponent's metrics start deteriorating?):\n")
				fmt.Printf("  %-20s  %6s  %8s  %8s  %8s  %s\n",
					"Metric", "Peak@", "PeakVal", "EndVal", "Drop", "Horizon?")
			}
			for mi, m := range attackMetrics {
				idx := findInflection(g.turns, m)
				if idx < 0 {
					continue
				}
				inflTurn := 0
				if g.turns[idx].Turn != nil {
					inflTurn = *g.turns[idx].Turn
				}
				turnsBeforeKill := killTurn - inflTurn
				peakVal := m.get(g.turns[idx])
				endVal := m.get(g.turns[len(g.turns)-1])

				attackStats[mi].total++
				attackStats[mi].turnsBeforeDeath = append(attackStats[mi].turnsBeforeDeath, turnsBeforeKill)
				if turnsBeforeKill <= horizon {
					attackStats[mi].withinHorizon++
				} else {
					attackStats[mi].beyondHorizon++
				}

				if showDetail {
					horizonLabel := "WITHIN"
					if turnsBeforeKill > horizon {
						horizonLabel = fmt.Sprintf("BEYOND (+%d)", turnsBeforeKill-horizon)
					}
					drop := peakVal - endVal
					if !m.declining {
						drop = endVal - peakVal
					}
					fmt.Printf("  %-20s  %6d  %8.1f  %8.1f  %+8.1f  %s\n",
						m.name, inflTurn, peakVal, endVal, drop, horizonLabel)
				}
			}

			if showDetail {
				// Show last 15 turns of win
				fmt.Printf("\n  KILL SEQUENCE (last 15 turns):\n")
				fmt.Printf("  %5s  %5s %5s  %5s %5s  %5s  %6s %6s  %+7s\n",
					"Turn", "MyTer", "OpTer", "OpFar", "MyEsc", "OpEsc",
					"OpCorR", "OpFnlR", "Eval")
				start := len(g.turns) - 15
				if start < 0 {
					start = 0
				}
				for _, t := range g.turns[start:] {
					if t.Turn == nil {
						continue
					}
					fmt.Printf("  %5d  %5d %5d  %5d %5d  %5d  %6.2f %6.2f  %+7.1f\n",
						*t.Turn, t.MyTerritory, t.OppTerritory,
						t.OppFarTerritory, t.MyEscapeRoutes, t.OppEscapeRoutes,
						t.OppCorridorRatio, t.OppFunnelRatio, t.Eval)
				}
				fmt.Println()
			}
		}
	}

	// --- Aggregate summary ---
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════")
	fmt.Println("AGGREGATE: Inflection Points vs Search Horizon (7 turns)")
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════")

	median := func(vals []int) float64 {
		if len(vals) == 0 {
			return 0
		}
		sorted := make([]int, len(vals))
		copy(sorted, vals)
		sort.Ints(sorted)
		n := len(sorted)
		if n%2 == 0 {
			return float64(sorted[n/2-1]+sorted[n/2]) / 2
		}
		return float64(sorted[n/2])
	}

	percentile := func(vals []int, p float64) float64 {
		if len(vals) == 0 {
			return 0
		}
		sorted := make([]int, len(vals))
		copy(sorted, vals)
		sort.Ints(sorted)
		idx := int(float64(len(sorted)-1) * p)
		return float64(sorted[idx])
	}

	fmt.Println("\n  DEFENSE: When does our collapse start (turns before death)?")
	fmt.Printf("  %-20s  %5s  %7s  %7s  %7s  %8s  %8s\n",
		"Metric", "N", "Median", "P25", "P75", "≤7turns", ">7turns")
	fmt.Printf("  %-20s  %5s  %7s  %7s  %7s  %8s  %8s\n",
		"------", "--", "------", "---", "---", "-------", "-------")
	for mi, m := range metrics {
		s := defenseStats[mi]
		if s.total == 0 {
			continue
		}
		med := median(s.turnsBeforeDeath)
		p25 := percentile(s.turnsBeforeDeath, 0.25)
		p75 := percentile(s.turnsBeforeDeath, 0.75)
		pctWithin := 100 * float64(s.withinHorizon) / float64(s.total)
		pctBeyond := 100 * float64(s.beyondHorizon) / float64(s.total)
		_ = m
		fmt.Printf("  %-20s  %5d  %7.0f  %7.0f  %7.0f  %7.0f%%  %7.0f%%\n",
			m.name, s.total, med, p25, p75, pctWithin, pctBeyond)
	}

	fmt.Println("\n  ATTACK: When does opponent's collapse start (turns before kill)?")
	fmt.Printf("  %-20s  %5s  %7s  %7s  %7s  %8s  %8s\n",
		"Metric", "N", "Median", "P25", "P75", "≤7turns", ">7turns")
	fmt.Printf("  %-20s  %5s  %7s  %7s  %7s  %8s  %8s\n",
		"------", "--", "------", "---", "---", "-------", "-------")
	for mi, m := range attackMetrics {
		s := attackStats[mi]
		if s.total == 0 {
			continue
		}
		med := median(s.turnsBeforeDeath)
		p25 := percentile(s.turnsBeforeDeath, 0.25)
		p75 := percentile(s.turnsBeforeDeath, 0.75)
		pctWithin := 100 * float64(s.withinHorizon) / float64(s.total)
		pctBeyond := 100 * float64(s.beyondHorizon) / float64(s.total)
		_ = m
		fmt.Printf("  %-20s  %5d  %7.0f  %7.0f  %7.0f  %7.0f%%  %7.0f%%\n",
			m.name, s.total, med, p25, p75, pctWithin, pctBeyond)
	}

	// --- Horizon visibility analysis ---
	// For each loss, check what the diagnostic metrics looked like at:
	// - The inflection point (where collapse begins)
	// - 7 turns BEFORE the inflection point (what the search leaf would see)
	// - The stable "safe" average (turns 50-100 or similar)
	// This answers: can the eval at the search leaf distinguish danger from safety?
	fmt.Println("\n  HORIZON VISIBILITY: Can the eval see danger at the search leaf?")
	fmt.Println("  (Comparing metric values at inflection vs 7 turns before inflection vs stable)")

	type visibilityCase struct {
		metric          string
		safeVal         float64 // average during stable mid-game
		leafVal         float64 // value at inflection - 7 (what search leaf sees)
		inflectionVal   float64 // value at inflection point
		deathVal        float64 // value at death
		turnsBeforeDeath int
		detectable      bool    // leafVal is meaningfully different from safeVal
	}

	visMetrics := []metricDef{
		{"MyFarTerritory", func(r record) float64 { return float64(r.MyFarTerritory) }, true},
		{"MyEscapeRoutes", func(r record) float64 { return float64(r.MyEscapeRoutes) }, true},
		{"MyCorridorRatio", func(r record) float64 { return r.MyCorridorRatio }, false},
		{"MyFunnelRatio", func(r record) float64 { return r.MyFunnelRatio }, false},
		{"MyConnectivity", func(r record) float64 { return r.MyConnectivity }, true},
	}

	type visAgg struct {
		detectableCount int
		totalCount      int
		leafDeltas      []float64 // how far leaf value is from safe average (in safe stddevs)
	}
	visStats := make(map[string]*visAgg)
	for _, m := range visMetrics {
		visStats[m.name] = &visAgg{}
	}

	visGameCount := 0
	for _, g := range games {
		if g.footer.Result != "loss" || len(g.turns) < 30 {
			continue
		}
		visGameCount++
		showDetail := visGameCount <= 3

		if showDetail {
			prefix := g.header.GameID
			if len(prefix) > 8 {
				prefix = prefix[:8]
			}
			fmt.Printf("\n    LOSS %s (turn %d, %s):\n", prefix, g.footer.TotalTurns, g.footer.DeathCause)
			fmt.Printf("    %-20s  %8s  %8s  %8s  %8s  %s\n",
				"Metric", "Safe", "Leaf(-7)", "Inflect", "Death", "Visible?")
		}

		for _, m := range visMetrics {
			// Compute "safe" average (turns 20% to 50% of game)
			safeStart := len(g.turns) / 5
			safeEnd := len(g.turns) / 2
			if safeEnd <= safeStart {
				continue
			}
			safeSum := 0.0
			safeN := 0
			safeSqSum := 0.0
			for i := safeStart; i < safeEnd; i++ {
				v := m.get(g.turns[i])
				safeSum += v
				safeSqSum += v * v
				safeN++
			}
			if safeN < 5 {
				continue
			}
			safeAvg := safeSum / float64(safeN)
			safeVar := safeSqSum/float64(safeN) - safeAvg*safeAvg
			safeStd := math.Sqrt(math.Abs(safeVar))
			if safeStd < 0.5 {
				safeStd = 0.5 // minimum to avoid division by near-zero
			}

			// Find inflection
			inflIdx := findInflection(g.turns, m)
			if inflIdx < 0 || inflIdx < 7 {
				continue
			}

			inflVal := m.get(g.turns[inflIdx])
			deathVal := m.get(g.turns[len(g.turns)-1])

			// Leaf value: 7 turns before inflection (what search would evaluate)
			leafIdx := inflIdx - 7
			leafVal := m.get(g.turns[leafIdx])

			// Is the leaf value distinguishable from safe average?
			leafDelta := (leafVal - safeAvg) / safeStd
			if !m.declining {
				leafDelta = -leafDelta // invert so positive = danger direction
			}
			detectable := math.Abs(leafDelta) > 0.5 // more than 0.5 stddev from safe

			va := visStats[m.name]
			va.totalCount++
			va.leafDeltas = append(va.leafDeltas, leafDelta)
			if detectable {
				va.detectableCount++
			}

			inflTurn := 0
			if g.turns[inflIdx].Turn != nil {
				inflTurn = *g.turns[inflIdx].Turn
			}

			if showDetail {
				vis := "NO  (leaf looks normal)"
				if detectable {
					vis = fmt.Sprintf("YES (%.1f σ from safe)", math.Abs(leafDelta))
				}
				_ = inflTurn
				fmt.Printf("    %-20s  %8.1f  %8.1f  %8.1f  %8.1f  %s\n",
					m.name, safeAvg, leafVal, inflVal, deathVal, vis)
			}
		}
	}

	// Aggregate visibility stats
	fmt.Println("\n  AGGREGATE VISIBILITY (can eval at search leaf detect pre-collapse?)")
	fmt.Printf("    %-20s  %8s  %8s  %8s\n", "Metric", "Detect%", "AvgDelta", "N")
	fmt.Printf("    %-20s  %8s  %8s  %8s\n", "------", "-------", "--------", "--")
	for _, m := range visMetrics {
		va := visStats[m.name]
		if va.totalCount == 0 {
			continue
		}
		pct := 100 * float64(va.detectableCount) / float64(va.totalCount)
		avgDelta := 0.0
		for _, d := range va.leafDeltas {
			avgDelta += d
		}
		avgDelta /= float64(len(va.leafDeltas))
		marker := ""
		if pct >= 60 {
			marker = " ** ACTIONABLE"
		} else if pct >= 40 {
			marker = " *  PROMISING"
		}
		fmt.Printf("    %-20s  %7.0f%%  %+8.2f  %8d%s\n",
			m.name, pct, avgDelta, va.totalCount, marker)
	}
	fmt.Println("    (Detect% = fraction where leaf value is >0.5σ from safe-game average)")
	fmt.Println("    (AvgDelta = mean distance from safe, in σ units; positive = danger direction)")

	// Last chance analysis
	if len(lastChances) > 0 {
		fmt.Println("\n  LAST CHANCE: Last turn with EscapeRoutes≥20 & Territory≥20")
		sort.Slice(lastChances, func(i, j int) bool {
			return lastChances[i].turnsBeforeDeath < lastChances[j].turnsBeforeDeath
		})

		withinHorizon := 0
		totalTurnsBefore := 0
		for _, lc := range lastChances {
			totalTurnsBefore += lc.turnsBeforeDeath
			if lc.turnsBeforeDeath <= horizon {
				withinHorizon++
			}
		}
		avgBefore := float64(totalTurnsBefore) / float64(len(lastChances))
		med := median(func() []int {
			vals := make([]int, len(lastChances))
			for i, lc := range lastChances {
				vals[i] = lc.turnsBeforeDeath
			}
			return vals
		}())
		p25 := percentile(func() []int {
			vals := make([]int, len(lastChances))
			for i, lc := range lastChances {
				vals[i] = lc.turnsBeforeDeath
			}
			return vals
		}(), 0.25)
		p75 := percentile(func() []int {
			vals := make([]int, len(lastChances))
			for i, lc := range lastChances {
				vals[i] = lc.turnsBeforeDeath
			}
			return vals
		}(), 0.75)

		fmt.Printf("    N=%d  Avg=%.0f  Median=%.0f  P25=%.0f  P75=%.0f  Within horizon: %d/%d (%.0f%%)\n",
			len(lastChances), avgBefore, med, p25, p75,
			withinHorizon, len(lastChances),
			100*float64(withinHorizon)/float64(len(lastChances)))

		fmt.Println("\n    Distribution of 'last chance' timing:")
		buckets := map[string]int{
			"1-3 turns":   0,
			"4-7 turns":   0,
			"8-15 turns":  0,
			"16-30 turns": 0,
			"31+ turns":   0,
		}
		bucketOrder := []string{"1-3 turns", "4-7 turns", "8-15 turns", "16-30 turns", "31+ turns"}
		for _, lc := range lastChances {
			switch {
			case lc.turnsBeforeDeath <= 3:
				buckets["1-3 turns"]++
			case lc.turnsBeforeDeath <= 7:
				buckets["4-7 turns"]++
			case lc.turnsBeforeDeath <= 15:
				buckets["8-15 turns"]++
			case lc.turnsBeforeDeath <= 30:
				buckets["16-30 turns"]++
			default:
				buckets["31+ turns"]++
			}
		}
		for _, b := range bucketOrder {
			bar := ""
			for i := 0; i < buckets[b]; i++ {
				bar += "█"
			}
			fmt.Printf("    %-14s %3d  %s\n", b, buckets[b], bar)
		}

		fmt.Println("\n    INTERPRETATION:")
		if float64(withinHorizon)/float64(len(lastChances)) > 0.5 {
			fmt.Println("    > Majority of collapses have last-chance within horizon.")
			fmt.Println("    > The engine CAN see the danger — eval just doesn't weight it enough.")
			fmt.Println("    > Fix: add escape/territory-depth signals to eval.")
		} else {
			pctBeyondDouble := 0
			for _, lc := range lastChances {
				if lc.turnsBeforeDeath > horizon*2 {
					pctBeyondDouble++
				}
			}
			if float64(pctBeyondDouble)/float64(len(lastChances)) > 0.3 {
				fmt.Println("    > Collapse starts far beyond horizon (>14 turns before death).")
				fmt.Println("    > The engine CANNOT see the danger through search alone.")
				fmt.Println("    > Fix: eval must recognize dangerous shapes (funnel/corridor/escape)")
				fmt.Println("    >       as static positional weakness, not just terminal collapse.")
			} else {
				fmt.Println("    > Mixed: some collapses are within reach, some aren't.")
				fmt.Println("    > Two-pronged fix needed:")
				fmt.Println("    >   1. Eval signals for early warning (static shape detection)")
				fmt.Println("    >   2. Stronger weighting when metrics approach danger thresholds")
			}
		}
	}
}

func main() {
	mode := flag.String("mode", "summary", "summary | turning-points | deaths | wins | signals | trajectories | correlation | decision-points")
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
	case "correlation":
		modeCorrelation(games)
	case "decision-points":
		modeDecisionPoints(games, *top)
	case "survival":
		modeSurvival(games)
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}

func modeSurvival(games []game) {
	// Section 1: Tail Reachability Distribution (wins vs losses)
	fmt.Println("=== Tail Reachability Distribution ===")
	type distStats struct {
		count         int
		reachable     int
		distSum       int
		distMin       int
		distMax       int
	}
	winStats := distStats{distMin: math.MaxInt32}
	lossStats := distStats{distMin: math.MaxInt32}

	for _, g := range games {
		isWin := g.footer.Result == "win"
		isLoss := g.footer.Result == "loss"
		if !isWin && !isLoss {
			continue
		}
		st := &lossStats
		if isWin {
			st = &winStats
		}
		for _, t := range g.turns {
			if t.Turn == nil {
				continue
			}
			st.count++
			if t.MyTailReachable {
				st.reachable++
				if t.MyTailBFSDist > 0 {
					st.distSum += t.MyTailBFSDist
					if t.MyTailBFSDist < st.distMin {
						st.distMin = t.MyTailBFSDist
					}
					if t.MyTailBFSDist > st.distMax {
						st.distMax = t.MyTailBFSDist
					}
				}
			}
		}
	}

	printDistStats := func(label string, st distStats) {
		if st.count == 0 {
			fmt.Printf("  %s: no data\n", label)
			return
		}
		pct := 100.0 * float64(st.reachable) / float64(st.count)
		fmt.Printf("  %s: %d turns, %.1f%% tail-reachable", label, st.count, pct)
		if st.reachable > 0 {
			avg := float64(st.distSum) / float64(st.reachable)
			fmt.Printf(", avg dist=%.1f, min=%d, max=%d", avg, st.distMin, st.distMax)
		}
		fmt.Println()
	}
	printDistStats("Wins ", winStats)
	printDistStats("Losses", lossStats)

	// Section 2: Loopability as Death Predictor
	fmt.Println("\n=== Loopability as Death Predictor ===")
	lookbacks := []int{1, 2, 5, 10, 20, 50}
	notLoopableCounts := make([]int, len(lookbacks))
	totalCounts := make([]int, len(lookbacks))

	for _, g := range games {
		if g.footer.Result != "loss" {
			continue
		}
		turns := g.turns
		if len(turns) == 0 {
			continue
		}
		deathIdx := len(turns) - 1
		for i, lb := range lookbacks {
			checkIdx := deathIdx - lb
			if checkIdx < 0 {
				continue
			}
			totalCounts[i]++
			if !turns[checkIdx].MyLoopable {
				notLoopableCounts[i]++
			}
		}
	}

	fmt.Println("  Turns before death | !Loopable | Total | %")
	for i, lb := range lookbacks {
		if totalCounts[i] == 0 {
			continue
		}
		pct := 100.0 * float64(notLoopableCounts[i]) / float64(totalCounts[i])
		fmt.Printf("  %18d | %9d | %5d | %5.1f%%\n", lb, notLoopableCounts[i], totalCounts[i], pct)
	}

	// Section 3: Loopability Asymmetry
	fmt.Println("\n=== Loopability Asymmetry ===")
	type asymBuckets struct {
		bothLoop int
		mineOnly int
		oppOnly  int
		neither  int
	}
	var winAsym, lossAsym asymBuckets

	for _, g := range games {
		isWin := g.footer.Result == "win"
		isLoss := g.footer.Result == "loss"
		if !isWin && !isLoss {
			continue
		}
		ab := &lossAsym
		if isWin {
			ab = &winAsym
		}
		for _, t := range g.turns {
			if t.Turn == nil {
				continue
			}
			switch {
			case t.MyLoopable && t.OppLoopable:
				ab.bothLoop++
			case t.MyLoopable && !t.OppLoopable:
				ab.mineOnly++
			case !t.MyLoopable && t.OppLoopable:
				ab.oppOnly++
			default:
				ab.neither++
			}
		}
	}

	printAsym := func(label string, ab asymBuckets) {
		total := ab.bothLoop + ab.mineOnly + ab.oppOnly + ab.neither
		if total == 0 {
			fmt.Printf("  %s: no data\n", label)
			return
		}
		pct := func(n int) float64 { return 100.0 * float64(n) / float64(total) }
		fmt.Printf("  %s (%d turns): both=%.1f%%, mine-only=%.1f%%, opp-only=%.1f%%, neither=%.1f%%\n",
			label, total, pct(ab.bothLoop), pct(ab.mineOnly), pct(ab.oppOnly), pct(ab.neither))
	}
	printAsym("Wins  ", winAsym)
	printAsym("Losses", lossAsym)

	// Section 4: TurnsToDeathEst Accuracy
	fmt.Println("\n=== TurnsToDeathEst Accuracy ===")
	var errors []float64
	for _, g := range games {
		if g.footer.Result != "loss" {
			continue
		}
		turns := g.turns
		if len(turns) == 0 {
			continue
		}
		deathTurn := 0
		if turns[len(turns)-1].Turn != nil {
			deathTurn = *turns[len(turns)-1].Turn
		}
		for _, t := range turns {
			if t.Turn == nil || t.MyLoopable || t.TurnsToDeathEst <= 0 {
				continue
			}
			actual := deathTurn - *t.Turn
			if actual <= 0 {
				continue
			}
			errors = append(errors, float64(t.TurnsToDeathEst)-float64(actual))
		}
	}

	if len(errors) == 0 {
		fmt.Println("  No data points (no losses with !loopable + TurnsToDeathEst > 0)")
	} else {
		sumAbs := 0.0
		sumSigned := 0.0
		for _, e := range errors {
			sumSigned += e
			if e < 0 {
				sumAbs -= e
			} else {
				sumAbs += e
			}
		}
		n := float64(len(errors))
		mae := sumAbs / n
		bias := sumSigned / n
		fmt.Printf("  Data points: %d\n", len(errors))
		fmt.Printf("  Mean Absolute Error: %.1f turns\n", mae)
		fmt.Printf("  Mean Signed Error (bias): %.1f turns (positive = overestimate)\n", bias)
	}
}
