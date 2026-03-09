# Haruko Battlesnake — Development Roadmap

> Active development plan. Completed iterations are archived in [ROADMAP_FINISHED.md](ROADMAP_FINISHED.md).
> Each iteration: implement → test → snapshot → compare → merge → move to finished → update ENGINE.md.
> Development follows a data-driven loop: trace games → analyze outcomes → identify root causes → design targeted fixes → verify with A/B comparison.

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iterations 1-20, 23-28, 30-34 (see below + ROADMAP_FINISHED.md) |
| **Dead ends** | Iter 21 (positional quality), Iter 22 (aggression), Iter 27 partial (full isSafeDir pruning: 32%), Iter 28 partial (tail-aware BRS pruning: 43%), Iter 29 (hybrid BRS+MCTS: 2–46%), Iter 33 (escape/territory eval signals: 37–54%), Iter 34 (bottleneck routing: 40-57%, depth regression) |
| **Current** | v32 Territory connectivity: absolute MyConnectivity signal; 56-61% vs v31 (N=100); ~443-451 avg turns |
| **Key insight** | Tarjan's AP is too expensive for hot-path eval (~2150ns/call). Any eval signal requiring Tarjan's causes depth regression that offsets the signal's benefit. Bottleneck detection must use cheaper proxies or operate outside the leaf evaluator. |

---

## Iter 33 — Escape Routes + Territory Depth Eval Signals (Dead End)

**Status:** ❌ DEAD END — 37–54% across 8 configurations

**Goal:** Add escape routes and far territory as eval signals to detect positional collapse.

**What was tried (A/B vs v32, N=100 each):**

| Config | Type | Weight | Threshold | Result |
|--------|------|--------|-----------|--------|
| Defensive escape + far territory | penalty + reward | wE=5.0, wF=1.0 | escape<25 | **37%** |
| Far territory only | reward | wF=0.5 | — | **46%** |
| Defensive escape only (gated) | penalty | wE=3.0 | escape<12 | **40%** |
| Defensive escape only (always) | penalty | wE=3.0 | escape<12 | **48%** |
| Offensive opponent squeeze | reward | w=3.0 | oppEscape<15 | **54%** |
| Offensive opponent squeeze | reward | w=5.0 | oppEscape<15 | **46%** |
| Offensive opponent squeeze | reward | w=4.0 | oppEscape<15 | **49%** |
| Offensive opponent squeeze | reward | w=3.0 | oppEscape<20 | **45%** |

**Why it failed:**

1. **Narrow corridors are normal in late game, not pathological.** On 11x11 after 250+ turns, corridors happen constantly. The diagnostic analysis cherry-picked LOSING cases and found correlations, but the same signals fire frequently in WINNING cases too. Reacting to the signal changes behavior in ALL cases — including the many where corridors are fine.

2. **Defensive signals cause conservative play.** Penalizing low escape routes makes the engine avoid aggressive squeezing positions. But squeezing is the winning strategy — winners have MORE fragile territory (same pattern as Iter 30's bottleneck signal). The v32 opponent doesn't have this penalty, squeezes freely, and wins.

3. **Offensive signals are weak or cancel.** Rewarding opponent squeeze (54% best case) provides marginal benefit at w=3.0 but overweighting (w=4.0-5.0) hurts. The signal adds cost (~500ns BFS per eval) for unreliable gain within statistical noise.

4. **Far territory is redundant with territory count.** Rewarding far territory (46%) double-counts what the territory signal already captures. More territory naturally means more far territory.

**Key lesson:** Correlation between a metric and outcomes does NOT mean the metric works as an eval signal. The metric detects a situation, but the right RESPONSE to that situation matters more than the detection. Narrow corridors need a different algorithm (routing, survival), not a score penalty.

**Infrastructure retained (no eval cost, useful for future iterations):**
- `VoronoiResult` depth profile fields: `MyNearTerritory`, `MyFarTerritory`, `OppNearTerritory`, `OppFarTerritory`, `MyCorridorCells`, `OppCorridorCells` — computed in existing territory loop, ~0 extra cost
- `EscapeReachabilityPooled(g, snakeIdx, maxDist)` — zero-alloc pooled BFS in `logic/voronoi.go`
- `EscapeReachability(g, snakeIdx, maxDist)` — allocating BFS in `logic/diagnostic.go`, trace-only
- Trace fields: escape routes, corridor/funnel ratios for both snakes
- `cmd/analyze`: `correlation` + `decision-points` modes

---

## Iter 34 — Bottleneck-Aware Routing (Dead End)

**Status:** ❌ DEAD END — 40-57% across 5 configurations (depth regression)

**Goal:** Detect which side of a territory bottleneck (articulation point) our head is on and guide it toward the larger region. Directional routing signal, not a general narrowness penalty.

**What was tried (A/B vs v32, N=100-200):**

| Config | Gate (lateBlend) | Weight | vs v32 | vs v28 |
|--------|-----------------|--------|--------|--------|
| gate=0.3, w=10 | ≥0.3 (~36% fill) | 10.0 | **40%** | **39%** |
| gate=0.5, w=10 | ≥0.5 (~40% fill) | 10.0 | **57%** (N=100) → **50.5%** (N=200) | **50%** |
| gate=0.7, w=10 | ≥0.7 (~44% fill) | 10.0 | **44%** | **53%** |
| gate=0.5, w=8 | ≥0.5 | 8.0 | **42%** | **48%** |

**Why it failed:**

1. **Tarjan's AP is too expensive for leaf evaluation.** Full Tarjan's + head-side BFS adds ~2150ns per Voronoi call (1055ns → 3200ns). At every leaf node in BRS, this causes 1-2 plies of depth regression. The depth loss overwhelms any routing benefit.

2. **Phase-gating doesn't fully solve cost.** Even at gate=0.5 (half the game), leaf evals in late-game still pay 3x cost. The gate=0.5/w=10 config appeared to work (57% N=100) but was noise — N=200 confirmed 50.5%.

3. **The signal concept is sound but the implementation path is wrong.** Bottleneck routing requires Tarjan's, which is only viable outside the hot-path leaf evaluator. Possible future approaches: root-only computation (once per move), or a cheaper AP proxy.

**Infrastructure retained (no eval cost):**
- `headSideFloodFill` method on voronoiWorkspace — zero-alloc BFS from head through non-AP territory cells
- `HeadSideRegion` field in VoronoiResult — cells reachable from head without crossing APs
- `headQueue`, `headFillDirty` arrays in voronoiWorkspace (pooled, 1.4KB)
- `BottleneckRoute` field in EvalBreakdown + trace record — diagnostic only
- All infrastructure is used only when `skipBottleneck=false` (currently always true in eval)

---

## Iter 35 — Survival Mode: Longest Path in Confined Space

**Status:** PLANNED

**Goal:** When partitioned (or very confined, e.g. escape routes < 20), switch from BRS minimax to a space-filling survival algorithm. In a partitioned board, the opponent can't reach us — the only question is who fills their space more efficiently. BRS is solving the wrong problem; a longest-path algorithm directly optimizes survival.

### Problem Analysis

When the board is partitioned, BRS simulates opponent moves that can't affect us. The eval compares territory that won't change. The engine effectively plays random moves in its partition. A space-filling algorithm would play optimally: follow the tail, avoid dead ends, maximize turns survived.

### Implementation Plan

**Step 1: Detect confined state**

Trigger conditions (any of):
- `vr.IsPartitioned == true` (board fully partitioned)
- `EscapeReachabilityPooled(g, myIdx, 6) < 20` (very confined, even if not fully partitioned)

**Step 2: Longest-path DFS**

When confined to N cells (N < ~30):
- Run exhaustive DFS from head through reachable cells
- Find the longest simple path (visits most cells without revisiting)
- With N ≈ 20, this is trivially fast — DFS with backtracking on a grid graph of 20 nodes takes microseconds
- Return the first move of the longest path

For larger confined areas (N = 30-60):
- Exhaustive DFS becomes too slow
- Use heuristic: prefer moves that maximize reachable cells after moving (greedy flood fill)
- Enhanced tail-chasing: strongly prefer moves toward own tail (creates a loop, maximizes survival)

**Step 3: Integration with BRS**

Options:
- **Replace BRS entirely** when partitioned — survival algorithm chooses the move directly
- **Use as eval** when partitioned — `score = survivable_turns` instead of territory-based eval
- Replacement is cleaner: BRS opponent modeling is wasted computation when partitioned

**Step 4: Tail-aware space filling**

Key insight: our tail moves as we move, opening cells behind us. So in a 20-cell partition, we can survive more than 20 turns if we follow our tail efficiently. The algorithm must account for tail movement when computing longest path.

### Testing Plan

- Focus on partition-specific metrics: compare survival turns in partitioned positions
- `make compare PREV=snapshots/haruko-69c43bb N=100` — vs v32. Must beat >55%.
- Trace analysis: verify the algorithm triggers in the right situations

### Files to Modify

| File | Change |
|------|--------|
| `logic/search.go` | Partition detection, survival mode bypass |
| `logic/survival.go` | New file: longest-path DFS, space-filling heuristic |
| `logic/eval.go` | Optional: survivable-turns eval for partition case |

---

## Iter 36 — Adaptive Time Management

**Status:** PLANNED (lower priority)

**Goal:** Allocate more search time in critical positions (partitioned, low escape, near-death) and less in stable positions (open board, high territory). Currently every move gets 300ms. Critical positions need deeper search; stable positions are decided by depth 8.

### Implementation Plan

- Base budget: 300ms
- Critical position (low escape, partitioned): up to 450ms
- Stable position (high territory, open board): down to 200ms
- Net budget per game remains similar

### Files to Modify

| File | Change |
|------|--------|
| `main.go` | Dynamic budget calculation based on position features |

---

## Iter 29 — Hybrid BRS+MCTS Root-Level Vote (Dead End)

**Status:** ❌ DEAD END — 2–46% across 6 configurations

**What was tried:**
- Flat depth-1 MCTS (UCB1 selection, random opponent, xorshift64 PRNG, ~52K sims/50ms)
- Sequential budget splits: 70/30 (2%), 95/5 (30% — pure budget loss)
- Tiebreaker-only: MCTS breaks exact BRS score ties (46%)
- Concurrent goroutines: data race on gameSimPool

**Why it failed:**
1. BRS depth is hypersensitive to budget — even 5% (15ms) reduction causes measurable depth regression
2. Depth-1 MCTS with random opponents systematically favors moves good against weak play
3. Even as tiebreaker on exact BRS ties, MCTS noise hurts (46%)
4. Concurrent BRS+MCTS blocked by data races on shared sync.Pool

**Infrastructure retained:** `BRSResult`, `bestMoveBRS()`, `mctsSearch()`, `logic/mcts.go`

---

## Future Directions

**Weight recalibration:**
After structural changes (bottleneck routing, survival mode), all weights may need recalibration. Test against v17.

**CorridorRatio / FunnelRatio:**
Data shows these are lagging indicators (only 4-12% detectable at search leaf). NOT viable as eval signals. But could inform survival mode trigger or time management.

---

## Snapshot Log (Active)

Continues from ROADMAP_FINISHED.md snapshot log.

| Iteration | Snapshot | Avg Turns | Notes |
|-----------|----------|-----------|-------|
| 19 | | | Voronoi strategic extraction (infra) |
| 20 | `snapshots/haruko-a989fbb` | ~443 | Food strategy signals; 54% vs v19 |
| 21 | — | — | ❌ Dead end (37–48%) |
| 22 | — | — | ❌ Dead end (42–49%) |
| 23 | `snapshots/haruko-0e6fdda` | ~287-350 | Territory bottleneck detection; 58% vs v20 |
| 24 | `snapshots/haruko-355c7d3` | ~337 | Weight calibration; 61% vs v23 |
| 25 | | | Win/loss trace analysis |
| 26 | `snapshots/haruko-fb7b3a1` | ~316 | Phase-gate bottleneck; 67% vs v24 |
| 27 | `snapshots/haruko-wallonly` | ~329 | Wall-only move pruning; 62% vs v26 |
| 28 | `snapshots/haruko-tailaware` | ~436 | Tail-aware isSafeDir; 61% vs v27 |
| 29 | — | — | ❌ Dead end: hybrid BRS+MCTS (2–46%) |
| 30 | `snapshots/haruko-c77fa1f` | ~433 | Remove bottleneck + phase confinement; 61% vs v28 |
| 31 | `snapshots/haruko-e7f195f` | ~442 | Eval diet: strip dead signals + Voronoi fields; 55% vs v17, 57% vs v28 |
| 32 | `snapshots/haruko-69c43bb` | ~443-451 | Territory connectivity (absolute MyConnectivity); 56-61% vs v31 |
| 34 | — | — | ❌ Dead end: bottleneck routing (40-57%, depth regression) |
