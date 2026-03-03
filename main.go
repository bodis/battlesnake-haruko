package main

import (
	"log"
	"math/rand"
	"sync"

	"github.com/bodist/haruko/logic"
)

// coordToLogicSingle converts a single API Coord to logic.Coord.
func coordToLogicSingle(c Coord) logic.Coord {
	return logic.Coord{X: c.X, Y: c.Y}
}

// GameSession holds per-game state across turns for a single game.
type GameSession struct {
	Board *logic.FastBoard
}

var (
	sessions   = make(map[string]*GameSession)
	sessionsMu sync.RWMutex
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
	session := &GameSession{
		Board: logic.NewFastBoard(state.Board.Width, state.Board.Height),
	}
	sessionsMu.Lock()
	sessions[state.Game.ID] = session
	sessionsMu.Unlock()
}

func end(state GameState) {
	log.Printf("GAME OVER  %s\n", state.Game.ID)
	sessionsMu.Lock()
	delete(sessions, state.Game.ID)
	sessionsMu.Unlock()
}

func move(state GameState) BattlesnakeMoveResponse {
	sessionsMu.RLock()
	session, ok := sessions[state.Game.ID]
	sessionsMu.RUnlock()
	if !ok {
		// Fallback: create session if /start was missed
		session = &GameSession{
			Board: logic.NewFastBoard(state.Board.Width, state.Board.Height),
		}
		sessionsMu.Lock()
		sessions[state.Game.ID] = session
		sessionsMu.Unlock()
	}

	// Sync FastBoard from current game state
	logicFood := coordsToLogic(state.Board.Food)
	session.Board.Update(logicFood, coordsToLogic(state.Board.Hazards), snakesToLogic(state.Board.Snakes), state.You.ID)

	isMoveSafe := map[string]bool{
		"up":    true,
		"down":  true,
		"left":  true,
		"right": true,
	}

	myHead := state.You.Head
	myNeck := state.You.Body[1]

	// Don't reverse into neck
	if myNeck.X < myHead.X {
		isMoveSafe["left"] = false
	} else if myNeck.X > myHead.X {
		isMoveSafe["right"] = false
	} else if myNeck.Y < myHead.Y {
		isMoveSafe["down"] = false
	} else if myNeck.Y > myHead.Y {
		isMoveSafe["up"] = false
	}

	// Avoid walls and occupied cells using FastBoard
	width := state.Board.Width
	height := state.Board.Height
	dirs := map[string]Coord{
		"up":    {myHead.X, myHead.Y + 1},
		"down":  {myHead.X, myHead.Y - 1},
		"left":  {myHead.X - 1, myHead.Y},
		"right": {myHead.X + 1, myHead.Y},
	}
	for dir, next := range dirs {
		if next.X < 0 || next.X >= width || next.Y < 0 || next.Y >= height {
			isMoveSafe[dir] = false
			continue
		}
		if session.Board.IsBlocked(next.X, next.Y) {
			isMoveSafe[dir] = false
		}
	}

	safeMoves := []string{}
	for dir, safe := range isMoveSafe {
		if safe {
			safeMoves = append(safeMoves, dir)
		}
	}

	if len(safeMoves) == 0 {
		log.Printf("MOVE %d: No safe moves detected! Moving down\n", state.Turn)
		return BattlesnakeMoveResponse{Move: "down"}
	}

	// Score each safe move by flood-fill reachable space + food bonus
	health := state.You.Health
	bestMove := safeMoves[0]
	bestScore := -1.0
	tiedMoves := []string{}
	for _, dir := range safeMoves {
		next := dirs[dir]
		spaceScore := session.Board.FloodFill(coordToLogicSingle(next))
		score := float64(spaceScore)

		dist := logic.NearestFoodDistance(coordToLogicSingle(next), logicFood)
		if dist >= 0 {
			urgency := float64(100-health) * 0.15
			score += urgency / float64(max(dist, 1))
		}

		if score > bestScore {
			bestScore = score
			bestMove = dir
			tiedMoves = []string{dir}
		} else if score == bestScore {
			tiedMoves = append(tiedMoves, dir)
		}
	}
	// Break ties randomly
	if len(tiedMoves) > 1 {
		bestMove = tiedMoves[rand.Intn(len(tiedMoves))]
	}

	log.Printf("MOVE %d: %s (score=%.1f, health=%d)\n", state.Turn, bestMove, bestScore, health)
	return BattlesnakeMoveResponse{Move: bestMove}
}

// coordsToLogic converts API Coord slice to logic.Coord slice.
func coordsToLogic(in []Coord) []logic.Coord {
	out := make([]logic.Coord, len(in))
	for i, c := range in {
		out[i] = logic.Coord{X: c.X, Y: c.Y}
	}
	return out
}

// snakesToLogic converts API Battlesnake slice to logic.Snake slice.
func snakesToLogic(in []Battlesnake) []logic.Snake {
	out := make([]logic.Snake, len(in))
	for i, s := range in {
		body := make([]logic.Coord, len(s.Body))
		for j, seg := range s.Body {
			body[j] = logic.Coord{X: seg.X, Y: seg.Y}
		}
		out[i] = logic.Snake{ID: s.ID, Body: body}
	}
	return out
}

func main() {
	RunServer()
}
