package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/bodist/haruko/logic"
)

// traceEnabled is set once at startup from HARUKO_TRACE env var.
var traceEnabled = os.Getenv("HARUKO_TRACE") == "1"

// traceRecord is a single JSONL line (header, turn, or footer).
type traceRecord struct {
	Type string `json:"type"`

	// Header fields
	GameID    string `json:"game_id,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	MySnakeID string `json:"my_snake_id,omitempty"`

	// Turn fields
	Turn      *int    `json:"turn,omitempty"`
	Move      string  `json:"move,omitempty"`
	Eval      float64 `json:"eval,omitempty"`
	Depth     int     `json:"depth,omitempty"`
	MyHealth  int     `json:"my_health,omitempty"`
	MyLen     int     `json:"my_len,omitempty"`
	OppLen    int     `json:"opp_len,omitempty"`
	MyTerritory  int  `json:"my_territory,omitempty"`
	OppTerritory int  `json:"opp_territory,omitempty"`
	IsPartitioned bool `json:"is_partitioned,omitempty"`
	FoodCount int    `json:"food_count,omitempty"`

	// Eval breakdown
	Territory       float64 `json:"territory,omitempty"`
	LenAdvantage    float64 `json:"len_advantage,omitempty"`
	H2H             float64 `json:"h2h,omitempty"`
	OppConfinement  float64 `json:"opp_confinement,omitempty"`
	SelfConfinement float64 `json:"self_confinement,omitempty"`
	FoodUrgency     float64 `json:"food_urgency,omitempty"`
	FoodCluster     float64 `json:"food_cluster,omitempty"`
	GrowthUrgency   float64 `json:"growth_urgency,omitempty"`
	TailChase       float64 `json:"tail_chase,omitempty"`
	Connectivity    float64 `json:"connectivity,omitempty"`
	MyConnectivity  float64 `json:"my_connectivity,omitempty"`
	OppConnectivity float64 `json:"opp_connectivity,omitempty"`
	EarlyBlend      float64 `json:"early_blend,omitempty"`
	LateBlend       float64 `json:"late_blend,omitempty"`

	// Diagnostic: territory depth profile
	MyNearTerritory  int `json:"my_near_territory,omitempty"`
	MyFarTerritory   int `json:"my_far_territory,omitempty"`
	OppNearTerritory int `json:"opp_near_territory,omitempty"`
	OppFarTerritory  int `json:"opp_far_territory,omitempty"`

	// Diagnostic: territory shape
	MyCorridorCells  int `json:"my_corridor_cells,omitempty"`
	OppCorridorCells int `json:"opp_corridor_cells,omitempty"`

	// Diagnostic: escape reachability (head-only BFS, 6 steps)
	MyEscapeRoutes  int `json:"my_escape_routes,omitempty"`
	OppEscapeRoutes int `json:"opp_escape_routes,omitempty"`

	// Diagnostic: derived ratios
	MyFunnelRatio    float64 `json:"my_funnel_ratio,omitempty"`    // near / territory (high = compact)
	MyCorridorRatio  float64 `json:"my_corridor_ratio,omitempty"`  // corridor cells / territory
	MyEscapeRatio    float64 `json:"my_escape_ratio,omitempty"`    // escape / territory (low = trapped)
	OppFunnelRatio   float64 `json:"opp_funnel_ratio,omitempty"`
	OppCorridorRatio float64 `json:"opp_corridor_ratio,omitempty"`
	OppEscapeRatio   float64 `json:"opp_escape_ratio,omitempty"`

	// Footer fields
	Result     string `json:"result,omitempty"`
	DeathCause string `json:"death_cause,omitempty"`
	TotalTurns int    `json:"total_turns,omitempty"`
}

// traceGame holds buffered records for one game+snake perspective.
type traceGame struct {
	gameID  string
	snakeID string
	records []traceRecord
}

var (
	traceGames sync.Map // key: "gameID:snakeID" → *traceGame
)

func traceKey(gameID, snakeID string) string {
	return gameID + ":" + snakeID
}

func traceStart(gameID, snakeID string, width, height int) {
	if !traceEnabled {
		return
	}
	tg := &traceGame{
		gameID:  gameID,
		snakeID: snakeID,
		records: make([]traceRecord, 0, 512),
	}
	tg.records = append(tg.records, traceRecord{
		Type:      "header",
		GameID:    gameID,
		Width:     width,
		Height:    height,
		MySnakeID: snakeID,
	})
	traceGames.Store(traceKey(gameID, snakeID), tg)
}

func traceTurn(gameID, snakeID string, state GameState, sim *logic.GameSim, moveName string) {
	if !traceEnabled {
		return
	}
	v, ok := traceGames.Load(traceKey(gameID, snakeID))
	if !ok {
		return
	}
	tg := v.(*traceGame)

	myIdx := sim.ResolveIdx(snakeID)
	if myIdx == -1 {
		return
	}

	bd := logic.EvaluateDetailed(sim, myIdx)
	vr := logic.VoronoiTerritory(sim, myIdx, true)

	oppLen := 0
	oppIdx := -1
	for i := range sim.Snakes {
		if i != myIdx && sim.Snakes[i].IsAlive() {
			oppLen = sim.Snakes[i].Length
			oppIdx = i
			break
		}
	}

	// Escape reachability (head-only BFS, 6 steps).
	myEscape := logic.EscapeReachabilityPooled(sim, myIdx, 6)
	// Opponent escape: use allocating diagnostic version (trace-only, not hot path).
	oppEscape := 0
	if oppIdx >= 0 {
		oppEscape = logic.EscapeReachability(sim, oppIdx, 6)
	}

	// Derived ratios.
	myFunnelRatio := 0.0
	myCorridorRatio := 0.0
	myEscapeRatio := 0.0
	if vr.MyTerritory > 0 {
		myFunnelRatio = float64(vr.MyNearTerritory) / float64(vr.MyTerritory)
		myCorridorRatio = float64(vr.MyCorridorCells) / float64(vr.MyTerritory)
		myEscapeRatio = float64(myEscape) / float64(vr.MyTerritory)
	}
	oppFunnelRatio := 0.0
	oppCorridorRatio := 0.0
	oppEscapeRatio := 0.0
	if vr.OppTerritory > 0 {
		oppFunnelRatio = float64(vr.OppNearTerritory) / float64(vr.OppTerritory)
		oppCorridorRatio = float64(vr.OppCorridorCells) / float64(vr.OppTerritory)
		oppEscapeRatio = float64(oppEscape) / float64(vr.OppTerritory)
	}

	turn := state.Turn
	rec := traceRecord{
		Type:            "turn",
		Turn:            &turn,
		Move:            moveName,
		Eval:            bd.Total,
		Depth:           sim.LastCompletedDepth,
		MyHealth:        state.You.Health,
		MyLen:           state.You.Length,
		OppLen:          oppLen,
		MyTerritory:     vr.MyTerritory,
		OppTerritory:    vr.OppTerritory,
		IsPartitioned:   vr.IsPartitioned,
		FoodCount:       len(state.Board.Food),
		Territory:       bd.Territory,
		LenAdvantage:    bd.LenAdvantage,
		H2H:             bd.H2H,
		OppConfinement:  bd.OppConfinement,
		SelfConfinement: bd.SelfConfinement,
		FoodUrgency:     bd.FoodUrgency,
		FoodCluster:     bd.FoodCluster,
		GrowthUrgency:   bd.GrowthUrgency,
		TailChase:       bd.TailChase,
		Connectivity:    bd.Connectivity,
		MyConnectivity:  vr.MyConnectivity,
		OppConnectivity: vr.OppConnectivity,
		EarlyBlend:      bd.EarlyBlend,
		LateBlend:       bd.LateBlend,

		// Diagnostic fields
		MyNearTerritory:  vr.MyNearTerritory,
		MyFarTerritory:   vr.MyFarTerritory,
		OppNearTerritory: vr.OppNearTerritory,
		OppFarTerritory:  vr.OppFarTerritory,
		MyCorridorCells:  vr.MyCorridorCells,
		OppCorridorCells: vr.OppCorridorCells,
		MyEscapeRoutes:   myEscape,
		OppEscapeRoutes:  oppEscape,
		MyFunnelRatio:    myFunnelRatio,
		MyCorridorRatio:  myCorridorRatio,
		MyEscapeRatio:    myEscapeRatio,
		OppFunnelRatio:   oppFunnelRatio,
		OppCorridorRatio: oppCorridorRatio,
		OppEscapeRatio:   oppEscapeRatio,
	}
	tg.records = append(tg.records, rec)
}

func traceEnd(gameID, snakeID string, state GameState) {
	if !traceEnabled {
		return
	}
	key := traceKey(gameID, snakeID)
	v, ok := traceGames.LoadAndDelete(key)
	if !ok {
		return
	}
	tg := v.(*traceGame)

	// Determine result.
	result := "loss"
	deathCause := ""
	youAlive := false
	for _, s := range state.Board.Snakes {
		if s.ID == snakeID && s.Health > 0 {
			youAlive = true
			break
		}
	}
	if youAlive {
		// Check if all opponents are dead.
		allOppsDead := true
		for _, s := range state.Board.Snakes {
			if s.ID != snakeID && s.Health > 0 {
				allOppsDead = false
				break
			}
		}
		if allOppsDead {
			result = "win"
		} else {
			result = "draw"
		}
	} else {
		result = "loss"
		deathCause = inferDeathCause(state.You, state)
	}

	tg.records = append(tg.records, traceRecord{
		Type:       "footer",
		Result:     result,
		DeathCause: deathCause,
		TotalTurns: state.Turn,
	})

	traceFlush(tg)
}

// inferDeathCause guesses the elimination reason from the end state.
func inferDeathCause(you Battlesnake, state GameState) string {
	if you.Health <= 0 {
		return "starvation"
	}
	head := you.Head
	if head.X < 0 || head.Y < 0 || head.X >= state.Board.Width || head.Y >= state.Board.Height {
		return "wall-collision"
	}
	// Check head-to-head with surviving snakes.
	for _, s := range state.Board.Snakes {
		if s.ID == you.ID {
			continue
		}
		if s.Head == head {
			return "head-collision"
		}
		// Check body collision with surviving snake.
		for j := 1; j < len(s.Body); j++ {
			if s.Body[j] == Coord(head) {
				return "body-collision"
			}
		}
	}
	return "collision"
}

func traceFlush(tg *traceGame) {
	if err := os.MkdirAll("traces", 0o755); err != nil {
		log.Printf("TRACE: failed to create traces dir: %s", err)
		return
	}

	prefix := tg.snakeID
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	fname := filepath.Join("traces", tg.gameID+"_"+prefix+".jsonl")

	f, err := os.Create(fname)
	if err != nil {
		log.Printf("TRACE: failed to create %s: %s", fname, err)
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, rec := range tg.records {
		if err := enc.Encode(rec); err != nil {
			log.Printf("TRACE: failed to write record: %s", err)
			return
		}
	}
	log.Printf("TRACE: wrote %d records to %s", len(tg.records), fname)
}
