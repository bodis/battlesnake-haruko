package main

import (
	"log"
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
	log.Printf("GAME START %s\n", state.Game.ID)
	traceStart(state.Game.ID, state.You.ID, state.Board.Width, state.Board.Height)
}

func end(state GameState) {
	log.Printf("GAME OVER  %s\n", state.Game.ID)
	traceEnd(state.Game.ID, state.You.ID, state)
}

func move(state GameState) BattlesnakeMoveResponse {
	sim := gameSimFromState(state)
	dir := sim.BestMoveIterative(state.You.ID, 300*time.Millisecond)
	m := logic.DirectionName(dir)
	log.Printf("MOVE %d: %s (health=%d)\n", state.Turn, m, state.You.Health)
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


func main() {
	RunServer()
}
