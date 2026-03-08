# Haruko Battlesnake — Development Roadmap

> Active development plan. Completed iterations are archived in [ROADMAP_FINISHED.md](ROADMAP_FINISHED.md).
> Each iteration: implement → test → snapshot → compare → merge → move to finished → update ENGINE.md.
> Development follows a data-driven loop: trace games → analyze outcomes → identify root causes → design targeted fixes → verify with A/B comparison.

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iterations 1-20, 23-28, 30-32 (see ROADMAP_FINISHED.md) |
| **Dead ends** | Iter 21 (positional quality), Iter 22 (aggression), Iter 27 partial (full isSafeDir pruning: 32%), Iter 28 partial (tail-aware BRS pruning: 43%), Iter 29 (hybrid BRS+MCTS: 2–46%) |
| **Current** | v32 Territory connectivity: absolute MyConnectivity signal; 56-61% vs v31 (N=100); ~443-451 avg turns |
| **Key insight** | Absolute connectivity (rewarding our own wide territory) works; delta connectivity (my-opp) is neutral in self-play like Iter 22's aggression. New signals must be asymmetric to matter in self-play. |

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

**Weight recalibration on quality eval:**
After adding connectivity signal, all weights may need recalibration. Test against v17.

**Opponent squeeze reward:**
If connectivity works, consider rewarding positions where opponent connectivity is *decreasing* (we're squeezing their corridors).

**Adaptive time management:**
Allocate more search time when territory connectivity is low (critical positions) and less when connectivity is high (stable positions).

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
| 32 | `snapshots/haruko-2fbfa99` | ~443-451 | Territory connectivity (absolute MyConnectivity); 56-61% vs v31 |
