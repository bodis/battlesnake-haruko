package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/bodist/haruko/logic"
)

func info() BattlesnakeInfoResponse {
	log.Println("INFO")
	return BattlesnakeInfoResponse{
		APIVersion: "1",
		Author:     "",
		Color:      "#C70039",
		Head:       "default",
		Tail:       "default",
	}
}

func start(state GameState) {
	log.Printf("GAME START %s (timeout=%dms)\n", state.Game.ID, state.Game.Timeout)
	initBudgetTracker(state.Game.ID, state.Game.Timeout)
	traceStart(state.Game.ID, state.You.ID, state.Board.Width, state.Board.Height)
}

func end(state GameState) {
	log.Printf("GAME OVER  %s\n", state.Game.ID)
	removeBudgetTracker(state.Game.ID)
	traceEnd(state.Game.ID, state.You.ID, state)
}

const brsBudget = 300 * time.Millisecond // fixed BRS budget, proven depth

func move(state GameState) BattlesnakeMoveResponse {
	computeStart := time.Now()

	totalBudget := computeBudget(state.Game.ID, state.You.Latency)
	mcBudget := totalBudget - brsBudget // adaptive headroom goes to MC rollouts
	if mcBudget < 20*time.Millisecond {
		mcBudget = 20 * time.Millisecond // minimum MC budget
	}

	sim := gameSimFromState(state)
	dir := sim.BestMoveIterative(state.You.ID, brsBudget, mcBudget)
	m := logic.DirectionName(dir)

	computeTime := time.Since(computeStart)
	recordComputeTime(state.Game.ID, computeTime)

	mcRollouts := 0
	if sim.LastMCResult != nil {
		for d := 0; d < 4; d++ {
			mcRollouts += sim.LastMCResult.Stats[d].Total
		}
	}

	log.Printf("MOVE %d: %s (health=%d brs=%dms mc=%dms compute=%dms depth=%d rollouts=%d)\n",
		state.Turn, m, state.You.Health,
		brsBudget.Milliseconds(), mcBudget.Milliseconds(),
		computeTime.Milliseconds(),
		sim.LastCompletedDepth, mcRollouts)
	traceTurn(state.Game.ID, state.You.ID, state, sim, m)
	return BattlesnakeMoveResponse{Move: m}
}

// coordsToLogic converts API Coord slice to logic.Coord slice.
func coordsToLogic(in []Coord) []logic.Coord {
	out := make([]logic.Coord, len(in))
	for i, c := range in {
		out[i] = logic.Coord{X: c.X, Y: c.Y}
	}
	return out
}

// gameSimFromState converts the API GameState into a logic.GameSim for
// simulation / minimax search. Builds the struct directly to avoid a
// redundant deep-copy through NewGameSim (caller already owns fresh slices).
func gameSimFromState(state GameState) *logic.GameSim {
	snakes := make([]logic.SimSnake, len(state.Board.Snakes))
	for i, s := range state.Board.Snakes {
		body := make([]logic.Coord, len(s.Body))
		for j, seg := range s.Body {
			body[j] = logic.Coord{X: seg.X, Y: seg.Y}
		}
		snakes[i] = logic.SimSnake{
			ID:     s.ID,
			Body:   body,
			Health: s.Health,
			Length: s.Length,
		}
	}
	return &logic.GameSim{
		Width:   state.Board.Width,
		Height:  state.Board.Height,
		Snakes:  snakes,
		Food:    coordsToLogic(state.Board.Food),
		Hazards: coordsToLogic(state.Board.Hazards),
		Turn:    state.Turn,
	}
}


func init() {
	// MC rollout tuning via environment variables.
	if v := os.Getenv("MC_WEIGHT"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			logic.MCWeight = f
			log.Printf("MC_WEIGHT=%.1f", f)
		}
	}
	if v := os.Getenv("MC_SPREAD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			logic.MCSpreadThreshold = f
			log.Printf("MC_SPREAD=%.2f", f)
		}
	}
	if v := os.Getenv("MC_GATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			logic.MCLateGate = f
			log.Printf("MC_GATE=%.2f", f)
		}
	}
	if v := os.Getenv("MC_TURNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			logic.RolloutMaxTurns = n
			log.Printf("MC_TURNS=%d", n)
		}
	}
}

func main() {
	RunServer()
}
