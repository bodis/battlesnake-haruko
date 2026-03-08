# Haruko Battlesnake — Development Roadmap

> Active development plan. Completed iterations are archived in [ROADMAP_FINISHED.md](ROADMAP_FINISHED.md).
> Each iteration: implement → test → snapshot → compare → merge → move to finished → update ENGINE.md.
> Development follows a data-driven loop: trace games → analyze outcomes → identify root causes → design targeted fixes → verify with A/B comparison.

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iterations 1-20, 23-28, 30 (see ROADMAP_FINISHED.md) |
| **Dead ends** | Iter 21 (positional quality), Iter 22 (aggression), Iter 27 partial (full isSafeDir pruning: 32%), Iter 28 partial (tail-aware BRS pruning: 43%), Iter 29 (hybrid BRS+MCTS: 2–46%) |
| **Next** | Iter 31 — Cross-version regression analysis + eval diet |
| **Current** | v30 Remove bottleneck + phase-dependent confinement; 61% vs v28 (N=100); ~433 avg turns |
| **Key insight** | v30 loses to v17 (49%) despite 13 iterations of improvements. Hill-climbing against previous version doesn't guarantee overall strength. Eval complexity costs search depth. |

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

## Iter 31 — Cross-Version Regression Analysis + Eval Diet

**Goal:** Understand why v30 (13 iterations of improvements) can't beat v17, and build a leaner eval that's stronger against diverse opponents — not just the previous version.

### Problem: Hill-Climbing Trap

Cross-version testing revealed that iterative improvements against the previous version haven't translated to overall strength:

| v30 vs | Iter | Gap | v30 Win% | N |
|--------|------|-----|----------|---|
| v7 (Voronoi eval) | 7 | 23 versions | **94%** | 50 |
| v9 (iterative deepening) | 9 | 21 versions | **68%** | 50 |
| v11 (TT + Zobrist) | 11 | 19 versions | **56%** | 50 |
| v17 (phase-adaptive eval) | 17 | 13 versions | **49%** | 100 |
| v20 (food strategy) | 20 | 10 versions | **64%** | 50 |
| v28 (previous) | 28 | 2 versions | **61%** | 100 |

**v17 is tied with v30 despite having 13 fewer iterations.** And v20 is harder to beat (64%) than v28 (61%).

### Root Cause Hypothesis: Eval Bloat → Depth Loss

v17 has a radically simpler eval:
- **6 signals:** territory, food count (flat), length, H2H, opp confinement, food urgency
- **No:** food cluster, food reach, food denial, starvation risk, growth urgency, tail chase, self-confinement, bottleneck
- **Voronoi:** 5 result fields (161 lines) vs v30's 15+ fields (424 lines, though Tarjan now skipped)
- **safeMoveCount:** 1 call (opp only) vs v30's 2 calls (opp + self)
- **Weights:** lower (territory 1.0 vs 1.5, length 2.0 vs 3.0, H2H 5.0 vs 8.0)

The cheaper eval means v17 searches deeper within the same 300ms budget. Since 70% of deaths are sudden territory flips beyond the search horizon, **extra depth to foresee these flips may matter more than extra eval signals to score them**.

### Why v30 beats v28 but not v17

Each iteration A/B tests against the immediately previous version. Both sides share the same eval structure, so improvements in signal quality are visible: the better-calibrated eval scores positions more accurately, and BRS exploits this.

But against a **structurally different opponent** (v17), the game plays out differently:
- v17 searches deeper → sees territory flips earlier → avoids traps v30 walks into
- v17's simpler eval produces different move preferences → v30's signals are calibrated for v30-vs-v30 games, not v30-vs-v17 games
- The signals added in Iter 20-28 may be locally optimal (beat prev version) but globally harmful (lose depth against everyone else)

### Phase 1: Measure the Depth Gap

**Step 1: Benchmark v17 eval cost**
```bash
# Build v17 in a temp directory, run benchmarks
git stash && git checkout ea8e2d7 -- logic/eval.go logic/voronoi.go
go test ./logic -bench BenchmarkEvaluate -benchtime 5s
# Then restore
git checkout HEAD -- logic/eval.go logic/voronoi.go && git stash pop
```
This won't work cleanly (API differences), so instead:

**Step 1 (alternative): Add a v17-style "lean eval" benchmark**
Create `BenchmarkEvaluateLean` that mimics v17's eval cost structure — call `VoronoiTerritory` with minimal field computation, skip self-confinement, skip food denial/starvation/growth/tail signals. Compare ns/op to understand exactly how many nanoseconds we're spending on the added signals.

**Step 2: Measure actual search depth in games**
```bash
make trace N=20   # v30 traces already exist
```
Check depth-per-turn in v30 traces. Then run v17 with tracing to compare depth-per-turn.

**Step 3: Profile which eval components cost the most**
Add optional timing to `Evaluate()` — measure Voronoi call, safeMoveCount calls, and signal arithmetic separately. Run for 1000 evals to get stable averages.

### Phase 2: Signal Audit — Keep or Kill

For each signal added since v17, determine: does it provide enough eval accuracy to justify its depth cost?

| Signal | Added in | Iter 30 signal analysis | Verdict |
|--------|----------|------------------------|---------|
| Food cluster (distance-weighted) | Iter 20 | +0.39 wins / +0.34 losses (δ=0.05) | **Suspect** — barely differentiates |
| Food reach advantage | Iter 20 | +0.02 / -0.02 (δ=0.04) | **Kill** — near zero contribution |
| Food denial | Iter 20 | +0.00 / +0.00 (δ=0.00) | **Kill** — completely dead |
| Starvation risk | Iter 20 | -0.00 / -0.00 (δ=0.00) | **Kill** — completely dead |
| Growth urgency | Iter 20 | -0.65 / -0.56 (δ=0.09) | **Suspect** — moderate, but only early game |
| Self-confinement | Iter 8 (added), Iter 30 (phased) | -0.24 / -0.37 (δ=0.13) | **Keep** — differentiates wins/losses |
| Tail chase | Iter 24 | +0.01 / +0.01 (δ=0.00) | **Kill** — near zero contribution |
| Opp confinement (phase) | Iter 30 | +1.00 / +0.70 (δ=0.30) | **Keep** — strong differentiator |
| Bottleneck | Iter 23 (removed Iter 30) | already removed | — |

**Dead signals (zero contribution):** FoodDenial, StarvationRisk, TailChase, FoodReach
**Suspect signals (minimal contribution):** FoodCluster, GrowthUrgency
**Valuable signals (clear differentiation):** Territory, OppConfinement, SelfConfinement, LenAdvantage, H2H

### Phase 3: Build & Test Lean Eval

**Step 1: Create "v30-lean" — strip dead signals**
Remove FoodDenial, StarvationRisk, TailChase, FoodReach from `Evaluate()`. These contribute <0.05 eval points difference between wins and losses. Zero gameplay impact, pure cost savings.

**Step 2: Measure depth recovery**
Benchmark v30-lean vs v30. Even small savings (50-100ns) compound over millions of BRS nodes.

**Step 3: Consider stripping Voronoi fat**
v30's `VoronoiTerritory` computes `MyFoodValue`, `MyClosestFoodDist`, `OppClosestFoodDist`, `MyTerritoryDepth`, `MyCenterX/Y`, `OppCenterX/Y`, `MyTailReachable` — but many of these are only used by the dead signals. If we kill FoodReach, we don't need `MyClosestFoodDist`/`OppClosestFoodDist`. If we kill FoodCluster, we don't need `MyFoodValue`. Strip unused Voronoi fields → faster BFS.

**Step 4: A/B test the lean eval**
```bash
make snapshot                                    # capture v30
# implement lean eval
make compare PREV=snapshots/haruko-<v30> N=100   # vs v30
make compare PREV=snapshots/haruko-ea8e2d7 N=100 # vs v17
make compare PREV=snapshots/haruko-a989fbb N=100 # vs v20
```

**Target:** Beat v17 (>55%) AND beat v28 (>55%) simultaneously. This proves the lean eval is stronger overall, not just against one opponent.

### Phase 4: Weight Recalibration on Lean Eval

After stripping dead signals, the remaining weights (territory, length, H2H, confinement) may need adjustment. The weight ratios change when signals are removed.

**Method:** Vary each remaining weight ±50%, test at N=50 against v17 (not v30 — this breaks the hill-climbing trap).

### Phase 5: Validate Against Multiple Opponents

Final validation must pass ALL of:
```bash
make compare PREV=snapshots/haruko-ea8e2d7 N=100  # vs v17: must be >55%
make compare PREV=snapshots/haruko-a989fbb N=100  # vs v20: must be >60%
make compare PREV=snapshots/haruko-c77fa1f N=100  # vs v28/v30: must be >55%
```

If any test fails, the change is rejected. This ensures we're building a **generally stronger** player, not one that beats a specific opponent.

### Critical Files
- `logic/eval.go` — strip dead signals, slim down Evaluate()
- `logic/voronoi.go` — remove unused VoronoiResult fields and their computation
- `logic/bench_test.go` — add lean eval benchmark, depth comparison

### Constraints
- Do NOT change search mechanics (BRS, TT, killers, wall pruning all stay)
- Do NOT add new signals — this iteration is about REMOVING complexity
- Do NOT change the Voronoi BFS algorithm — only strip unused result fields
- Keep `EvaluateDetailed` and `VoronoiResult` struct fields for tracing (can skip computation when `skipBottleneck=true` already gates Tarjan)
- isSafeDir tail-awareness stays (it's in the eval hot path and proven at 61%)

### Success Criteria
v30-lean must beat BOTH v17 (>55%) and v28 (>55%) at N=100. This is the first iteration with a multi-opponent validation gate.

---

## Future Directions (After Iter 31)

**Split bottleneck into offensive/defensive:**
Iter 30 removed bottleneck entirely. Concept may work as offensive-only: `OppThreatenedTerritory` reward with no defensive penalty.

**Weight recalibration on lean eval:**
If Iter 31 strips signals, remaining weights need recalibration against v17 (not self-play).

**Fix starvation risk (if kept):**
Condition `MyFood==0 && Health<50` too strict. Better: health-to-food-distance ratio.

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
