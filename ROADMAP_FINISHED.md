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

## Phase 9: Optimization & Calibration

### Iteration 24 — Weight Calibration

**Status:** DONE
**Depends on:** Iterations 20, 23

**Goal:** Systematically A/B test all eval weights. Most were set by intuition. Iter 20 proved weights are sensitive.

**What was built:**
- Prioritized sweep of 12 weight tests (N=50 each), cumulative keep/revert
- 6 weights improved, 6 reverted (noise or loss)

**Weight changes:**

| Weight | Old | New | Test Win% | Decision |
|--------|-----|-----|-----------|----------|
| wBottleneck | 0.3 | 0.6 | 42% | REVERT |
| wBottleneck | 0.3 | 0.15 | 44% | REVERT |
| wTerritory | 1.0 | 1.5 | 56% | KEEP |
| wLen | 2.0 | 3.0 | 56% | KEEP |
| wH2H | 5.0 | 3.0 | 54% | REVERT |
| wH2H | 5.0 | 8.0 | 64% | KEEP |
| wTailChase | 3.0 | 5.0 | 58% | KEEP |
| wStarvationRisk | 2.5 | 1.5 | 62% | KEEP |
| wFoodDenial | 2.0 | 1.0 | 52% | REVERT |
| wFoodCluster | 1.5 | 1.0 | 62% | KEEP |
| wGrowthUrgency | 0.3 | 0.15 | 38% | REVERT |
| wFoodReach | 0.5 | 0.3 | 52% | REVERT |

**Key findings:**
- Core weights (territory, length, H2H) were all undertuned — the biggest wins
- H2H at 8.0 was the single strongest individual change (64%)
- Food weights continue to benefit from reduction (Iter 20 trend confirmed)
- wBottleneck 0.3 is well-calibrated — both directions lost
- wGrowthUrgency 0.3 is important — halving it lost badly (38%)

**Cost:** No change — same eval, just different constants.

**Result:** 61% vs v23 (N=100 confirmed). ~337 avg turns.

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

### Iteration 25 — Win/Loss Trace Analysis

**Status:** DONE
**Depends on:** Iteration 24

**Goal:** Comprehensive analysis of v24's game outcomes to guide next iteration.

**Findings (N=20 self-play games, 40 traces):**

1. **Win classification: 100% territory-squeeze.** No h2h-kills or starvation-kills.
2. **Death causes:** collision=50%, wall-collision=25%, body-collision=10%, starvation=10%.
3. **Death phase:** early=1, mid=9, late=10 (roughly even mid/late).
4. **Games decided by sudden 1-2 turn territory flips** (200+ eval swing) — beyond BRS horizon.
5. **LenAdvantage is the strongest win/loss differentiator** (not territory).
6. **OppConfinement is the kill signal** — jumps to +50 1-2 turns before victory.

**Synthesis:** More search depth to foresee territory flips is the next lever. Led to Iter 26.

---

### Iteration 26 — Phase-Gate Bottleneck Detection

**Status:** DONE
**Depends on:** Iterations 23, 25

**Goal:** Skip Tarjan's AP detection in early game (`lateBlend < 0.1`) to reclaim eval cost as extra search depth.

**What was built:**
- Added `skipBottleneck bool` parameter to `VoronoiTerritory`
- `Evaluate` passes `lateBlend < 0.1` — skips bottleneck when board fill < 32%
- `EvaluateDetailed` and trace always run full Tarjan (diagnostic data)
- New benchmarks: `BenchmarkVoronoiNoBottleneck`, `BenchmarkEvaluateLateGame`

**Performance:**
- Voronoi: 2414ns → 1051ns without bottleneck (57% faster)
- Early-game Evaluate: ~2450ns → ~1116ns (54% faster)
- Late-game Evaluate unchanged (Tarjan still runs when lateBlend ≥ 0.1)

**Risk:** Minimal. At `lateBlend=0.1`, bottleneck weight is only 0.15 — contributes at most ~1.65 eval points. Well below noise floor.

**Result:** 67% vs v24 (N=100). ~316 avg turns. Strongest single-iteration improvement since Iter 8 (88%).

---

### Iteration 25 (original) — Territory Shape Quality ❌ SUPERSEDED

**Status:** SUPERSEDED by Iteration 23
**Depends on:** Iteration 19

Original plan: detect corridor-shaped territory via thin-cell counting (cells with ≤1 owned neighbor). Iter 23's Tarjan AP detection captures the dangerous case directly — corridor territory that can be severed by opponent moves. Thin-cell counting would add a softer, redundant version of the same signal at additional eval cost. Skipped.

---

### Iteration 27 — Wall-Only Move Pruning in BRS

**Status:** DONE
**Depends on:** Iteration 26

**Goal:** Reduce effective branching factor by pruning guaranteed-death moves in `brsMax`/`brsMin`.

**What was built:**
- `wallSafeMoves(head, w, h, ordered)` — filters moves that go off-board (wall collision only)
- Applied in both `brsMax` (our moves) and `brsMin` (opponent moves)
- Fallback: if all moves hit walls, keep all 4 (needed for corner death scoring)
- `searchContext.nodes` counter + `GameSim.LastNodeCount` — search instrumentation
- `BenchmarkBestMoveIterativeDepth` / `BenchmarkBestMoveIterativeNodes` — resource consumption benchmarks
- `TestNodeCount` — verify instrumentation

**Key discovery:** Full `isSafeDir` pruning (wall + body) scored 32% vs v26 — a severe regression. Body-collision filtering is unsound because `isSafeDir` checks the *current* board statically but tails move on the next turn. Pruning an opponent move that appears to hit a tail (but won't after the tail retracts) removes the opponent's actual best response, causing the engine to overestimate positions. Wall-only pruning is sound because walls never move.

**Result:** 62% vs v26 (N=100). ~329 avg turns.

---

### Iteration 27 (dead end) — Full Unsafe Move Pruning ❌

**Status:** DEAD END
**Result:** 32% vs v26 (N=100)

Full `isSafeDir` pruning (walls + body segments) in both `brsMax` and `brsMin`. Failed because `isSafeDir` is a static check — it doesn't account for tail retraction. Removing an opponent body-collision move that is actually safe (tail will vacate) removes the min's best option, making us overestimate positions. Wall-only subset succeeded (62%).

---

### Iteration 28 — Tail-Aware Safe Move Check

**Status:** DONE
**Depends on:** Iteration 27

**Goal:** Make `isSafeDir` account for tail retraction to improve eval accuracy (confinement scoring).

**What was built:**
- `tailStacked(s)` — returns true if snake's tail is doubled (won't retract after eating)
- `foodAdjacentToHead(g, s)` — returns true if food within Manhattan distance 1 of head (may eat → tail stays)
- Replaced `isSafeDir` body-segment loop: skips tail segment (`Body[len-1]`) when tail will retract (not stacked, no food adjacent to that snake's head). Conservative: food near head = assume eating = tail stays. Sound: never marks a truly unsafe move as safe.
- Works for both self-chase (own tail retracts) and opponent body checks

**Key discovery:** Tail-aware `isSafeDir` improves eval accuracy (61% vs v27), but using it for BRS move pruning is too aggressive (43% vs v27). The eval benefit comes from removing false +50/+15 confinement bonuses when opponents can actually escape via retracting tails. The pruning failure suggests that even with correct tail-awareness, body-collision pruning removes strategically important moves from BRS — the search needs to consider "bad" moves to find the best response. Wall-only pruning in BRS remains the right approach.

**Result:** 61% vs v27 (N=100). ~436 avg turns. BRS pruning unchanged (wall-only).

---

### Iteration 28 (dead end) — Tail-Aware Body Pruning in BRS ❌

**Status:** DEAD END
**Result:** 43% vs v27 (N=100)

Used tail-aware `isSafeDir` for BRS move pruning (replacing `wallSafeMoves` with `safeMoves` that calls `isSafeDir`). Even with correct tail-retraction logic, pruning body-collision moves from BRS is harmful — the search benefits from exploring "unsafe" moves to find optimal responses. Eval-only tail awareness succeeded (61%).

---

### Iteration 30 — Remove Bottleneck Signal + Phase-Dependent Confinement

**Goal:** Data-driven improvement based on v28 trace analysis.

**Trace analysis findings (20 games, 120 perspectives):**
- Deaths: collision=34 (57%), body-collision=8 (13%), wall-collision=7 (12%), starvation=6 (10%), head-collision=5 (8%)
- Death phase: early=2, mid=16, late=42 (70% late game)
- Win types: 100% territory-squeeze (unchanged from Iter 25)
- StarvationRisk signal: 0.0 for both wins and losses (completely dead — condition too strict)
- **Bottleneck signal anti-correlated**: -0.09 for wins, +0.09 for losses (wrong direction)
- Territory dominates eval (90%+ of variance)
- OppConfinement/SelfConfinement differentiate wins from losses

**What was changed:**
1. **Removed bottleneck signal** — always skip Tarjan AP in Voronoi (`skipBottleneck=true`). The anti-correlation occurs because aggressive squeezers push into narrow corridors (creating fragile territory for themselves) to cut off opponents. The signal penalized the winning playstyle.
2. **Phase-dependent confinement weights** — confinement bonuses/penalties now scale with `lateBlend`:
   - OppConfinement 0 moves: 50 + 25×late (was 50 flat)
   - OppConfinement 1 move: 15 + 10×late (was 15 flat)
   - SelfConfinement 0 moves: -(25 + 25×late) (was -25 flat)
   - SelfConfinement 1 move: -(5 + 10×late) (was -5 flat)
   - Rationale: 70% of deaths are late-game, and confinement is the kill signal.

**Result:** 61% vs v28 (N=100). ~433 avg turns. Bottleneck-only removal: 53%; both changes together: 61%.

---

## Iter 31 — Eval Diet: Strip Dead Signals + Unused Voronoi Fields

**Goal:** Strip dead eval signals and unused Voronoi fields to recover search depth. First iteration with multi-opponent validation gate (must beat both v17 >55% and v28 >55%).

**What was done:**
1. **Removed 3 dead signals from `Evaluate()`** — FoodReach, FoodDenial, StarvationRisk (all δ < 0.05 between wins/losses, zero contribution in trace data).
2. **Stripped 6 unused `VoronoiResult` fields** — `MyTerritoryDepth`, `MyCenterX/Y`, `OppCenterX/Y`, `MyTailReachable`, `MyClosestFoodDist`, `OppClosestFoodDist`. Removed their computation from `VoronoiTerritory()`.
3. **Kept TailChase** — initially removed (plan classified it as dead, δ=0.00), but A/B testing showed removing it hurt significantly (44% vs v17). Restored — it provides meaningful late-game survival guidance despite low average contribution.
4. **Updated tracing infrastructure** — `EvalBreakdown`, `traceRecord`, `cmd/analyze/main.go` all slimmed to match.

**Key insight:** TailChase's trace analysis (δ=0.00) was misleading — the signal has low *average* contribution but high *conditional* importance in late-game confinement scenarios. Low average δ between wins and losses doesn't mean a signal is dead; it may be critical in specific situations that determine outcomes. Always A/B test signal removal; don't rely solely on aggregate trace statistics.

**Benchmark improvements:**
- Evaluate (late game): 194.6ns (was ~244ns, **20% faster**)
- Evaluate (early game): 1138ns (was ~1163ns, ~2% faster)
- Voronoi: 1055ns (was ~1090ns, ~3% faster)
- BRS node: 1199ns (was ~1233ns, ~3% faster)

**Result:** 55% vs v17 (N=100), 57% vs v28 (N=100), 52% vs v30 (N=100). ~442 avg turns. First version to pass multi-opponent validation.

---

## Iter 32 — Territory Quality: Connectivity Signal

**Goal:** Add territory connectivity signal to penalize narrow corridor territory and reward wide-open territory. Late-game losses follow a universal pattern: gradual territory erosion into narrow corridors → self-confinement → wall/collision death. Current eval counts all territory cells equally.

**What was done:**
1. **Added `MyConnectivity`/`OppConnectivity` to `VoronoiResult`** — computed in existing territory counting loop. For each owned cell, count how many of its 4 neighbors have the same owner; divide by territory count for average connectivity. Wide-open territory ~3.0-3.5, narrow corridor ~2.0, dead-end ~1.5.
2. **Added connectivity signal to `Evaluate()`** — `score += 5.0 * lateBlend * vr.MyConnectivity`. Uses absolute value (not delta) because absolute connectivity teaches the engine to prefer wide positions regardless of opponent. Scaled with lateBlend since territory quality matters most in crowded late game.
3. **Updated `EvalBreakdown`/`EvaluateDetailed()`** and **trace infrastructure** — `Connectivity` field in breakdown, `MyConnectivity`/`OppConnectivity` raw values in trace records.

**Key insight:** Delta-based connectivity (`MyConnectivity - OppConnectivity`) was neutral in self-play (50-52%) — symmetric like Iter 22's aggression modulation. Absolute MyConnectivity (rewarding our own wide territory regardless of opponent) works because it changes *move preferences* — the engine now prefers wide-open positions over narrow corridors at all depths. This is fundamentally different from symmetric delta signals.

**Weight tuning:**
- wConnectivity=5.0 (delta): 52%, 50% — neutral
- wConnectivity=10.0 (delta): 50% — still neutral
- wConnectivity=5.0 (absolute): **56%, 61%** — strong
- wConnectivity=8.0 (absolute): 49% — too strong, interferes with territory count

**Benchmark:** Evaluate early game: ~1270ns (+132ns), late game: ~208ns (+13ns). Zero allocs maintained. Well within <200ns budget.

**Result:** 56% and 61% vs v31 (N=100 each). ~443-451 avg turns.

---

## Iter 35 — Tail Reachability / Loopability Diagnostic (Dead End)

**Status:** ❌ DEAD END — signals are lagging indicators, not predictive

**Goal:** Instrument trace-only diagnostic signals (tail reachability, loopability, turns-to-death estimate) to evaluate whether they predict death. No eval or search changes — all code runs only when `HARUKO_TRACE=1`.

**What was built:**
1. `voronoiTerritoryImpl` — extracted core Voronoi BFS so `VoronoiTerritoryWithOwner` can copy the owner array without duplicating 200 lines
2. `VoronoiTerritoryWithOwner(g, myIdx, skipBottleneck, ownerOut)` — copies per-cell owner tags to caller buffer
3. `BFSTailDist(g, snakeIdx, owner, w, h)` — zero-alloc BFS from head to tail through owned territory + body cells
4. `MaxBoardCells` exported constant
5. 7 new trace fields: `MyTailReachable`, `MyTailBFSDist`, `OppTailReachable`, `OppTailBFSDist`, `MyLoopable`, `OppLoopable`, `TurnsToDeathEst`
6. `modeSurvival` analyze mode with 4 analysis sections

**Analysis results (100 self-play games, N=200 perspectives):**

| Finding | Detail |
|---------|--------|
| Tail always reachable | 100% of all turns, wins and losses — zero discriminative value |
| Loopability is a lagging indicator | 48% at 1 turn before death, 19% at 5 turns, 6% at 20 turns, 1% at 50 turns |
| Deaths are instantaneous | Territory collapses from 20-50 to 1 in a single turn (90% of non-starvation deaths) |
| Loopability asymmetry is symmetric | mine-only 3.9% wins / 2.4% losses — mirrored, small effect |
| TurnsToDeathEst is inaccurate | MAE=36 turns, bias=-27.3 (massive underestimate) |
| Starvation deaths are different | 10% of deaths — loopable=true, territory=58-86, just ran out of health |

**Why it failed:**

1. **Deaths are sudden, not gradual.** The dominant death pattern (90% of non-starvation losses): everything looks fine, then in 1 turn territory collapses from 20-50 to 1 and the snake is dead next turn. The opponent closes the last corridor exit in a single move.

2. **The collapse happens beyond BRS horizon.** BRS depth 14 (~7 full turns) cannot foresee the squeeze. The opponent's corridor-closing move is decided many turns ahead, and the 1-turn collapse is the result, not the cause.

3. **Tail reachability is trivially satisfied.** BFS through owned territory + body cells always finds a path. The body itself provides a highway from head to tail. The signal never fires false.

4. **Loopability is a consequence, not a cause.** It drops because territory collapses, not the other way around. By the time loopability goes false, the snake is already dead in 1-2 turns.

5. **TurnsToDeathEst (escape reachability) doesn't correlate with actual survival time.** Escape routes measure local space, not long-term viability.

**Key lesson:** The hypothesis "if a snake can't reach its tail, it's trapped" is not supported. The snake can almost always reach its tail. The real problem is that deaths are 1-turn territory collapses beyond search horizon. Survival signals need to predict the collapse BEFORE it happens, not detect it after. Possible approaches: territory trend detection, opponent corridor proximity, or longer-horizon rollout simulation (see Iter 36).

**Infrastructure retained:**
- `voronoiTerritoryImpl` refactor (cleaner code separation, enables future owner-array access)
- `VoronoiTerritoryWithOwner` function (zero-alloc owner array access for diagnostic use)
- `BFSTailDist` function (zero-alloc BFS, useful for future survival mode if needed)
- `MaxBoardCells` exported constant
- `modeSurvival` analyze mode (4-section tail/loop/death analysis)
- 7 trace fields for future diagnostic use

---

## Iter 36 — MC Rollout with Random Play (Dead End — Random Policy Too Noisy)

**Status:** ❌ DEAD END (random policy) — 32–52% across 22 configurations

**Goal:** Run random game rollouts from root position to detect which direction leads to long-term death. Computed once per move after BRS completes (separate budget, no BRS depth loss).

### What was implemented

- `logic/rollout.go`: `MCRolloutTimed(g, myIdx, oppIdx, maxTurns, budget)` — timed round-robin rollouts across valid root directions, `randomSafeMove` (isSafeDir + wall fallback), xorshift64 PRNG
- `logic/rollout.go`: `applyMCBias(result, mc, lateBlend)` — phase-scaled BRS score adjustment using MC survival spread
- `logic/search.go`: `BestMoveIterative` wired to run MC after BRS, apply bias
- `main.go`: Adaptive budget system, env var tuning (MC_WEIGHT, MC_SPREAD, MC_GATE, MC_TURNS)
- `trace.go`: Diagnostic MC rollouts (30ms, outside game budget) with per-direction survival rates
- Throughput: ~4000-6000 rollouts in 20-50ms budget (~100 turns each)

### Parameter sweep results

**Phase 1 (N=50 vs v32):**

| Config | Win% | Notes |
|--------|------|-------|
| mc-off (w=0) | 40% | Baseline (noise at N=50) |
| w=10, s=0.10 | 50% | |
| w=15, s=0.10 | 44% | Original setting |
| w=20, s=0.10 | 44% | |
| w=25, s=0.10 | 38% | Too aggressive |
| w=30, s=0.10 | 38% | Too aggressive |
| w=15, s=0.05 | 50% | |
| w=20, s=0.05 | **56%** | Best N=50 — but noise |
| w=15, gate=0.3 | 32% | Late-only MC terrible |
| w=15, turns=50 | 40% | Too few rollout turns |
| w=15, turns=200 | 42% | Too few samples |

**Phase 2 (N=100 vs v32) — confirming top configs:**

| Config | Win% |
|--------|------|
| w=20, s=0.05 | **43%** (N=50 "56%" was noise) |
| w=18, s=0.05 | 45% |
| w=22, s=0.05 | 45% |
| w=20, s=0.03 | 44% |
| w=20, s=0.07 | 52% |
| w=20, s=0.05, gate=0.05 | 44% |
| w=20, s=0.05, turns=75 | 43% |

**No configuration reliably beats v32 at N=100.**

### Why random rollouts fail

**Same root cause as Iter 29 (MCTS):** random opponents don't model how deaths actually happen.

1. **MC favors conservative play.** Random survival is highest in open space. MC penalizes aggressive squeezing — but squeezing is the winning strategy. Same failure mode as Iter 33 (defensive eval signals: 37-48%).

2. **MC is extremely noisy.** Trace analysis (10 games, 3642 turns):
   - MC disagrees with BRS on **44% of all turns** (early game: 52%, late: 33%)
   - Median survival spread: 0.078 — barely above activation thresholds
   - BRS picks MC's "worst" direction 28% of the time — but BRS is right (tactical depth)

3. **Random play can't model territory collapse.** Deaths happen when the optimal opponent closes a corridor exit in one move. Random opponents do this by accident ~5% of the time — MC survival rates converge to ~50-70% in all directions, drowning the signal.

### Infrastructure retained

- `logic/rollout.go`: `MCRolloutTimed`, `RolloutStats`, `MCRolloutResult`, `applyMCBias`, `randomSafeMove`, tunable params (MC_WEIGHT, MC_SPREAD, MC_GATE, MC_TURNS)
- `logic/search.go`: `BestMoveIterative` with BRS+MC pipeline, `BRSResult`
- `main.go`: Adaptive budget system, env var MC tuning
- `trace.go`: Diagnostic MC rollout per-direction survival in trace output
- `scripts/mc_sweep.sh`, `scripts/mc_sweep2.sh`: parameter sweep infrastructure

---

## Iter 36b — MC Rollout v2: Smart Policy + Top-2 Focused (Dead End)

**Status:** ❌ DEAD END — 47% vs v32 (N=100), 0 MC overrides in 7504 traced turns

**Goal:** Fix Iter 36's random rollout policy with (1) territory-aware heuristic (`smartRolloutMove`: flood-count BFS + chase/flee + food-seek, ~1730ns/move) and (2) focus MC budget on BRS top-2 directions only (skip when gap ≥ 5).

**What was implemented:**
- `quickFloodCount(g, start, 12)`: zero-alloc BFS reachable cell count (~655ns, 0 allocs)
- `smartRolloutMove(g, snakeIdx, oppIdx, rng)`: layered policy — safe moves → food-seek (health<25) → flood count → chase/flee bias, 20% random exploration (~1730ns, 0 allocs)
- `MCRolloutTop2Timed(g, myIdx, oppIdx, brsResult, maxTurns, budget, scoreGap)`: MC on BRS top-2 only, skips if gap ≥ scoreGap
- Env vars: MC_GAP, MC_SMART for A/B sweep

**Result:** 47% vs v32 (N=100)

**Deep trace analysis (20 games, 7504 turns):**

1. **Phase gate kills 98% of MC decisions.** `lateBlend ≥ 0.1` occurs in only 1.2% of turns (88/7504). MC fires 2220 times but 2176 (98%) are phase-gated — `applyMCBias` returns BRS unchanged.

2. **When phase gate passes, spread too low.** Only 44 turns have `lateBlend ≥ 0.1` AND MC fired. Of those, only 4 also have spread ≥ 0.10.

3. **MC adjustment too weak to override.** Max MC adjustment ever: 1.47 points. BRS gaps: 0.0-1.5. Zero overrides in 7504 turns — MC is structurally unable to change any decision.

4. **Removing phase gate: MC is anti-predictive.** Simulation across all turns:
   - MC disagrees with BRS ~47% of turns (coin flip)
   - Loss rate when ignoring MC: 50.8% at all spreads (noise)
   - At high confidence (spread ≥ 0.15, gap < 2): loss=41.5% — **following BRS wins more when MC disagrees most**
   - Winners have LOWER MC survival (0.396) than losers (0.447)

5. **Smart policy = same wrong signal as random.** Territory-aware heuristic still optimizes survival, and survival is anti-correlated with winning. Aggressive squeezing (the winning strategy) gets penalized by both random and smart rollout policies.

6. **Rollout volume too low.** Smart policy at 1730ns/move → 145 rollouts/dir in 50ms (StdErr=0.042). Need 400+ for statistical significance at spread=0.10.

**RL as rollout policy — infeasible:**
- Current 2.1M CNN: ~500μs/inference → 0.5 rollouts/dir in 50ms
- Tiny MLP (5K params): fast enough (~1-3μs) but can't learn territorial strategy
- Survival RL: learns same anti-predictive signal (prefer open space = conservative = loses)
- Competitive RL: 0% vs BRS at 2.1M params; 5K-param version dramatically worse
- For useful RL rollouts: need ≤625ns/move AND correct signal — mutually exclusive

**Key lesson:** MC rollouts are a closed dead end for this engine. The survival signal is fundamentally anti-correlated with winning in self-play. No rollout policy (random, heuristic, or RL) can fix this because the signal itself points the wrong way. BRS at depth 14 already captures everything within its horizon; rollouts beyond that horizon with approximate play add noise, not information. Future improvement paths: eval distillation from RL, RL single-eval tiebreaker (ONNX in Go), survival mode for confined spaces, or adaptive time management.

**Infrastructure retained:** All infrastructure from Iter 36 remains unchanged. No new infrastructure added to codebase (all Iter 36b code was reverted).
