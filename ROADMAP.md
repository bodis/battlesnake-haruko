# Haruko Battlesnake — Development Roadmap

> Active development plan. Completed iterations are archived in [ROADMAP_FINISHED.md](ROADMAP_FINISHED.md).
> Each iteration: implement → test → snapshot → compare → merge → move to finished → update ENGINE.md.
> Development follows a data-driven loop: trace games → analyze outcomes → identify root causes → design targeted fixes → verify with A/B comparison.

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iterations 1-20, 23-27 (see ROADMAP_FINISHED.md) |
| **Dead ends** | Iter 21 (positional quality), Iter 22 (aggression), Iter 27 partial (full isSafeDir pruning: 32%) |
| **Next** | Iter 28 — Tail-aware safe move check |
| **Current** | v27 Wall-only move pruning; 62% vs v26 (N=100); ~329 avg turns |
| **Key insight** | `isSafeDir` is static — doesn't account for tail retraction. This causes two problems: (1) BRS body-collision pruning is unsound (Iter 27 dead end at 32%), (2) eval overestimates opponent confinement when their escape is a retracting tail. |

---

## Iter 28 — Tail-Aware Safe Move Check

**Goal:** Make `isSafeDir` account for tail retraction. Two independent benefits:

1. **More accurate eval** — `safeMoveCount` / opponent confinement scoring currently reports "0 safe moves" when the opponent can actually escape via a retracting tail. Fixing this removes false +50/+15 confinement bonuses → more accurate position assessment at every node.

2. **Sound body-collision pruning in BRS** — with a correct `isSafeDir`, we can prune body-collision moves in `brsMax`/`brsMin` (which failed at 32% with the static check). Body collisions are more frequent than wall collisions, especially in mid/late game where snakes are long.

**Approach:**

Create `isSafeDirTailAware(g, s, d)` that skips a snake's tail segment (`Body[len-1]`) when checking body collisions, **unless**:
- **Stacked tail:** `Body[len-1] == Body[len-2]` — the growth segment after eating occupies the same cell, so only one retracts and the cell stays blocked.
- **Food adjacent to head:** if any food is within Manhattan distance 1 of that snake's head, it might eat this turn → tail won't retract. Conservative: assume it eats.

This is strictly conservative — it may still flag some moves as unsafe that aren't, but will never flag a truly unsafe move as safe. Sound for both eval and pruning.

**Implementation:**

| File | Change |
|------|--------|
| `logic/eval.go` | Add `isSafeDirTailAware`, update `isSafeDir` to call it (or replace). Update `safeMoveCount` accordingly. |
| `logic/search.go` | Replace `wallSafeMoves` with tail-aware body pruning in `brsMax`/`brsMin` (subsumes wall-only). |
| `logic/search_test.go` | Tests: tail-chase escape, stacked tail, food-adjacent tail, cornered opponent. |
| `logic/eval_test.go` | Tests: confinement scoring with retracting tail. |

**Verification:**
1. Correctness: `go test ./logic/ -run TestTailAware -v`
2. Resource: `go test -bench=BenchmarkBestMoveIterativeDepth -benchtime=10x ./logic/`
3. Outcome: `make snapshot && make compare PREV=snapshots/haruko-wallonly N=100`

**Expected impact:** Both eval accuracy and BF reduction. Eval fix affects every node. Body pruning reduces BF beyond wall-only (most moves blocked by `isSafeDir` are body hits, not walls). Combined effect should exceed the 62% from wall-only pruning.

---

## Next Steps

Candidate directions for future iterations:

**Late-game survival signals:**
Space-to-length ratio, partition food planning, opponent space crisis.

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
| 27 | `snapshots/haruko-wallonly` | ~329 | Wall-only move pruning; 62% vs v26 |
