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
