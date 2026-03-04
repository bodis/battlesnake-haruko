# Haruko Battlesnake — Development Roadmap

> Living document tracking iterative improvements from random safe-move to competitive Minimax AI.
> Each iteration is a self-contained unit: implement → test → snapshot → compare → merge.

---

## How to Use This Document

1. Start each session by reading the **current iteration** section
2. Implement, run `go test ./...`, then `make bench N=100`
3. `make snapshot` after implementation
4. `make compare PREV=<previous-snapshot> N=100` to measure improvement
5. Update the **Results** row in the iteration table and mark as done

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iteration 5 |
| **Next** | Iteration 6 |
| **Baseline** | v0 random safe-move: ~68 avg turns (self-play) |
| **Current** | v5 1-ply paranoid minimax: ~87 avg turns (self-play), 16% vs v2 flood-fill |

---

## Phase 1: Survival Foundation

### Iteration 1 — Flood Fill + Space-Aware Moves ✅

**Status:** DONE
**Snapshot:** (create with `make snapshot` before starting Iter 2)

**Goal:** Stop trapping ourselves. Biggest bang-for-buck safety improvement.

**What was built:**
- `InBounds(x, y)` on FastBoard
- `CellSnakeTail` constant — tail awareness in `Update()` (non-stacked tails marked passable)
- `Direction` type + constants (`Up/Down/Left/Right`), `DirectionName()`, `Coord.Move(d)` in `logic/flood.go`
- `FloodFill(start)` — BFS counting reachable cells (empty + food + tail are passable)
- `move()` in `main.go` scores each safe move by flood-fill reachable space, picks highest

**Files touched:**
- `logic/board.go` — `InBounds`, `CellSnakeTail`, tail marking in `Update()`
- `logic/flood.go` — **new** — Direction, FloodFill, Coord.Move
- `logic/board_test.go` — **new** — 5 tests
- `logic/flood_test.go` — **new** — 11 tests
- `main.go` — flood-fill scoring replaces random selection

**Results:**
| Metric | Before | After |
|--------|--------|-------|
| Avg turns (self-play) | ~68 | ~78 |
| Max turns | ~140 | 249 |
| Draw rate | ~10% | 26% |
| Tests | 0 | 16 |

> Self-play understates improvement since both snakes use the same code. The doubled draw rate
> and higher max turns confirm significantly better survival.

---

### Iteration 2 — Food-Seeking Heuristic ✅

**Status:** DONE
**Depends on:** Iteration 1
**Snapshot:** `snapshots/haruko-244a28f`

**Goal:** Don't starve. After self-trapping, starvation is the #1 cause of death. This is a quick heuristic win that doesn't need the game simulator — just bias the move scorer toward food.

**Problem:** Currently the snake picks the move with the most flood-fill space. It completely ignores food. On an 11x11 board, health ticks down 1/turn starting at 100. A snake that doesn't eat will die on turn 100 regardless of how much space it has.

**Approach:**
- Among safe moves, calculate Manhattan distance from each next-position to the nearest food
- Combine flood-fill space and food proximity into a composite score
- When health is high (>50), space dominates. When health is low (<30), food proximity dominates
- Simple linear interpolation or threshold-based weighting

**Implementation:**
1. Add `NearestFoodDistance(start Coord, food []Coord) int` helper in `logic/flood.go` (or a new `logic/food.go`)
   - Manhattan distance to closest food, returns large number if no food
2. Update `move()` in `main.go`:
   - For each safe move: `score = spaceScore + foodBonus`
   - `foodBonus` increases as health decreases (e.g., `foodBonus = (100 - health) * k / distance`)
   - Tune `k` so the snake doesn't dive into tiny corridors for food but does seek food when hungry

**Key constraint:** Don't sacrifice space safety for food. If a move leads to a dead-end of 5 cells but has food, prefer the move with 80 open cells. The food bonus must never overpower a large space difference.

**Files:**
| File | Action |
|------|--------|
| `logic/food.go` | **New** — `NearestFoodDistance()` |
| `logic/food_test.go` | **New** — distance tests, no-food edge case |
| `main.go` | Update scoring in `move()` |

**Results:**
| Metric | Before (v1) | After (v2) |
|--------|-------------|------------|
| Avg turns (self-play) | ~78 | ~28 (shorter due to food-chase collisions) |
| vs v1 win rate | — | 95% (100 games) |
| Draw rate (self-play) | 26% | 39% |
| Tests | 16 | 21 |

> Self-play turns dropped because both snakes now aggressively chase the same food, causing more
> head-to-head collisions. The 95% win rate against v1 confirms the food heuristic is a massive
> upgrade — the food-seeking snake outlives the non-eating snake almost every time.

---

## Phase 2: Game Simulator

The game simulator is the foundation for all search-based AI. It must replicate the official Battlesnake rules exactly so that minimax can accurately predict future game states.

### Iteration 3 — Game Simulator Core ✅

**Status:** DONE
**Depends on:** Iteration 2

**Goal:** Build `GameSim` struct that can represent a full game state, clone itself, and apply basic snake movement (before rules resolution).

**What was built:**
- `SimSnake` struct with `Head()` / `Tail()` accessors
- `GameSim` struct with `Width`, `Height`, `Snakes`, `Food`, `Hazards`, `Turn`
- `NewGameSim()` — deep-copies all input slices so GameSim owns its data
- `Clone()` — full deep copy, no shared backing arrays
- `MoveSnakes(moves map[string]Direction)` — in-place head advance + tail drop, zero allocation. Skips dead snakes and snakes not in map
- `SnakeByID(id)` — linear scan, returns pointer for in-place mutation
- `gameSimFromState()` bridge in `main.go` — converts API types to `logic.GameSim` (unused until Iter 5)

**Files:**
| File | Action |
|------|--------|
| `logic/sim.go` | **New** — SimSnake, GameSim, NewGameSim, Clone, MoveSnakes, SnakeByID |
| `logic/sim_test.go` | **New** — 18 tests (init, clone, movement, SnakeByID, accessors) |
| `main.go` | Added `gameSimFromState()` bridge |

**Results:**
| Metric | Value |
|--------|-------|
| New tests | 18 (total: 39) |
| Behavioral change | None (infrastructure only) |

---

### Iteration 4 — Game Simulator Rules ✅

**Status:** DONE
**Depends on:** Iteration 3

**Goal:** Complete the simulator so it matches official Battlesnake rules for Standard mode. After this iteration, `GameSim.Step(moves)` produces the same result as the official engine.

**What was built:**
- `HazardDamage = 14` constant
- `SimSnake.IsAlive()` — checks `EliminatedCause == ""`
- `GameSim.IsOver()` — returns true when fewer than 2 snakes alive
- `GameSim.Step(moves map[string]Direction)` — full turn execution in 7 phases:
  1. **Save tails** (by snake index, not map — zero extra allocation)
  2. **Move snakes** (reuses `MoveSnakes` from Iter 3)
  3. **Reduce health** by 1 for each alive, moved snake
  4. **Hazard damage** — subtract `HazardDamage` (14) if head is on a hazard coord
  5. **Feed snakes** — head on food: health=100, length++, append saved tail. Guarded by `moves` map (unmoved snakes can't eat). Two snakes can eat the same food. Eaten food removed via backwards swap-and-truncate.
  6. **Eliminate snakes** (simultaneous — collect all, apply at end):
     - 5a: Health ≤ 0 → `"starvation"`
     - 5b: Head out of bounds → `"wall"`
     - 5c: Head on any body segment (index > 0, including self) → `"body-collision"`
     - 5d: Head-to-head (2+ heads same cell) → shorter dies `"head-collision"`, equal length all die. Only snakes surviving 5a-5c participate.
  7. **Increment turn**

**Design decisions:**
- Eliminated snakes stay in `Snakes` slice with `EliminatedCause` set — needed for eval in later iterations
- Body collision checks run against grown bodies (feeding before elimination matches official rules)
- Feeding guarded by `moves` map — unmoved snakes can't eat, matching official engine behavior

**Files:**
| File | Action |
|------|--------|
| `logic/sim.go` | Added `HazardDamage`, `IsAlive()`, `Step()`, `IsOver()` |
| `logic/sim_test.go` | 20 new tests covering all rule scenarios |

**Results:**
| Metric | Value |
|--------|-------|
| New tests | 20 (total: 57) |
| Behavioral change | None (infrastructure only) |

---

## Phase 3: Search

### Iteration 5 — 1-Ply Lookahead (Paranoid Minimax)

**Status:** DONE
**Depends on:** Iteration 4

**Goal:** For each of our 4 possible moves, simulate all 4 opponent moves (worst case), and pick our move that maximizes our worst-case outcome. This is depth-1 paranoid minimax.

**Why 1-ply first?** It's simple, fast (4×4 = 16 simulations per turn), and already a huge improvement over pure heuristics. It validates the simulator works correctly under game play before we add deeper search.

**Approach:**
- For each of our moves (up to 4):
  - For each opponent move (up to 4):
    - Clone game state → apply both moves → evaluate
  - Take the minimum score (opponent plays worst case for us)
- Pick our move with the maximum of these minimums

**Evaluation function (simple for now):**
- If we're dead: -1000
- If opponent is dead: +1000
- Otherwise: `FloodFill(ourHead)` — space available to us

**Implementation:**
- **New file:** `logic/search.go`
- `func (g *GameSim) BestMove(myID string) Direction`
- Construct moves map, call `Clone().Step()`, evaluate
- Bridge in `main.go`: convert API state → `GameSim`, call `BestMove`, return direction

**Key design decision:** The `move()` function in `main.go` will switch from direct flood-fill scoring to using `GameSim.BestMove()`. The flood-fill + food heuristic from Iters 1-2 becomes the evaluation function inside the search, not the top-level decision maker.

**Files:**
| File | Action |
|------|--------|
| `logic/search.go` | **New** — `BestMove()`, 1-ply paranoid minimax |
| `logic/eval.go` | **New** — `Evaluate(g *GameSim, myID string) float64` |
| `logic/search_test.go` | **New** — known positions with expected best moves |
| `main.go` | Switch `move()` to use GameSim + BestMove |

**Verify:** `make compare` against Iter 2 snapshot.

**Results:**
| Metric | Value |
|--------|-------|
| Avg turns (self-play) | ~87 (up from ~28) |
| vs v2 (flood-fill) win rate | 16% (50 games) |
| Tests | 7 new (total: 64) |

> Loses to v2 flood-fill despite better self-play turns. Root causes: (1) paranoid worst-case
> assumption at depth 1 is too conservative — the real opponent rarely plays the worst move for us;
> (2) food urgency was missing from `Evaluate()` initially (0% → 16% after fix). Both issues
> resolve naturally with deeper search (Iter 6) and a better eval (Iters 8-9).

---

### Iteration 6 — Deeper Minimax + Alpha-Beta Pruning

**Status:** TODO
**Depends on:** Iteration 5
**Expected improvement:** Moderate. Deeper search finds moves that are safe 2-3 turns ahead, not just 1. Alpha-beta makes it affordable.

**Goal:** Extend search to configurable depth with alpha-beta pruning to cut the search tree.

**Background:** Without pruning, depth-2 is 16×16 = 256 evaluations, depth-3 is 4096. Alpha-beta prunes branches that can't affect the result, typically reducing the effective branching factor from 16 to ~6-8 with good move ordering.

**Implementation:**
- Modify `BestMove` to accept a depth parameter
- Implement recursive `minimax(g *GameSim, depth int, alpha, beta float64, maximizing bool, myID string) float64`
- At depth 0 or game over: return `Evaluate()`
- Maximizing layer: try our 4 moves
- Minimizing layer: try opponent's 4 moves
- Alpha-beta cutoffs in both layers

**Start with depth 3.** Profile to ensure it stays under ~100ms on an 11x11 board.

**Files:**
| File | Action |
|------|--------|
| `logic/search.go` | Rewrite BestMove with recursive minimax + alpha-beta |
| `logic/search_test.go` | Add depth-specific tests, verify pruning reduces node count |

**Verify:** `make compare` against Iter 5 snapshot. Should win convincingly. Log node count per move to track search efficiency.

---

### Iteration 7 — Iterative Deepening + Time Management

**Status:** TODO
**Depends on:** Iteration 6
**Expected improvement:** Small-moderate. Guarantees we always have a move within the time limit while searching as deep as possible.

**Goal:** Search depth 1, then depth 2, then depth 3, etc., until the time budget (~150ms) runs out. Return the best move from the deepest completed search.

**Why?** Fixed-depth search is fragile — some positions are simple (depth 5 in 10ms) while others are complex (depth 3 takes 200ms). Iterative deepening adapts automatically.

**Implementation:**
- `func (g *GameSim) BestMoveIterative(myID string, budget time.Duration) Direction`
- Start at depth 1, increment depth each iteration
- Check `time.Since(start) > budget` at the start of each depth iteration
- Store best move from each completed depth
- If a search at depth N completes, start depth N+1. If it doesn't complete in time, return the depth N-1 result
- Add `context.Context` or a simple deadline check in the minimax recursion to abort early

**Bonus:** The depth-1 result from iterative deepening provides a good "first guess" for move ordering in deeper searches (not implemented yet, but sets up Iteration 11).

**Files:**
| File | Action |
|------|--------|
| `logic/search.go` | Add `BestMoveIterative`, deadline checks in recursion |
| `logic/search_test.go` | Test that it respects time budgets, returns valid moves |
| `main.go` | Call `BestMoveIterative` with 150ms budget |

**Verify:** `make bench` — verify no timeouts in game logs. `make compare` against Iter 6 — should be comparable or slightly better.

---

## Phase 4: Smarter Evaluation

The evaluation function is the "brain" of the minimax. Better eval = better decisions at the same search depth. Each iteration here improves what the snake values.

### Iteration 8 — Voronoi Territory Evaluation

**Status:** TODO
**Depends on:** Iteration 7
**Expected improvement:** Moderate-large. Voronoi measures *controlled territory* rather than just *reachable space*, giving much better positional awareness.

**Goal:** Replace flood fill in the evaluation with Voronoi territory — the number of cells closer to us than to any opponent.

**Background:** Flood fill counts all reachable cells, but in a 1v1, both snakes can reach most of the board. Voronoi assigns each cell to the nearest snake head (by BFS distance), so it measures who *controls* the space. A snake with 70 Voronoi cells vs 40 is winning the territorial battle.

**Implementation:**
- **New file:** `logic/voronoi.go`
- `func (g *GameSim) Voronoi(myID string) (myTerritory, opponentTerritory int)`
- Multi-source BFS: seed queue with all living snake heads, expand simultaneously. Each cell is claimed by the first snake to reach it. Ties go unclaimed (neutral).
- Use in `Evaluate()`: `score = myTerritory - opponentTerritory`

**Files:**
| File | Action |
|------|--------|
| `logic/voronoi.go` | **New** — multi-source BFS Voronoi |
| `logic/voronoi_test.go` | **New** — symmetric board, one snake cornered, partition |
| `logic/eval.go` | Use Voronoi instead of FloodFill |

**Verify:** `make compare` against Iter 7 snapshot. Expect better positional play — snake should control center and cut off opponent.

---

### Iteration 9 — Composite Evaluation Function

**Status:** TODO
**Depends on:** Iteration 8
**Expected improvement:** Moderate. Multi-factor eval catches scenarios that territory alone misses (starvation risk, length advantage, food positioning).

**Goal:** Combine multiple factors into a weighted evaluation score. Tune weights by comparing snapshots.

**Evaluation components:**
1. **Territory** (from Voronoi): `w1 * (myTerritory - oppTerritory)`
2. **Length advantage**: `w2 * (myLength - oppLength)` — longer snake wins head-to-head
3. **Health safety**: penalty when health < 30, bonus when health > 70
4. **Food proximity**: `w3 * (1.0 / distToNearestFood)` — scaled by health urgency
5. **Center control**: small bonus for head near board center (more escape routes)

**Initial weights:** Start with territory dominant (w1=10, w2=3, w3 varies by health, center=1). Tune based on `make compare` results.

**Files:**
| File | Action |
|------|--------|
| `logic/eval.go` | Expand `Evaluate()` with all components |
| `logic/eval_test.go` | **New** — test each component in isolation, verify scoring direction |

**Verify:** `make compare` against Iter 8. Tweak weights if needed, re-snapshot, re-compare.

---

### Iteration 10 — Aggression + Kill Detection

**Status:** TODO
**Depends on:** Iteration 9
**Expected improvement:** Moderate. The snake will actively try to eliminate opponents rather than just surviving.

**Goal:** Detect and exploit head-to-head collision opportunities. When we're longer, aggressively move toward the opponent's head. When shorter, avoid their head zone.

**Two components:**

**A. Head-to-head awareness:**
- If our length > opponent length, moves that put our head adjacent to their head are *good* (we win the collision)
- If our length ≤ opponent length, such moves are *bad* (we die or mutual kill)
- Add this as a bonus/penalty in `Evaluate()`

**B. Kill detection:**
- After our move, count opponent's safe moves (moves that don't lead to wall/body collision)
- If the opponent has 0 safe moves, that's a forced kill — max bonus
- If the opponent has only 1 safe move, we might be able to cut it off — bonus
- This naturally emerges from deeper search, but explicit detection helps at shallow depths

**Files:**
| File | Action |
|------|--------|
| `logic/eval.go` | Add head-to-head and kill-detection bonuses |
| `logic/eval_test.go` | Test aggression scoring: longer-vs-shorter, cornered opponent |

**Verify:** `make compare` — expect higher win rate, shorter average game length (kills happen earlier).

---

## Phase 5: Search Optimization

### Iteration 11 — Move Ordering

**Status:** TODO
**Depends on:** Iteration 10
**Expected improvement:** Indirectly large. Better move ordering means alpha-beta prunes more branches, allowing deeper search in the same time budget. Net effect: searches 1-2 plies deeper.

**Goal:** Try the most promising moves first so alpha-beta cutoffs happen earlier.

**Techniques:**
1. **Previous best move first:** The best move from depth N-1 (via iterative deepening) is tried first at depth N
2. **Killer heuristic:** Moves that caused cutoffs at the same depth in sibling nodes are tried early
3. **Static ordering:** Moves toward center > moves toward walls. Moves toward food > moves away.

**Implementation:**
- Add `moveOrder []Direction` parameter to minimax
- In `BestMoveIterative`, pass the previous depth's best move to the next depth
- Track killer moves per depth in a small array

**Files:**
| File | Action |
|------|--------|
| `logic/search.go` | Add move ordering logic, killer heuristic tracking |
| `logic/search_test.go` | Verify node count decreases with ordering vs without |

**Verify:** Log average depth reached per move. Compare against Iter 10 — same or better results, but searching deeper.

---

### Iteration 12 — Transposition Table + Zobrist Hashing

**Status:** TODO
**Depends on:** Iteration 11
**Expected improvement:** Moderate. Avoids re-evaluating positions reached via different move orders. Enables 1-2 additional plies of effective depth.

**Goal:** Cache evaluated positions so the search never evaluates the same game state twice.

**Zobrist hashing:**
- Pre-generate random `uint64` values for each (cell, piece-type) combination
- Board hash = XOR of all piece hashes → O(1) incremental update on move
- Use hash as transposition table key

**Transposition table:**
- Fixed-size hash map (e.g., 1M entries)
- Store: hash, depth, score, best move, flag (exact/lower-bound/upper-bound)
- On lookup: if stored depth ≥ current depth, use cached result
- Replace strategy: always-replace (simplest, effective enough)

**Files:**
| File | Action |
|------|--------|
| `logic/zobrist.go` | **New** — hash generation, incremental update |
| `logic/ttable.go` | **New** — transposition table with lookup/store |
| `logic/search.go` | Integrate TT lookups and stores |
| `logic/zobrist_test.go` | **New** — hash consistency, incremental vs full recompute |

**Verify:** Log TT hit rate. `make compare` against Iter 11 — should search noticeably deeper.

---

### Iteration 13 — Memory + Performance Optimization

**Status:** TODO
**Depends on:** Iteration 12
**Expected improvement:** Enables deeper search by reducing per-node cost. Hard to quantify in isolation — measured by search depth increase.

**Goal:** Reduce allocations in the hot path (Clone, Step, Evaluate) so each minimax node is cheaper.

**Techniques:**
1. **`sync.Pool` for GameSim clones:** Pool pre-allocated GameSim structs, reset and reuse instead of allocating
2. **Pre-allocated slice backing arrays:** Clone uses `copy()` into pooled slices instead of `make()`
3. **Avoid map allocations in Step:** Use fixed-size arrays for move maps (4 snakes max × 4 directions)
4. **Profile-guided:** Run `go tool pprof` on a bench run, fix the top 3 allocation hotspots

**Files:**
| File | Action |
|------|--------|
| `logic/sim.go` | Add `sync.Pool`, optimize Clone/Step allocations |
| `logic/pool.go` | **New** (if needed) — pool management |
| `logic/search.go` | Use pooled GameSim in minimax loop |

**Verify:** `go test -bench . -benchmem ./logic/` before and after. `make compare` to confirm no regression.

---

## Phase 6: Advanced (Stretch Goals)

These iterations are optional and depend on how competitive the snake already is after Phase 5.

### Iteration 14 — Opponent Modeling

**Status:** TODO (stretch)
**Depends on:** Iteration 13

**Goal:** Track opponent behavior patterns across turns and bias the minimax opponent model accordingly.

- Track opponent's last N moves and detected style (aggressive, defensive, random)
- Weight the minimax opponent moves by observed probability instead of assuming worst case
- If opponent tends to chase food, predict food-seeking moves
- If opponent plays randomly, search shallower (save time) since opponent isn't dangerous

**Risk:** Over-fitting to opponent patterns can backfire if they change strategy. Keep the worst-case assumption as a fallback.

---

### Iteration 15 — Parameter Tuning Tournament

**Status:** TODO (stretch)
**Depends on:** Iteration 13

**Goal:** Systematically tune evaluation weights via automated tournaments.

- Create a parameter config struct with all weights
- Run round-robin tournaments: current best vs variations (±20% on each weight)
- Hill-climbing: keep the winner, vary again
- Use `make bench` infrastructure with different configs loaded via env vars or flags
- Target: 500+ games per config pair for statistical significance

---

## Snapshot Log

Track all snapshots here for easy reference in `make compare` commands.

| Iteration | Snapshot | Avg Turns | Notes |
|-----------|----------|-----------|-------|
| 0 (baseline) | — | ~68 | Random safe-move, never snapshotted |
| 1 | `snapshots/haruko-244a28f` | ~78 (self-play) | Flood fill + space-aware |
| 2 | `snapshots/haruko-244a28f` | ~28 (self-play) | Food-seeking heuristic, 95% vs v1 |
| 3 | — | — | Infrastructure only, no behavioral change |
| 4 | — | — | Infrastructure only, no behavioral change |
| 5 | `snapshots/haruko-7d164ae` | ~87 (self-play), 16% vs v2 | 1-ply paranoid minimax; loses to flood-fill (see Iter 5 notes) |
| 6 | | | |
| 7 | | | |
| 8 | | | |
| 9 | | | |
| 10 | | | |
| 11 | | | |
| 12 | | | |
| 13 | | | |

---

## Architecture Overview

```
main.go                 ← HTTP handlers, API type bridge
models.go               ← Battlesnake API types
server.go               ← HTTP server

logic/
  board.go              ← FastBoard (1D grid, cell types, Update)     [Iter 1]
  flood.go              ← Direction, FloodFill, Coord.Move            [Iter 1]
  food.go               ← Food distance heuristics                    [Iter 2]
  sim.go                ← GameSim (Clone, Step, full rules)           [Iter 3-4]
  search.go             ← Minimax, alpha-beta, iterative deepening    [Iter 5-7]
  eval.go               ← Evaluation function (composite scoring)     [Iter 5, 9-10]
  voronoi.go            ← Multi-source BFS territory counting         [Iter 8]
  zobrist.go            ← Zobrist hashing for positions               [Iter 12]
  ttable.go             ← Transposition table                         [Iter 12]

cmd/bench/main.go       ← Benchmark runner (already exists)
```
