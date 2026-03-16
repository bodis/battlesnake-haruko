# Iter 36 — Monte Carlo Strategic Rollout (Root-Level, Diagnostic)

## Context

Read `ROADMAP.md` (Iter 36 section), `CLAUDE.md`, `ENGINE.md`, and the memory files in `.claude/projects/-Users-bodist-work-github-battlesnake-haruko/memory/MEMORY.md` for full project context.

**IMPORTANT:** Iter 35 (diagnostic signal instrumentation) should be completed before this iteration. Check if it's been committed. If not, this iteration can still proceed independently — they don't share code.

This is **Phase A** — diagnostic only. We are NOT changing BRS or eval. We are running Monte Carlo rollouts alongside the trace system to see if cheap random simulations can detect long-term danger that BRS can't see.

## What to implement

### 1. `logic/rollout.go` — New file

Core function:

```go
type RolloutStats struct {
    SurvivalRate float64 // 0.0-1.0
    AvgTurns     float64 // average turns survived
    Rollouts     int     // number of rollouts run for this direction pair
}

type DirectionPairStats struct {
    Dir1  Direction
    Dir2  Direction
    Stats RolloutStats
}

// MCRollout runs Monte Carlo random rollouts from the current position.
// It generates all valid 2-ply direction pairs for myIdx, distributes
// totalRollouts evenly across them, and returns per-direction survival stats.
// Returns: map from each first-move direction to aggregated RolloutStats.
func MCRollout(g *GameSim, myIdx, oppIdx int, totalRollouts, maxTurns int) map[Direction]RolloutStats
```

**Step-by-step logic:**

1. **Generate direction pairs:**
   - Get valid moves for our snake from current position (exclude wall collisions using existing `wallSafeMoves` or equivalent logic)
   - For each valid dir1: clone game, apply dir1 for us + random valid move for opponent, step the game
   - From the resulting position, get valid moves again
   - For each valid dir2: record `(dir1, dir2)` as a valid pair
   - Typically produces 6-8 pairs

2. **Distribute rollouts evenly:**
   - `rolloutsPerPair = totalRollouts / len(pairs)`
   - Remainder distributed round-robin

3. **Run rollouts for each pair:**
   - Clone game from root, apply dir1+random_opp, step, apply dir2+random_opp, step
   - Then loop up to maxTurns: both sides play `randomValidMove`, step
   - Track: did we survive? How many turns?
   - Use `CloneFromPool`/`Release` for all clones

4. **Aggregate to per-direction stats:**
   - For each direction (Up/Down/Left/Right), average the stats across all pairs starting with that direction
   - Return `map[Direction]RolloutStats`

**`randomValidMove` function:**

```go
func randomValidMove(g *GameSim, snakeIdx int, rng *uint64) Direction
```

- Get snake head position
- Enumerate 4 directions, filter out wall collisions (head + dir is out of bounds)
- Pick uniformly at random from remaining valid moves
- Use xorshift64 PRNG (same as in `logic/mcts.go` — reuse `xorshift64` function)
- If no valid moves (shouldn't happen normally), return Down as fallback

### 2. Trace integration in `trace.go`

Add fields to `traceRecord`:

```go
// Diagnostic: Monte Carlo rollout survival rates per direction
MCUp    float64 `json:"mc_up,omitempty"`
MCDown  float64 `json:"mc_down,omitempty"`
MCLeft  float64 `json:"mc_left,omitempty"`
MCRight float64 `json:"mc_right,omitempty"`
MCBestDir  string `json:"mc_best_dir,omitempty"`   // direction with highest survival
MCWorstDir string `json:"mc_worst_dir,omitempty"`  // direction with lowest survival
MCSpread   float64 `json:"mc_spread,omitempty"`    // best - worst survival rate
```

In `traceTurn()`, after existing diagnostics:

```go
if oppIdx >= 0 {
    mcStats := logic.MCRollout(sim, myIdx, oppIdx, 1000, 200)
    // Fill trace fields from mcStats
}
```

This runs outside the game budget (trace is diagnostic-only, not time-constrained).

### 3. New analyze mode: `rollout` in `cmd/analyze/main.go`

Add the MC fields to the `record` struct.

New mode `rollout` that reports:

**Section 1: MC-BRS Agreement**
- For each turn: compare BRS chosen direction vs MC best direction
- Report: % of turns where they agree
- Breakdown by game phase (early/mid/late using `late_blend`)

**Section 2: MC Disagreement Analysis**
- When MC disagrees with BRS:
  - What % of those games did we eventually win vs lose?
  - Compare to baseline win rate
  - Was MC's preferred direction "better" in hindsight? (harder to measure — use eventual outcome as proxy)

**Section 3: MC as Death Predictor**
- For games ending in loss: at each turn, was MC's worst direction the direction BRS chose?
- Report: % of deaths preceded by "BRS chose MC's worst direction" within N turns of death (N = 5, 10, 20)
- MC spread (best - worst survival rate) as danger indicator: high spread = MC sees directional danger

**Section 4: MC Survival Rate Distribution**
- For wins: avg MC best/worst survival rate, avg spread
- For losses: same
- Does MC spread correlate with game outcome?

**Section 5: Per-Direction Summary**
- Average MC survival rate per direction across all games/turns
- Any systematic direction biases?

## Important design decisions

- **1000 rollouts, 200 max turns** — these are the default parameters. Can adjust after seeing data.
- **Random valid move = wall-only filtering** — same as BRS pruning (wall-only is sound, body pruning is not). Do NOT filter body collisions — random play should include body collisions since they're part of the probability space.
- **Use full `GameSim.Step()`** — accurate rules, 49ns/step. No simplified sim. Keeps it simple and correct.
- **Opponent moves are also random** in the rollout (after the initial 2 setup plies). Both sides play uniformly random from non-wall moves.

## What NOT to do

- Do NOT change `Evaluate()` or any eval weights
- Do NOT change `BestMoveIterative()` or any search logic
- Do NOT wire MC results into BRS move selection
- All new code only runs when `traceEnabled == true`
- Keep the hot path completely untouched
- Do NOT overcomplicate the random move selection — uniform random from non-wall moves is sufficient

## Testing

1. `go build ./...` — must compile
2. `go test ./logic/...` — existing tests must pass
3. Add tests for `MCRollout`:
   - Test that all valid directions get rollouts
   - Test that survival rates are between 0.0 and 1.0
   - Test on a simple position where one direction is obviously bad (e.g., toward a wall corner)
4. Add test for `randomValidMove`: verify it never returns a wall move
5. Benchmark `MCRollout` with 1000 rollouts / 200 turns — should be ~15ms
6. `make trace N=20` — run with tracing
7. `make analyze MODE=rollout` — verify analysis output

## After implementation

Do NOT commit yet. Show me the `make analyze MODE=rollout` output so we can review the data together. Key questions to answer:
1. Does MC spread (best - worst survival) correlate with game phase? (expect higher in late game)
2. How often does MC disagree with BRS?
3. When MC disagrees with BRS, is there a pattern in game outcomes?
