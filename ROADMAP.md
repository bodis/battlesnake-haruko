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
| **Completed** | Iteration 9 |
| **Next** | Iteration 10 |
| **Baseline** | v0 random safe-move: ~68 avg turns (self-play) |
| **Current** | v9 iterative deepening (300ms budget) + composite eval; ~330 avg turns self-play |
| **Key insight** | Iterative deepening ensures safe time management. Deeper search with good eval finds better moves. |

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

### Iteration 6 — Deeper Minimax + Alpha-Beta Pruning ✅

**Status:** DONE
**Depends on:** Iteration 5
**Snapshot:** `snapshots/haruko-f344869` (binary built from uncommitted v6 code on v5 commit)

**Goal:** Extend search to configurable depth with alpha-beta pruning to cut the search tree.

**What was built:**
- `BestMove(myID string, depth int)` — top-level max over our 4 moves
- `minimaxMin(...)` — minimizing layer: enumerates all opponent move combos, applies Step, beta cutoff
- `minimaxMax(...)` — maximizing layer: tries our 4 moves, alpha cutoff
- `forEachOppCombo` callback changed to `func(map[string]Direction) bool` for early exit on pruning
- Default depth = 3 in `main.go`

**Files touched:**
| File | Action |
|------|--------|
| `logic/search.go` | Rewrite: `BestMove(myID, depth)`, `minimaxMin`, `minimaxMax`, early-exit `forEachOppCombo` |
| `logic/search_test.go` | Updated all calls to new signature, added `TestBestMove_DepthComparison` |
| `main.go` | `BestMove(myID)` → `BestMove(myID, 3)` |

**Results:**
| Metric | Before (v5) | After (v6) |
|--------|-------------|------------|
| Avg turns (self-play) | ~87 | ~328 |
| vs v2 (flood-fill) win rate | 16% | 30% (50 games) |
| Tests | 64 | 65 |

> Self-play turns nearly 4× longer — deeper search avoids short-term traps. Win rate vs v2 doubled
> (16% → 30%) but still losing. **Root cause confirmed:** the eval function is the bottleneck, not
> search depth. Deeper search with a bad eval just makes conservative decisions more confidently.
> Three specific eval problems: (1) only measures our space, not relative territory — opponent can
> control 70% of the board and we don't penalize it; (2) food urgency too weak to prevent starvation;
> (3) enemy heads treated as fully blocked, making us overly avoidant even when we're longer.

---

## Phase 3b: Eval Fix (Priority Reorder)

> **Critical pivot:** The original roadmap had search optimization (iterative deepening, move ordering)
> before eval improvements. Iterations 5-6 proved this is backwards — eval is the bottleneck.
> Iterations 7-8 now fix the eval before returning to search optimization in Iter 9-10.

### Iteration 7 — Voronoi Territory + Food Urgency (Eval Overhaul) ✅

**Status:** DONE
**Depends on:** Iteration 6
**Snapshot:** `snapshots/haruko-3aac093`

**What was built:**
- `logic/voronoi.go` — `VoronoiTerritory(g, myID)`: multi-source BFS from all alive snake heads simultaneously. Each cell claimed by first snake to reach it; ties unclaimed. Interior body segments blocked, tails passable.
- `logic/eval.go` — replaced `floodFillSim` with Voronoi territory difference (`myCount - oppCount`). Added food urgency: when health < 40, adds `foodWeight * (1/dist)` where `foodWeight = (40-health) * 0.5` (scales 0→20 as health drops 40→0).

**Files:**
| File | Action |
|------|--------|
| `logic/voronoi.go` | **New** — multi-source BFS Voronoi territory |
| `logic/voronoi_test.go` | **New** — symmetric board, cornered snake, body-wall partition |
| `logic/eval.go` | Replace flood fill with Voronoi, fix food urgency scaling |

**Results:**
| Metric | Before (v6) | After (v7) |
|--------|-------------|------------|
| Avg turns (self-play, 10 games) | ~328 | ~250 (noisy at N=10) |
| vs v6 win rate (50 games) | — | **98%** (49/50) |

> 98% win rate vs v6 confirms Voronoi territory is a massive eval upgrade. The `Turn` field is now also correctly propagated from the API state (was hardcoded to 0 before).

---

### Codebase Cleanup (post-Iter 7) ✅

**Status:** DONE
**Depends on:** Iteration 7

**Goal:** Remove all dead code now that GameSim + Voronoi is the active path. FastBoard, FloodFill, and NearestFoodDistance are unused.

**What was removed / refactored:**
- Deleted `logic/food.go`, `logic/food_test.go`, `logic/board_test.go`, `logic/flood_test.go`
- Deleted `logic/board.go` (FastBoard + Cell constants) and `logic/flood.go` (FloodFill)
- Created `logic/types.go` — all shared types in one place: `Coord`, `Snake`, `Direction`, `AllDirections`, `DirectionName()`, `Coord.Move()`
- `logic/voronoi.go` — replaced hardcoded `dx/dy` arrays with `AllDirections` + `Coord.Move()`
- `logic/sim.go` — extracted `cloneSnakes()` helper, simplified `NewGameSim`/`Clone`
- `main.go` — `gameSimFromState()` builds `GameSim` struct directly (no double-copy through `NewGameSim`)

---

## Phase 4: Smarter Evaluation

> **Why eval before search optimization?** Iterations 5-6 proved that deeper search with a bad eval
> is counterproductive. Minimax at depth 3 with space-only eval still loses to simple flood-fill (30%
> win rate). The eval must correctly measure board control before deeper search adds value.

### Iteration 8 — Composite Eval: Length + Aggression ✅

**Status:** DONE
**Depends on:** Iteration 7
**Snapshot:** `snapshots/haruko-85b3726`

**Goal:** The snake should understand that being longer = safer (wins head-to-head), and should actively trap shorter opponents rather than just passively controlling space.

**Evaluation components (added to Iter 7 base):**
1. **Length advantage**: `w_len * (myLength - oppLength)` — longer snake wins head-to-head, so growing is valuable. Suggested `w_len = 2.0`.
2. **Head-to-head pressure**: If we're longer and our head is adjacent to the opponent's head, bonus (we threaten a kill). If shorter, penalty (we're in danger). `w_h2h * (1 if longer, -1 if shorter) * (1 if adjacent, 0 if not)`. Suggested `w_h2h = 5.0`.
3. **Opponent safe moves**: Count how many of the opponent's 4 moves don't immediately die (wall/body). If 0, forced kill → big bonus (+50). If 1, near-kill → moderate bonus (+15). This helps the search see traps at shallow depth.

**Key constraint:** Territory (from Voronoi) should remain the dominant factor (weight ~10). Length and aggression are secondary signals (weight 2-5). Food urgency from Iter 7 stays as-is.

**Files:**
| File | Action |
|------|--------|
| `logic/eval.go` | Add length advantage, head-to-head pressure, opponent safe-move count |
| `logic/eval_test.go` | Test each new component: longer-vs-shorter scoring, cornered opponent detection |

**Results:**
| Metric | Before (v7) | After (v8) |
|--------|-------------|------------|
| Avg turns (self-play) | ~250 | ~330 |
| vs v7 win rate (100 games) | — | 88% |

> Composite eval with length advantage, h2h pressure, and opponent confinement gives 88% win rate over
> territory-only eval. Longer self-play turns indicate fewer premature deaths.

---

## Phase 4b: Search Optimization (with good eval)

> Now that the eval correctly measures board control (Voronoi + composite), deeper search actually
> finds better moves instead of just amplifying a bad heuristic.

### Iteration 9 — Iterative Deepening + Time Management

**Status:** DONE
**Depends on:** Iteration 8

**Why now (not earlier):** With the bad eval from v5/v6, deeper search was counterproductive — it just amplified the eval's mistakes. Now that Voronoi + composite eval correctly measures board control, deeper search finds genuinely better moves.

**Goal:** Search depth 1, then depth 2, then depth 3, etc., until the time budget (~300ms) runs out. Return the best move from the deepest completed search.

**Implementation:**
- `func (g *GameSim) BestMoveIterative(myID string, budget time.Duration) Direction`
- Start at depth 1, increment depth each iteration
- Check `time.Since(start) > budget * 0.7` before starting next depth (leave margin)
- Store best move from each completed depth
- If a search at depth N completes, start depth N+1. If it doesn't complete in time, return the depth N result
- Add deadline check inside `minimaxMin`/`minimaxMax` — if time expired, return current best estimate and stop recursing

**Files:**
| File | Action |
|------|--------|
| `logic/search.go` | Add `BestMoveIterative`, deadline checks in minimax |
| `logic/search_test.go` | Test that it respects time budgets, returns valid moves at all depths |
| `main.go` | Call `BestMoveIterative` with 300ms budget |

**What was built:**
- `searchContext` struct with deadline + timedOut flag for time management
- `BestMoveIterative(myID, budget)` — iterative deepening loop, searches depth 1, 2, 3, ... within budget (capped at depth 5)
- Modified `minimaxMin`/`minimaxMax` to accept optional `ctx *searchContext` — bail early on timeout
- 300ms budget in `main.go` (leaves 200ms margin for the 500ms server timeout)
- `BestMove` still works unchanged (passes `nil` ctx)
- Max depth capped at 5: deeper paranoid search is counterproductive (assumes perfect opponent play, becomes overly defensive)

**Results:**
| Metric | Before (v8) | After (v9) |
|--------|-------------|------------|
| Avg turns (self-play, 100 games) | ~330 | ~306 |
| vs v8 win rate (100 games) | — | **76%** |

> Iterative deepening with depth cap of 5 beats fixed depth 3. Deeper search (depth 7+) was
> counterproductive due to paranoid minimax's worst-case assumption — the opponent "plays perfectly"
> at every level, making the snake overly defensive. The depth 5 cap balances deeper lookahead with
> realistic opponent modeling.
>
> **Key insight:** Paranoid minimax has diminishing and eventually negative returns with depth.
> Uncapped search (reaching depth 7) scored 0% vs Iter 8 — the snake saw threats everywhere and
> froze up. Two paths to unlock deeper search: (1) move ordering (Iter 10) makes existing depth
> more efficient; (2) switching from paranoid to a probabilistic opponent model (Iter 13) would
> reduce the pessimism that makes deep search harmful.

---

### Iteration 10 — Move Ordering + Killer Heuristic

**Status:** TODO
**Depends on:** Iteration 9
**Expected improvement:** Indirectly large. Better move ordering means alpha-beta prunes more, allowing 1-2 deeper plies in the same time budget. Combined with iterative deepening, this is a significant depth boost.

**Goal:** Try the most promising moves first so alpha-beta cutoffs happen earlier.

**Techniques:**
1. **Previous best move first:** The best move from depth N-1 (via iterative deepening) is tried first at depth N
2. **Killer heuristic:** Moves that caused cutoffs at the same depth in sibling nodes are tried early
3. **Static ordering:** Moves toward center > walls. Moves toward food when hungry.

**Implementation:**
- Add `moveOrder []Direction` parameter to minimax
- In `BestMoveIterative`, pass the previous depth's best move to the next depth
- Track killer moves per depth in a small array

**Files:**
| File | Action |
|------|--------|
| `logic/search.go` | Add move ordering logic, killer heuristic tracking |
| `logic/search_test.go` | Verify node count decreases with ordering vs without |

**Verify:** Log average depth reached per move. Compare against Iter 9 — same or better results, but searching deeper.

---

## Phase 5: Advanced Search

### Iteration 11 — Transposition Table + Zobrist Hashing

**Status:** TODO
**Depends on:** Iteration 10
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

**Verify:** Log TT hit rate. `make compare` against Iter 10 — should search noticeably deeper.

---

### Iteration 12 — Memory + Performance Optimization

**Status:** TODO
**Depends on:** Iteration 11
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

### Iteration 13 — Opponent Modeling

**Status:** TODO (stretch)
**Depends on:** Iteration 12

**Goal:** Track opponent behavior patterns across turns and bias the minimax opponent model accordingly.

- Track opponent's last N moves and detected style (aggressive, defensive, random)
- Weight the minimax opponent moves by observed probability instead of assuming worst case
- If opponent tends to chase food, predict food-seeking moves
- If opponent plays randomly, search shallower (save time) since opponent isn't dangerous

**Risk:** Over-fitting to opponent patterns can backfire if they change strategy. Keep the worst-case assumption as a fallback.

---

### Iteration 14 — Parameter Tuning Tournament

**Status:** TODO (stretch)
**Depends on:** Iteration 12

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
| 6 | `snapshots/haruko-f344869` | ~328 (self-play), 30% vs v2 | Depth-3 alpha-beta; still loses — eval is bottleneck |
| 7 | `snapshots/haruko-3aac093` | ~250 self-play (N=10), 98% vs v6 | Voronoi territory + food urgency eval overhaul |
| 8 | `snapshots/haruko-85b3726` | ~330 (self-play), 88% vs v7 | Composite eval: length + aggression + confinement |
| 9 | `snapshots/haruko-83cd760` | ~306 (self-play), 76% vs v8 | Iterative deepening (300ms, max depth 5) |
| 10 | | | Move ordering |
| 11 | | | Transposition table |
| 12 | | | Memory optimization |

---

## Architecture Overview

```
main.go                 ← HTTP handlers, API type bridge
models.go               ← Battlesnake API types
server.go               ← HTTP server

logic/
  types.go              ← Coord, Snake, Direction, AllDirections, Coord.Move  [Iter 1, cleanup]
  sim.go                ← GameSim (Clone, Step, full rules)                   [Iter 3-4]
  search.go             ← Minimax, alpha-beta, iterative deepening            [Iter 5-6, 9-10]
  eval.go               ← Evaluation function (Voronoi + food urgency)        [Iter 5, 7-8]
  voronoi.go            ← Multi-source BFS territory counting                 [Iter 7]
  zobrist.go            ← Zobrist hashing for positions                       [Iter 11]
  ttable.go             ← Transposition table                                 [Iter 11]

cmd/bench/main.go       ← Benchmark runner (already exists)
```
