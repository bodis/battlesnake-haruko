# Haruko Battlesnake — Development Roadmap

> Active development plan. Completed iterations are archived in [ROADMAP_FINISHED.md](ROADMAP_FINISHED.md).
> Each iteration: implement → test → snapshot → compare → merge → move to finished → update ENGINE.md.
> Development follows a data-driven loop: trace games → analyze outcomes → identify root causes → design targeted fixes → verify with A/B comparison.

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iterations 1-20, 23-26 (see ROADMAP_FINISHED.md for 1-20, 23-26) |
| **Dead ends** | Iter 21 (positional quality), Iter 22 (aggression) |
| **Next** | TBD — analyze Iter 26 results to determine next lever |
| **Current** | v26 Phase-gate bottleneck; BRS depth ~14-15 early, ~12-13 late; Evaluate ~1116ns early / ~2450ns late; 67% vs v24 |
| **Key insight** | Skipping Tarjan's AP in early game (lateBlend < 0.1) recovers 57% of Voronoi cost, translating to ~2 extra search plies. Early-game depth is critical for foreseeing territory flips. |

---

## Next Steps

Candidate directions for future iterations:

**Late-game survival signals:**
Space-to-length ratio, partition food planning, opponent space crisis.

**New eval signals (TBD):**
Data-driven — trace new v26 games to identify remaining weaknesses.

**Further phase-gating:**
Other eval signals that could be skipped/simplified in early game for more depth.

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
