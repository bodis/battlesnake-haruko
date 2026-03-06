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
	FoodReach       float64 `json:"food_reach,omitempty"`
	FoodDenial      float64 `json:"food_denial,omitempty"`
	StarvationRisk  float64 `json:"starvation_risk,omitempty"`
	GrowthUrgency   float64 `json:"growth_urgency,omitempty"`
	TailChase       float64 `json:"tail_chase,omitempty"`
	Bottleneck      float64 `json:"bottleneck,omitempty"`
	MyThreatened    int     `json:"my_threatened,omitempty"`
	OppThreatened   int     `json:"opp_threatened,omitempty"`
	EarlyBlend      float64 `json:"early_blend,omitempty"`
	LateBlend       float64 `json:"late_blend,omitempty"`

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
	vr := logic.VoronoiTerritory(sim, myIdx)

	oppLen := 0
	for i := range sim.Snakes {
		if i != myIdx && sim.Snakes[i].IsAlive() {
			oppLen = sim.Snakes[i].Length
			break
		}
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
		FoodReach:       bd.FoodReach,
		FoodDenial:      bd.FoodDenial,
		StarvationRisk:  bd.StarvationRisk,
		GrowthUrgency:   bd.GrowthUrgency,
		TailChase:       bd.TailChase,
		Bottleneck:      bd.Bottleneck,
		MyThreatened:    vr.MyThreatenedTerritory,
		OppThreatened:   vr.OppThreatenedTerritory,
		EarlyBlend:      bd.EarlyBlend,
		LateBlend:       bd.LateBlend,
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
