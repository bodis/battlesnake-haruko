# Haruko Battlesnake — Development Roadmap

> Active development plan. Completed iterations are archived in [ROADMAP_FINISHED.md](ROADMAP_FINISHED.md).
> Each iteration: implement → test → snapshot → compare → merge → move to finished → update ENGINE.md.
> Development follows a data-driven loop: trace games → analyze outcomes → identify root causes → design targeted fixes → verify with A/B comparison.

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iterations 1-20, 23-28, 30-35 (see below + ROADMAP_FINISHED.md) |
| **Dead ends** | Iter 21 (positional quality), Iter 22 (aggression), Iter 27 partial (full isSafeDir pruning: 32%), Iter 28 partial (tail-aware BRS pruning: 43%), Iter 29 (hybrid BRS+MCTS: 2–46%), Iter 33 (escape/territory eval signals: 37–54%), Iter 34 (bottleneck routing: 40-57%, depth regression), Iter 35 (tail reachability/loopability: signals too late, 1-turn death collapse) |
| **Current** | v32 Territory connectivity: absolute MyConnectivity signal; 56-61% vs v31 (N=100); ~443-451 avg turns |
| **Next** | Iter 36 (MC strategic rollout, diagnostic) → Iter 37 (survival mode) |
| **Key insight** | Deaths are instantaneous 1-turn territory collapses beyond BRS horizon. Survival signals (tail reachability, loopability) are lagging indicators — they detect collapse after it happens, not before. Need longer-horizon methods (MC rollouts, territory projection) to predict collapse before it occurs. |

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

## Iter 36 — Monte Carlo Strategic Rollout (Root-Level, Diagnostic)

**Status:** PLANNED

**Goal:** Run cheap random game rollouts from the root position to detect which initial direction pairs lead to long-term death. Computed once per move alongside BRS. **Phase A is diagnostic only (trace + analyze). Phase B (wiring into BRS) only happens if Phase A shows clear value.**

**Why:** BRS sees 12-14 plies (~6-7 full turns). Deaths are decided by territory flips 20-50+ turns ahead. Random rollouts are tactically stupid but can detect **structural** long-term problems — like "going left leads to a partition where I have 30 cells and die in 40 turns."

### Design

**Step 1: Generate initial direction pairs**

From current position, enumerate all valid 2-ply openings for us:
- Ply 1: our valid moves (exclude wall collisions) → typically 3 directions
- Ply 2: from each ply-1 result, our valid moves → typically 2-3 directions
- Result: 6-9 direction pairs, minus wall overlaps → typically **6-8 valid pairs**
- Opponent plays random valid move at each setup ply

**Step 2: Distribute rollouts evenly**

Given total rollout budget N (e.g., 1000):
- Divide evenly across all valid direction pairs: ~125-167 rollouts each
- Each rollout: from the 2-ply position, both sides play random valid moves (uniform from non-wall moves) until game over or max turns (200)
- Random move selection: xorshift64 PRNG (reuse from MCTS infra in `logic/mcts.go`)

**Step 3: Measure per direction pair**

For each `(dir1, dir2)` pair, record:
- **Survival rate**: % of rollouts where we're alive at the end
- **Avg turns survived**: mean game length across rollouts
- **Territory at death**: average territory when we die

**Step 4: Phase A — Log to trace (diagnostic only)**

Log per-direction survival rates to trace. Analyze:
- Does the worst direction pair correlate with BRS's chosen move being wrong?
- How far ahead of actual death does MC detect the bad direction?
- Does MC agree with BRS most of the time, or does it add genuinely new information?
- When MC disagrees with BRS AND game is eventually lost, was MC right?

**Step 5: Phase B — Wire as BRS signal (only if Phase A shows value)**

If data shows MC rollouts detect real danger:
- Collapse pairs to per-direction survival: `dir1_survival = avg of all pairs starting with dir1`
- Pass to BRS as root-level bias: penalize directions with low survival rate
- Weight TBD based on Phase A data

### Implementation

| File | Change |
|------|--------|
| `logic/rollout.go` | New file: `MCRollout(g, myIdx, oppIdx, totalRollouts, maxTurns) map[Direction]RolloutStats`. Generates direction pairs, distributes rollouts, runs random games using `CloneFromPool`/`Release` + `Step`. |
| `logic/rollout.go` | `randomValidMove(g, snakeIdx)` — uniform random from non-wall moves, xorshift64 PRNG |
| `logic/rollout.go` | `RolloutStats` struct: `SurvivalRate float64`, `AvgTurns float64`, `Rollouts int` |
| `trace.go` | Add per-direction rollout fields: `MCUp`, `MCDown`, `MCLeft`, `MCRight` (survival rate 0.0-1.0), `MCBestDir`, `MCWorstDir` |
| `trace.go` | Call `MCRollout` in `traceTurn()` when trace enabled (outside game budget) |
| `cmd/analyze/main.go` | New analyze mode: `rollout` — compare MC survival rates vs BRS chosen direction vs game outcome. Report: (1) MC-BRS agreement rate, (2) when MC disagrees, who was right, (3) MC worst-direction as death predictor |

### Cost budget

| Component | Cost |
|-----------|------|
| Per rollout step | `CloneFromPool` (19ns) + `Step` (49ns) + random move (~5ns) ≈ 73ns |
| Per rollout (200 turns) | ~15μs |
| 1000 rollouts total | ~15ms |
| **Trace-only (Phase A)** | 15ms, computed outside game budget — fine |
| **If wired later (Phase B)** | ~30-50ms carved from 300ms budget → ~2000 rollouts → ~250/direction pair |

### Testing plan

```
make trace N=50
make analyze MODE=rollout
```

**Success criteria for Phase A:**
- MC worst-direction survival rate < 30% predicts actual death within 50 turns
- MC direction ranking disagrees with BRS choice on > 10% of turns (otherwise redundant)
- When MC disagrees with BRS AND game is lost, MC was right > 60% of the time

---

## Iter 37 — Survival Mode: Longest Path in Confined Space

**Status:** PLANNED (deferred, depends on Iter 35 data)

**Goal:** When partitioned or confined, switch from BRS to a space-filling survival algorithm. BRS simulates irrelevant opponent moves in partitions. A longest-path algorithm directly optimizes survival.

**Trigger:** `vr.IsPartitioned == true` or `EscapeReachabilityPooled(g, myIdx, 6) < 20`

**Approach:** Exhaustive DFS for small partitions (N < 30, microseconds), greedy flood-fill + tail-chasing for larger (N = 30-60). Tail-aware: account for tail retraction opening cells behind us.

**Files:** `logic/survival.go` (new), `logic/search.go` (partition bypass)

---

## Iter 38 — Adaptive Time Management

**Status:** PLANNED (lower priority)

**Goal:** Allocate more search time in critical positions (partitioned, low escape, near-death) and less in stable positions (open board, high territory). Currently every move gets 300ms.

- Critical position: up to 450ms
- Stable position: down to 200ms
- Net budget per game remains similar

**Files:** `main.go` (dynamic budget calculation)

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

**Long-horizon alternatives (if MC rollout doesn't pan out):**
- **Simplified territory projection**: strip board to heads + territory boundaries, simulate territory evolution without full game rules. No body tracking, no food — just "who controls what space."
- **Voronoi trajectory simulation**: project Voronoi boundary forward assuming both snakes move toward territory center. Detect convergence toward partition.
- **Root-only Tarjan's**: compute AP/bottleneck once at root (not per leaf), use as strategic bias. Avoids the depth regression that killed Iter 34.

**Weight recalibration:**
After structural changes, all weights may need recalibration. Test against v17.

**CorridorRatio / FunnelRatio:**
Data shows these are lagging indicators (only 4-12% detectable at search leaf). NOT viable as eval signals. But could inform survival mode trigger or time management.

**Tail reachability as eval signal: ❌ ruled out by Iter 35 data.**
Tail is always reachable (100% of turns). Loopability is a lagging indicator (fires 1-2 turns before death). Not viable as eval signal.

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
| 35 | — | — | ❌ Dead end: tail reachability/loopability (signals too late) |
