# Haruko Battlesnake — Development Roadmap

> Active development plan. Completed iterations are archived in [ROADMAP_FINISHED.md](ROADMAP_FINISHED.md).
> Each iteration: implement → test → snapshot → compare → merge → move to finished → update ENGINE.md.
> Development follows a data-driven loop: trace games → analyze outcomes → identify root causes → design targeted fixes → verify with A/B comparison.

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iterations 1-20, 23 (see ROADMAP_FINISHED.md) |
| **Dead ends** | Iter 21 (positional quality), Iter 22 (aggression), Iter 25 (superseded by 23) |
| **Next** | Iteration 24 |
| **Current** | v23 Territory bottleneck detection; BRS depth ~12-13; Evaluate ~2450ns/0 allocs; 58% vs v20 |
| **Key insight** | Eval quality > search depth. But eval cost doubled in Iter 23 — next steps should extract more value from existing signals (calibration) and reclaim eval budget (phase-gating). |

---

## Phase 9: Optimization & Calibration

> **The situation:** We have 13 eval signals, most with weights set by intuition. Iter 23 doubled eval cost
> for a strong 58% win. The next phase extracts maximum value from existing infrastructure before adding
> new signals. Three steps: calibrate weights, reclaim eval speed, then reassess with fresh trace data.

### Iteration 24 — Weight Calibration

**Status:** TODO
**Depends on:** Iterations 20, 23

**Goal:** Systematically tune all eval weights. Many were set by intuition. Iter 20 showed weights are highly sensitive (halving food strategy weights: 47% → 54%).

**Weights to tune (~13):**

| Category | Weights | Current |
|----------|---------|---------|
| Territory | wTerritory coefficients | `1.0 - 0.2×early + 0.3×late` |
| Length | wLen | `2.0 + 1.0×early - 0.5×late` |
| H2H pressure | wH2H | `5.0 - 2.0×late` |
| Confinement | opp / self | 50/15 and 25/5 |
| Tail chase | wTailChase | 3.0 |
| Food urgency | wFoodUrgency | 0.5 |
| Food cluster | wFoodCluster | `1.5 × early` |
| Food reach | wFoodReach | 0.5 |
| Food denial | wFoodDenial | 2.0 |
| Starvation risk | wStarvationRisk | 2.5 |
| Growth urgency | wGrowthUrgency | `0.3 × early` |
| Bottleneck | wBottleneck | `0.3 × (0.5 + 0.5×late)` |

**Approach:**
1. Start with bottleneck weight (never tuned, conservative 0.3 — try 0.5, 0.2)
2. Then food signals (known sensitive from Iter 20)
3. Then core signals (territory, length, H2H)
4. One weight at a time: 2× it, 0.5× it, compare N=50
5. If >55%: keep. If <50%: revert. If 50-55%: noise, skip.
6. After individual sweeps, test 2-3 combined adjustments
7. ~15-20 compare runs total

**Files:**
| File | Action |
|------|--------|
| `logic/eval.go` | Adjust weights based on A/B results |

**Verify:** Each change via `make compare N=50`. Final combined result via `make compare N=100`.

---

### Iteration 25 — Phase-Gate Bottleneck Detection

**Status:** TODO
**Depends on:** Iteration 24

**Goal:** Reclaim search depth by skipping Tarjan's AP detection when it adds little value. The bottleneck signal matters most in mid/late game when corridors form. In early game (open board, small snakes), territories are compact — bottlenecks don't exist yet.

**Approach:**
1. Skip both Tarjan calls when `lateBlend < 0.1` (roughly: board fill < 32%, early game)
2. Optionally skip opponent Tarjan when `OppTerritory < 12` (too small for meaningful bottleneck)
3. Measure eval cost savings: target ~1600-1800ns average (vs 2450ns current), reclaiming 1-2 search plies for early/mid game

**Expected impact:**
- Early game: eval drops to ~1100ns (no Tarjan), gaining ~2 extra search plies
- Late game: eval stays at ~2450ns (Tarjan runs), keeping bottleneck awareness
- Net effect: more depth when depth matters most (early positioning), same eval quality when eval matters most (late survival)

**Files:**
| File | Action |
|------|--------|
| `logic/voronoi.go` | Add lateBlend parameter or board-fill check before Tarjan calls |
| `logic/eval.go` | Pass phase info or let Voronoi check board state directly |

**Verify:** `go test -bench BenchmarkEvaluate` — target <1800ns average across game phases. `make compare N=50` — target >50% (should not hurt since early bottleneck signal was near-zero anyway).

---

### Iteration 26 — Trace Analysis & Late-Game Survival (Conditional)

**Status:** TODO (data-driven — trace first, design second)
**Depends on:** Iterations 24, 25

**Goal:** Run fresh trace analysis on the calibrated v24/v25 engine to identify remaining death patterns. Iter 23's bottleneck detection may have shifted the death distribution. Design targeted fixes only if trace data shows a clear, addressable root cause.

**Step 1: Trace & analyze**
```
make trace N=20
make analyze MODE=deaths
make analyze MODE=signals
make analyze MODE=turning-points
```

**Candidate signals (only if trace data supports them):**

1. **Space-to-length ratio**: `MyTerritory / me.Length`. Below 1.5× = danger, below 1.0× = critical. Gate by `lateBlend`.
2. **Partition food planning**: When `IsPartitioned`, food in our territory becomes survival-critical. Bonus/penalty based on food count vs health.
3. **Opponent space crisis**: If opponent's space ratio is worse than ours, we're likely to outlast them. Bonus.

**Decision criteria:** Only implement if trace analysis shows >30% of losses have a clear signal gap that these candidates would address. Otherwise, mark as "no clear target" and focus on other approaches (e.g., opening book, endgame tablebase, or multi-opponent support).

**Files:**
| File | Action |
|------|--------|
| `logic/eval.go` | Add late-game signals (if trace data supports) |
| `logic/eval_test.go` | Tests for new signals |

**Verify:** `make compare N=50` — target >53%.

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
| 24 | | | Weight calibration |
| 25 | | | Phase-gate bottleneck detection |
| 26 | | | Late-game survival (conditional on trace data) |
