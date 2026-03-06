# Haruko Battlesnake — Completed Iterations

> Archive of all completed iterations. Moved from ROADMAP.md to keep the active roadmap focused.
> Each iteration was a self-contained unit: implement → test → snapshot → compare → merge.

---

## Phase 1: Survival Foundation

### Iteration 1 — Flood Fill + Space-Aware Moves

**Status:** DONE
**Snapshot:** (pre-snapshot era)

**What was built:**
- `InBounds(x, y)` on FastBoard
- `CellSnakeTail` constant — tail awareness in `Update()` (non-stacked tails marked passable)
- `Direction` type + constants (`Up/Down/Left/Right`), `DirectionName()`, `Coord.Move(d)` in `logic/flood.go`
- `FloodFill(start)` — BFS counting reachable cells (empty + food + tail are passable)
- `move()` in `main.go` scores each safe move by flood-fill reachable space, picks highest

**Results:** ~68 → ~78 avg turns. Draw rate 10% → 26%.

---

### Iteration 2 — Food-Seeking Heuristic

**Status:** DONE
**Snapshot:** `snapshots/haruko-244a28f`

**What was built:**
- Manhattan distance to closest food, combined with flood-fill into composite score
- Health-weighted food urgency: low health → food priority overrides space

**Results:** 95% vs v1. Avg turns dropped to ~28 (food-chase collisions in self-play).

---

## Phase 2: Game Simulator

### Iteration 3 — Game Simulator Core

**Status:** DONE

**What was built:**
- `SimSnake` struct with `Head()` / `Tail()` accessors
- `GameSim` with `Clone()`, `MoveSnakes()`, `SnakeByID()`
- Deep-copy semantics, no shared backing arrays

**Results:** Infrastructure only, no behavioral change.

---

### Iteration 4 — Game Simulator Rules

**Status:** DONE

**What was built:**
- `GameSim.Step(moves)` — full 7-phase turn execution matching official Battlesnake Standard rules
- `IsAlive()`, `IsOver()`, `HazardDamage = 14`

**Results:** Infrastructure only, no behavioral change.

---

## Phase 3: Search

### Iteration 5 — 1-Ply Lookahead (Paranoid Minimax)

**Status:** DONE
**Snapshot:** `snapshots/haruko-7d164ae`

**What was built:**
- `BestMove(myID, depth)` — paranoid minimax, worst-case over all opponent moves
- Simple eval: dead = -1000, opponent dead = +1000, else flood-fill space

**Results:** ~87 avg turns, 16% vs v2. Deeper search with bad eval is counterproductive.

---

### Iteration 6 — Deeper Minimax + Alpha-Beta Pruning

**Status:** DONE
**Snapshot:** `snapshots/haruko-f344869`

**What was built:**
- `minimaxMin` / `minimaxMax` with alpha-beta pruning
- `forEachOppCombo` with early exit for cutoffs
- Default depth = 3

**Results:** ~328 avg turns, 30% vs v2. Eval confirmed as bottleneck, not search depth.

---

## Phase 3b: Eval Fix

### Iteration 7 — Voronoi Territory + Food Urgency (Eval Overhaul)

**Status:** DONE
**Snapshot:** `snapshots/haruko-3aac093`

**What was built:**
- `VoronoiTerritory(g, myID)` — multi-source BFS, body blocks, tails passable
- Eval: territory difference + health-gated food urgency

**Results:** 98% vs v6. Voronoi territory is a massive eval upgrade.

---

### Codebase Cleanup (post-Iter 7)

**Status:** DONE

Removed all dead code: FastBoard, FloodFill, NearestFoodDistance. Created `logic/types.go` with shared types.

---

## Phase 4: Smarter Evaluation

### Iteration 8 — Composite Eval: Length + Aggression

**Status:** DONE
**Snapshot:** `snapshots/haruko-85b3726`

**What was built:**
- Length advantage: `wLen * (myLength - oppLength)`
- Head-to-head pressure: bonus when longer and adjacent, penalty when shorter
- Opponent confinement: 0 safe moves = +50, 1 safe move = +15

**Results:** 88% vs v7. ~330 avg turns.

---

## Phase 4b: Search Optimization

### Iteration 9 — Iterative Deepening + Time Management

**Status:** DONE
**Snapshot:** `snapshots/haruko-83cd760`

**What was built:**
- `BestMoveIterative(myID, budget)` — depth 1→5 within 300ms
- `searchContext` with deadline + timedOut flag
- Depth capped at 5 (paranoid minimax degrades at 7+)

**Results:** 76% vs v8. ~306 avg turns.

---

### Iteration 10 — Move Ordering + Killer Heuristic

**Status:** DONE
**Snapshot:** `snapshots/haruko-c12e218`

**What was built:**
- `orderedMoves(pv, killers)` — PV first, killers next, rest last
- `killerTable` — 2 moves per depth that caused beta cutoffs
- PV from previous depth passed to next iteration

**Results:** 54% vs v9, 75% vs v8.

---

## Phase 5: Advanced Search

### Iteration 11 — Transposition Table + Zobrist Hashing

**Status:** DONE
**Snapshot:** `snapshots/haruko-0bf91d3`

**What was built:**
- Zobrist hashing (`GameSim.Hash()`)
- TT with 1M entries, generation-based invalidation, singleton reuse
- Max depth raised 5→6

**Results:** 65% vs v10. TT hit rate ~25% at depth 6. Shorter self-play (197) = stronger play.

---

### Iteration 12 — Best-Reply Search (Algorithm Change)

**Status:** DONE
**Snapshot:** `snapshots/haruko-cee3f49`

**What was built:**
- `brsMax` / `brsMin` — 2-player minimax, only "best replier" opponent per ply
- BF drops from 16/round to 4/ply, enabling depth 14
- Paranoid minimax retained for multi-opponent fallback

**Results:** 59% vs v11. ~213 avg turns.

---

## Phase 6: Search Refinement

### Iteration 13 — Eval Hardening + Quiescence Search

**Status:** DONE (QS not wired — too expensive)

**What was built:**
- `safeMoveCount(g, s)` helper extracted
- `Evaluate()` generalized to N opponents
- QS infrastructure: `isQuiet`, `forcingMoves`, `qsMax`/`qsMin` (built, not active)

**Results:** ~50% vs v12 (neutral — N-opponent loop is no-op in 1v1). QS tested in 5 configs, all ≤51%.

---

### Iteration 14 — Performance Optimization

**Status:** DONE
**Snapshot:** `snapshots/haruko-ad0f0f3`

**What was built:**
- `sync.Pool` for `GameSim` clones (`CloneFromPool`/`Release`)
- `MoveSet` array-based (replaces `map[string]Direction`)
- Pooled Voronoi workspace
- Fixed-size arrays in `Step` (eaten, elims, heads)

**Results:** 56% vs v12. CloneFromPool: 19ns/0 allocs. Step: 49ns/0 allocs. Evaluate: 1090ns/0 allocs.

---

### Iteration 15 — Search Pruning + Extensions (FAILED)

**Status:** DONE — all techniques tested, none effective.

**Tested:** LMR, NMP, volatile position extensions — every combination ≤50% vs v14.
**Root cause:** BRS has only 4x4=16 nodes per ply pair. Alpha-beta with TT+killers already prunes efficiently. These techniques need high-BF games.

---

## Phase 7: Strategic Evaluation

### Iteration 16 — Rich Voronoi + Food Control

**Status:** DONE

**What was built:**
- `VoronoiResult` struct: `MyTerritory`, `OppTerritory`, `MyFood`, `OppFood`, `IsPartitioned`
- Food counting and partition detection from existing BFS data

**Results:** Infrastructure for Iter 17. Constant-weight food control and partition short-circuit both failed as standalone features (28-51%).

---

### Iteration 17 — Game Phase + Adaptive Weights

**Status:** DONE

**What was built:**
- `earlyBlend` (0-1): max of length-based and turn-based factors
- `lateBlend` (0-1): board fill ratio, boosted on partition
- Phase-modulated weights for territory, length, h2h, food threshold, food control
- Tail chase bonus in late game

**Results:** 59% vs v16. ~451 avg turns. Evaluate: ~1090ns/0 allocs (unchanged).

---

### Iteration 18 — Heuristic Move Ordering (FAILED)

**Status:** DONE — tested, not effective.

**Tested:** isSafeDir-based ordering at BRS call sites. TT+killers already handle the 1-2 best moves; reordering the remaining 2-3 has negligible cutoff impact. Center proximity tiebreaker actively misleads.

**Results:** 47-51.5% vs v17.

---

## Snapshot Log

| Iteration | Snapshot | Avg Turns | Notes |
|-----------|----------|-----------|-------|
| 0 (baseline) | — | ~68 | Random safe-move, never snapshotted |
| 1 | `snapshots/haruko-244a28f` | ~78 | Flood fill + space-aware |
| 2 | `snapshots/haruko-244a28f` | ~28 | Food-seeking heuristic, 95% vs v1 |
| 3 | — | — | Infrastructure only |
| 4 | — | — | Infrastructure only |
| 5 | `snapshots/haruko-7d164ae` | ~87 | 1-ply paranoid minimax; 16% vs v2 |
| 6 | `snapshots/haruko-f344869` | ~328 | Depth-3 alpha-beta; 30% vs v2 |
| 7 | `snapshots/haruko-3aac093` | ~250 | Voronoi + food urgency; 98% vs v6 |
| 8 | `snapshots/haruko-85b3726` | ~330 | Composite eval; 88% vs v7 |
| 9 | `snapshots/haruko-83cd760` | ~306 | Iterative deepening; 76% vs v8 |
| 10 | `snapshots/haruko-c12e218` | ~417 | Move ordering + killer; 54% vs v9 |
| 11 | `snapshots/haruko-0bf91d3` | ~197 | TT + Zobrist; 65% vs v10 |
| 12 | `snapshots/haruko-cee3f49` | ~213 | BRS; 59% vs v11 |
| 13 | — | ~200 | Eval hardening + QS infra (not wired) |
| 14 | `snapshots/haruko-ad0f0f3` | ~215 | Zero-alloc hot path; 56% vs v12 |
| 15 | — | — | Failed: search pruning |
| 16 | — | — | VoronoiResult infrastructure |
| 17 | — | ~451 | Phase-adaptive eval; 59% vs v16 |
| 18 | — | — | Failed: heuristic move ordering |
| 19 | — | — | Voronoi strategic extraction (infra) |
| 20 | `snapshots/haruko-a989fbb` | ~443 | Food strategy signals; 54% vs v19 |

---

## Findings

Key technical insights discovered during development.

### Eval > Search Depth (Iter 5-7)
Deeper search with a bad eval is counterproductive. v6 (depth 3, space-only eval) lost to v2 (flood-fill heuristic). Only after Voronoi territory eval (v7) did deeper search add value.

### Paranoid Minimax Depth Ceiling (Iter 9, confirmed Iter 11)
Paranoid minimax assumes ALL opponents coordinate perfectly. Depth 7+: overly defensive, avg drops to ~150 turns. Depth 10+: catastrophic (~18 turns). Solved by BRS (Iter 12).

### TT Allocation Matters (Iter 11)
32MB TT per move call = GC pressure. Solution: singleton with generation-based invalidation.

### Self-Play Turns != Strength (Iter 11)
Shorter self-play can mean STRONGER play. Always verify with `make compare`.

### QS at BRS Leaves Too Expensive (Iter 13)
Each QS node costs Clone+Step+Evaluate. Extensions steal depth from main search. All 5 configs ≤51%.

### Depth Is King, But Only If Nodes Are Cheap (Iter 10-13)
+1 ply = big wins (65%, 59%). Fewer nodes same depth = modest (54%). More nodes current cost = loss (≤51%).

### Search Pruning Doesn't Work at Low BF (Iter 15)
LMR, NMP, extensions all ≤50%. BRS already narrow (4x4=16). These need high-BF games (chess ~35).

### Constant-Weight Food Control Fails (Iter 16)
Food value changes across phases. Flat weights average early benefit and late harm. Solved by phase-adaptive eval (Iter 17).

### Heuristic Move Ordering Negligible (Iter 18)
TT+killers already handle the best 1-2 moves. Reordering remaining 2-3 has no measurable cutoff impact at BF=4.

### Eval Signal Weights Are Sensitive (Iter 20)
Initial food strategy weights (2.0/0.8/3.0/4.0/0.5) scored 47% — worse than baseline. Halving to (1.5/0.5/2.0/2.5/0.3) yielded 54%. Starvation risk is especially sensitive: overweighting causes over-cautious play that sacrifices territory for food proximity.

### Key Principle
Every past win came from deeper search or better eval. Search mechanics (pruning, ordering) are saturated at BF=4. The remaining lever is **eval quality**.

---

## Phase 8: Strategic Board Understanding

### Iteration 19 — Voronoi Strategic Extraction

**Status:** DONE
**Depends on:** Iteration 17

**Goal:** Extract rich spatial signals from the existing Voronoi BFS at near-zero incremental cost. Infrastructure for Iter 20-23.

**What was built:**
- Extended `VoronoiResult` with 10 new fields extracted from the same BFS pass:
  - Food quality: `MyClosestFoodDist`, `OppClosestFoodDist`, `MyFoodValue` (sum of 1/dist)
  - Territory shape: `MyTerritoryDepth` (max BFS distance)
  - Positional: `MyCenterX/Y`, `OppCenterX/Y` (territory centroids)
  - Tail reachability: `MyTailReachable` (tail cell in own Voronoi territory)
- Enriched territory count loop with centroid accumulators and depth tracking
- Enriched food count loop with distance-weighted value and closest-food tracking
- 8 new test cases covering all new fields

**Cost:** Voronoi ~1025ns (was ~1015ns). Zero new allocations. Evaluate and BRS node unchanged.

**Result:** Infrastructure only — no behavioral change. Eval doesn't consume new fields until Iter 20.

### Iteration 20 — Food Strategy Signals

**Status:** DONE
**Depends on:** Iteration 19

**Goal:** Teach the eval to reason about food access quality, not just food count.

**What was built:**
- **Food cluster value**: Replaced flat `vr.MyFood` count with distance-weighted `vr.MyFoodValue` (sum of 1/dist). Weight 1.5 × earlyBlend.
- **Food reach advantage**: Reward having closer food access than opponent. Weight 0.5. Always-on.
- **Food denial**: Bonus when opponent has 0 food in territory and health < 40. Weight 2.0.
- **Starvation risk**: Penalty when we have 0 food in territory and health < 50. Weight 2.5.
- **Growth urgency**: Early-game penalty when snake length < expected for game turn. Weight 0.3 × earlyBlend.
- 3 new tests: FoodClusterValue, FoodDenial, GrowthUrgency

**Weight tuning:** Initial weights (2.0/0.8/3.0/4.0/0.5) scored 47% — too aggressive. Reduced to (1.5/0.5/2.0/2.5/0.3) for 54%.

**Cost:** Evaluate ~1130ns (was ~1090ns). Zero new allocations.

**Result:** 54% vs v19. ~443 avg turns.

---

### Iteration 23 — Territory Bottleneck Detection

**Status:** DONE
**Depends on:** Iteration 19

**Goal:** Detect territory bottlenecks — articulation points in the territory subgraph that opponents can exploit to sever corridor-shaped territory.

**Analysis finding:** In every loss traceback, the largest signal drop is territory (eval +55 → -90 in 1-2 turns). Voronoi counts cells but not cell quality — a corridor of 55 cells looks identical to a compact 55-cell region. The search can't see it because even at depth 14, leaf nodes report "+55 territory" until the corridor is actually cut.

**What was built:**
- **Tarjan's articulation point algorithm** on territory subgraph (iterative, zero-alloc)
- **Border filter**: Only count APs adjacent to non-owned cells ("live" APs — exploitable by opponent)
- **Dual-use**: `MyThreatenedTerritory` (defense) and `OppThreatenedTerritory` (attack)
- **Eval signal**: `wBottleneck = 0.3 × (0.5 + 0.5×lateBlend)` — 0.15 early, 0.3 late
- **Early exit**: Skip if territory < 8 cells
- **Dirty-list cleanup**: Only clear Tarjan arrays for cells actually visited (avoids full-array clear)
- 5 Voronoi tests: corridor, compact, opponent corridor, internal AP ignored, small territory
- 1 eval test: bottleneck signal fires correctly

**Cost:** Voronoi ~2400ns (was ~1025ns), Evaluate ~2450ns (was ~1130ns), BRS node ~2490ns (was ~1180ns). Zero allocations. Roughly halves effective search depth (~12-13 from ~14), but eval improvement more than compensates.

**Result:** 58% vs v20 (two independent N=50 runs both at 58%). ~287-350 avg turns.

---

### Iteration 21 — Positional Quality ❌ DEAD END

**Status:** DEAD END
**Depends on:** Iteration 19

All three signals (edge/corner penalty, territory depth adequacy, center-of-mass advantage) individually harmful (37–48%). Voronoi territory already captures positional quality implicitly — center positions get more territory, edge positions get less. Explicit positional signals double-count and confuse BRS. Tried halving weights (48%), individual isolation (37–45%), normalized center (43%). All negative. See ENGINE.md dead ends.

---

### Iteration 22 — Opponent Pressure & Aggression Mode ❌ DEAD END

**Status:** DEAD END
**Depends on:** Iteration 19

Dominance score (length+territory+food composite) used to modulate H2H range, confinement weights, health pressure, directional pressure. Tested 7 variants isolating each signal (42–49%). Root cause: in self-play, both sides use the same eval, so aggression modulation gives no asymmetric advantage. The search already finds aggressive moves when they lead to better positions. See ENGINE.md dead ends.

---

### Iteration 25 — Territory Shape Quality ❌ SUPERSEDED

**Status:** SUPERSEDED by Iteration 23
**Depends on:** Iteration 19

Original plan: detect corridor-shaped territory via thin-cell counting (cells with ≤1 owned neighbor). Iter 23's Tarjan AP detection captures the dangerous case directly — corridor territory that can be severed by opponent moves. Thin-cell counting would add a softer, redundant version of the same signal at additional eval cost. Skipped.
