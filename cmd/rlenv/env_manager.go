package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	pb "github.com/bodist/haruko/cmd/rlenv/envpb"
	"github.com/bodist/haruko/logic"
)

// opponentType enumerates opponent policies.
type opponentType int

const (
	oppNone   opponentType = 0
	oppRandom opponentType = 1
	oppBRS    opponentType = 2
)

// singleEnv holds the state for one parallel environment.
type singleEnv struct {
	mu       sync.Mutex
	game     *logic.GameSim
	prevGame *logic.GameSim // snapshot before last step (for reward computation)
	rng      *rand.Rand
	myIdx    int
	oppIdx   int // -1 if no opponent
	turn     int
	done     bool
}

// EnvManager manages N parallel game environments.
type EnvManager struct {
	envs        []*singleEnv
	numEnvs     int
	boardWidth  int
	boardHeight int
	oppType     opponentType
	oppPolicy   OpponentPolicy
	maxTurns    int
	brsBudget   time.Duration
}

// NewEnvManager creates and configures a manager with the given parameters.
func NewEnvManager(cfg *pb.ConfigRequest) *EnvManager {
	numEnvs := int(cfg.NumEnvs)
	if numEnvs <= 0 {
		numEnvs = 128
	}
	w := int(cfg.BoardWidth)
	if w <= 0 {
		w = 11
	}
	h := int(cfg.BoardHeight)
	if h <= 0 {
		h = 11
	}
	maxTurns := int(cfg.MaxTurns)
	if maxTurns <= 0 {
		maxTurns = 500
	}
	brsBudget := time.Duration(cfg.BrsBudgetMs) * time.Millisecond
	if brsBudget <= 0 {
		brsBudget = 50 * time.Millisecond
	}

	var ot opponentType
	var policy OpponentPolicy
	switch cfg.OpponentType {
	case "random":
		ot = oppRandom
		policy = &RandomPolicy{}
	case "brs":
		ot = oppBRS
		policy = &BRSPolicy{Budget: brsBudget}
	default:
		ot = oppNone
		policy = nil
	}

	mgr := &EnvManager{
		numEnvs:     numEnvs,
		boardWidth:  w,
		boardHeight: h,
		oppType:     ot,
		oppPolicy:   policy,
		maxTurns:    maxTurns,
		brsBudget:   brsBudget,
		envs:        make([]*singleEnv, numEnvs),
	}

	for i := 0; i < numEnvs; i++ {
		mgr.envs[i] = &singleEnv{
			rng: rand.New(rand.NewSource(time.Now().UnixNano() + int64(i))),
		}
	}

	return mgr
}

// hasOpponent returns true if this manager was configured with an opponent.
func (m *EnvManager) hasOpponent() bool {
	return m.oppType != oppNone
}

// resetEnv resets a single environment to a random initial board state.
func (m *EnvManager) resetEnv(e *singleEnv) {
	w, h := m.boardWidth, m.boardHeight

	var snakes []logic.SimSnake

	// Place our snake at a random position with 3-length body.
	mySnake := randomSnake(w, h, e.rng, "agent", nil)
	snakes = append(snakes, mySnake)
	e.myIdx = 0

	if m.hasOpponent() {
		// Place opponent, avoiding our snake's cells.
		occupied := bodySet(mySnake.Body)
		oppSnake := randomSnake(w, h, e.rng, "opponent", occupied)
		snakes = append(snakes, oppSnake)
		e.oppIdx = 1
	} else {
		e.oppIdx = -1
	}

	// Collect all occupied cells.
	occupied := make(map[logic.Coord]bool)
	for _, s := range snakes {
		for _, c := range s.Body {
			occupied[c] = true
		}
	}

	// Spawn 3 food at random empty cells.
	food := spawnFoodInitial(w, h, 3, e.rng, occupied)

	e.game = logic.NewGameSim(w, h, snakes, food, nil)
	e.prevGame = nil
	e.turn = 0
	e.done = false
}

// randomSnake creates a snake at a random position with a 3-length body
// pointing in a random direction. avoid is a set of cells to not place the head on.
func randomSnake(w, h int, rng *rand.Rand, id string, avoid map[logic.Coord]bool) logic.SimSnake {
	dirs := []logic.Direction{logic.Up, logic.Down, logic.Left, logic.Right}

	for attempts := 0; attempts < 1000; attempts++ {
		headX := 1 + rng.Intn(w-2) // avoid edges for body room
		headY := 1 + rng.Intn(h-2)
		head := logic.Coord{X: headX, Y: headY}

		if avoid != nil && avoid[head] {
			continue
		}

		// Pick a random direction for the body to extend (opposite of facing).
		bodyDir := dirs[rng.Intn(4)]
		seg1 := head.Move(bodyDir)
		seg2 := seg1.Move(bodyDir)

		// Check all segments are in bounds and not in avoid set.
		if seg1.X < 0 || seg1.X >= w || seg1.Y < 0 || seg1.Y >= h {
			continue
		}
		if seg2.X < 0 || seg2.X >= w || seg2.Y < 0 || seg2.Y >= h {
			continue
		}
		if avoid != nil && (avoid[seg1] || avoid[seg2]) {
			continue
		}

		return logic.SimSnake{
			ID:     id,
			Body:   []logic.Coord{head, seg1, seg2},
			Health: 100,
			Length: 3,
		}
	}

	// Fallback: place at center.
	head := logic.Coord{X: w / 2, Y: h / 2}
	seg1 := logic.Coord{X: w / 2, Y: h/2 - 1}
	seg2 := logic.Coord{X: w / 2, Y: h/2 - 2}
	return logic.SimSnake{
		ID:     id,
		Body:   []logic.Coord{head, seg1, seg2},
		Health: 100,
		Length: 3,
	}
}

// bodySet returns a set of coordinates from a body slice.
func bodySet(body []logic.Coord) map[logic.Coord]bool {
	s := make(map[logic.Coord]bool, len(body))
	for _, c := range body {
		s[c] = true
	}
	return s
}

// spawnFoodInitial places n food items at random empty cells.
func spawnFoodInitial(w, h, n int, rng *rand.Rand, occupied map[logic.Coord]bool) []logic.Coord {
	var food []logic.Coord
	for attempts := 0; len(food) < n && attempts < 1000; attempts++ {
		c := logic.Coord{X: rng.Intn(w), Y: rng.Intn(h)}
		if occupied[c] {
			continue
		}
		food = append(food, c)
		occupied[c] = true
	}
	return food
}

// Configure creates a new set of environments with the given parameters.
func (s *EnvServer) Configure(ctx context.Context, req *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	// Clean up previous shm if any.
	if s.shm != nil {
		s.shm.closeSHM()
		s.shm = nil
	}

	s.mgr = NewEnvManager(req)

	// Create shared memory.
	var shmPath string
	if s.shmPath != "" {
		shm, err := createSHM(s.shmPath, s.mgr.numEnvs)
		if err != nil {
			return nil, fmt.Errorf("shm init: %w", err)
		}
		s.shm = shm
		shmPath = s.shmPath
	}

	return &pb.ConfigResponse{
		NumEnvs:         int32(s.mgr.numEnvs),
		SpatialChannels: spatialChannels,
		SpatialSize:     spatialSize,
		NumScalars:      numScalars,
		NumActions:      numActions,
		ShmPath:         shmPath,
	}, nil
}

// Reset resets the specified environments (or all if empty) and returns observations.
func (s *EnvServer) Reset(ctx context.Context, req *pb.ResetRequest) (*pb.Observations, error) {
	if s.mgr == nil {
		return nil, fmt.Errorf("not configured; call Configure first")
	}

	envIDs := req.EnvIds
	if len(envIDs) == 0 {
		envIDs = make([]int32, s.mgr.numEnvs)
		for i := range envIDs {
			envIDs[i] = int32(i)
		}
	}

	// Reset environments in parallel.
	var wg sync.WaitGroup
	for _, id := range envIDs {
		idx := int(id)
		if idx < 0 || idx >= s.mgr.numEnvs {
			continue
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			e := s.mgr.envs[i]
			e.mu.Lock()
			defer e.mu.Unlock()
			s.mgr.resetEnv(e)
		}(idx)
	}
	wg.Wait()

	// Also write to shm if available.
	if s.shm != nil {
		s.writeObsToSHM(envIDs)
	}

	return s.collectObservations(envIDs), nil
}

// Step applies actions to the specified environments and returns results.
func (s *EnvServer) Step(ctx context.Context, req *pb.StepRequest) (*pb.StepResponse, error) {
	if s.mgr == nil {
		return nil, fmt.Errorf("not configured; call Configure first")
	}

	n := len(req.EnvIds)
	if n != len(req.Actions) {
		return nil, fmt.Errorf("env_ids length (%d) != actions length (%d)", n, len(req.Actions))
	}

	rewards := make([]float32, n)
	dones := make([]bool, n)
	infos := make([]string, n)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			envID := int(req.EnvIds[idx])
			if envID < 0 || envID >= s.mgr.numEnvs {
				return
			}
			action := logic.Direction(req.Actions[idx])

			e := s.mgr.envs[envID]
			e.mu.Lock()
			defer e.mu.Unlock()

			// Save previous state for reward computation.
			if e.prevGame != nil {
				// Reuse: just overwrite.
			}
			e.prevGame = e.game.Clone()

			// Correct invalid actions: if the chosen action is a 180-degree
			// reversal or hits a wall/body, pick a safe alternative.
			action = correctAction(e.game, e.myIdx, action)

			// Build MoveSet: agent action + opponent action.
			var ms logic.MoveSet
			ms.Dir[e.myIdx] = action
			ms.Has[e.myIdx] = true

			if e.oppIdx >= 0 && e.game.Snakes[e.oppIdx].IsAlive() && s.mgr.oppPolicy != nil {
				oppDir := s.mgr.oppPolicy.ChooseMove(e.game, e.oppIdx)
				ms.Dir[e.oppIdx] = oppDir
				ms.Has[e.oppIdx] = true
			}

			e.game.Step(ms)
			e.turn++

			// Maybe spawn food.
			maybeSpawnFood(e.game, e.rng)

			// Compute reward.
			reward := computeReward(e.game, e.prevGame, e.myIdx, e.oppIdx, s.mgr.hasOpponent())
			rewards[idx] = reward

			// Check if done.
			agentDead := !e.game.Snakes[e.myIdx].IsAlive()
			maxTurnsReached := e.turn >= s.mgr.maxTurns
			// In solo mode (no opponent), IsOver() is always true (< 2 alive).
			// Only use it when there IS an opponent.
			oppDead := false
			if e.oppIdx >= 0 && !e.game.Snakes[e.oppIdx].IsAlive() {
				oppDead = true
			}
			done := agentDead || maxTurnsReached || oppDead
			dones[idx] = done

			// Build info.
			info := map[string]interface{}{
				"turn":        e.turn,
				"agent_alive": e.game.Snakes[e.myIdx].IsAlive(),
			}
			if e.oppIdx >= 0 {
				info["opp_alive"] = e.game.Snakes[e.oppIdx].IsAlive()
			}
			if done {
				info["terminal_reason"] = terminalReason(agentDead, maxTurnsReached, oppDead)
			}
			infoJSON, _ := json.Marshal(info)
			infos[idx] = string(infoJSON)

			// Auto-reset if done.
			if done {
				e.done = true
				s.mgr.resetEnv(e)
			}
		}(i)
	}
	wg.Wait()

	obs := s.collectObservations(req.EnvIds)
	return &pb.StepResponse{
		Observations: obs,
		Rewards:      rewards,
		Dones:        dones,
		Infos:        infos,
	}, nil
}

// collectObservations gathers observations for the given env IDs.
func (s *EnvServer) collectObservations(envIDs []int32) *pb.Observations {
	n := len(envIDs)
	spatialLen := spatialChannels * spatialSize * spatialSize
	allSpatial := make([]float32, 0, n*spatialLen)
	allScalars := make([]float32, 0, n*numScalars)
	allMask := make([]bool, 0, n*numActions)

	for _, id := range envIDs {
		idx := int(id)
		if idx < 0 || idx >= s.mgr.numEnvs {
			// Pad with zeros for invalid IDs.
			allSpatial = append(allSpatial, make([]float32, spatialLen)...)
			allScalars = append(allScalars, make([]float32, numScalars)...)
			allMask = append(allMask, make([]bool, numActions)...)
			continue
		}
		e := s.mgr.envs[idx]
		e.mu.Lock()
		spatial, scalars := computeObservation(e.game, e.myIdx)
		mask := computeActionMask(e.game, e.myIdx)
		e.mu.Unlock()

		allSpatial = append(allSpatial, spatial...)
		allScalars = append(allScalars, scalars...)
		for _, m := range mask {
			allMask = append(allMask, m)
		}
	}

	return &pb.Observations{
		Spatial:    allSpatial,
		Scalars:    allScalars,
		ActionMask: allMask,
		EnvIds:     envIDs,
	}
}

// writeObsToSHM writes observations for the given envs into shared memory.
func (s *EnvServer) writeObsToSHM(envIDs []int32) {
	for _, id := range envIDs {
		idx := int(id)
		if idx < 0 || idx >= s.mgr.numEnvs {
			continue
		}
		e := s.mgr.envs[idx]
		e.mu.Lock()
		spatial, scalars := computeObservation(e.game, e.myIdx)
		mask := computeActionMask(e.game, e.myIdx)
		e.mu.Unlock()

		s.shm.writeSpatial(idx, spatial)
		s.shm.writeScalars(idx, scalars)
		s.shm.writeMask(idx, mask)
	}
}

// StepSignal performs a step using shared memory for data transfer.
func (s *EnvServer) StepSignal(ctx context.Context, req *pb.StepSignalRequest) (*pb.StepSignalResponse, error) {
	if s.mgr == nil {
		return nil, fmt.Errorf("not configured; call Configure first")
	}
	if s.shm == nil {
		return nil, fmt.Errorf("shared memory not initialized")
	}

	n := s.mgr.numEnvs
	envIDs := make([]int32, n)
	for i := range envIDs {
		envIDs[i] = int32(i)
	}

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			action := logic.Direction(s.shm.readAction(idx))

			e := s.mgr.envs[idx]
			e.mu.Lock()
			defer e.mu.Unlock()

			// Save previous state for reward computation.
			e.prevGame = e.game.Clone()

			action = correctAction(e.game, e.myIdx, action)

			var ms logic.MoveSet
			ms.Dir[e.myIdx] = action
			ms.Has[e.myIdx] = true

			if e.oppIdx >= 0 && e.game.Snakes[e.oppIdx].IsAlive() && s.mgr.oppPolicy != nil {
				oppDir := s.mgr.oppPolicy.ChooseMove(e.game, e.oppIdx)
				ms.Dir[e.oppIdx] = oppDir
				ms.Has[e.oppIdx] = true
			}

			e.game.Step(ms)
			e.turn++

			maybeSpawnFood(e.game, e.rng)

			reward := computeReward(e.game, e.prevGame, e.myIdx, e.oppIdx, s.mgr.hasOpponent())
			s.shm.writeReward(idx, reward)

			agentDead := !e.game.Snakes[e.myIdx].IsAlive()
			maxTurnsReached := e.turn >= s.mgr.maxTurns
			oppDead := false
			if e.oppIdx >= 0 && !e.game.Snakes[e.oppIdx].IsAlive() {
				oppDead = true
			}
			done := agentDead || maxTurnsReached || oppDead
			s.shm.writeDone(idx, done)

			if done {
				e.done = true
				s.mgr.resetEnv(e)
			}
		}(i)
	}
	wg.Wait()

	// Write observations (post-reset for done envs).
	s.writeObsToSHM(envIDs)

	return &pb.StepSignalResponse{StepId: req.StepId}, nil
}

// terminalReason returns a human-readable reason for episode termination.
func terminalReason(agentDead, maxTurns, oppDead bool) string {
	if agentDead {
		return "agent_dead"
	}
	if maxTurns {
		return "max_turns"
	}
	if oppDead {
		return "opp_dead"
	}
	return "unknown"
}
