# Haruko Battlesnake — Development Roadmap

> Active development plan. Completed iterations are archived in [ROADMAP_FINISHED.md](ROADMAP_FINISHED.md).
> Each iteration: implement → test → snapshot → compare → merge → move to finished → update ENGINE.md.
> Development follows a data-driven loop: trace games → analyze outcomes → identify root causes → design targeted fixes → verify with A/B comparison.

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iterations 1-20, 23-25 (see ROADMAP_FINISHED.md for 1-20, 23-24) |
| **Dead ends** | Iter 21 (positional quality), Iter 22 (aggression) |
| **Next** | Iteration 26 (phase-gate bottleneck detection) |
| **Current** | v24 Weight calibration; BRS depth ~12-13; Evaluate ~2450ns/0 allocs; 61% vs v23 |
| **Key insight** | 100% of wins are territory-squeeze. Games decided by sudden 1-2 turn territory flips (200+ eval swing). More depth to foresee these flips is the next lever. |

---

## Phase 10: Data-Driven Analysis

> **The situation:** v24 changed 6 eval weights significantly (H2H nearly doubled, territory +50%, food weights reduced).
> This likely shifted how the engine wins and loses. Before building the next feature (phase-gating, new signals, etc.),
> we need to understand the current engine's behavior: what kills us, what kills the opponent, and where the biggest
> improvement opportunities lie. Analysis first, design second.

### Iteration 25 — Win/Loss Trace Analysis

**Status:** DONE
**Depends on:** Iteration 24

**Goal:** Comprehensive analysis of v24's game outcomes to guide next iteration.

**Findings (N=20 self-play games, 40 traces):**

1. **Win classification: 100% territory-squeeze.** No h2h-kills or starvation-kills. Territory is the only mechanism that wins games.

2. **Death causes:** collision=10 (50%), wall-collision=5 (25%), body-collision=2 (10%), starvation=2 (10%), head-collision=1 (5%). Collisions dominate — snakes are dying by running into things, not starving.

3. **Death phase:** early(<50)=1, mid(50-200)=9, late(>200)=10. Roughly even mid/late — no single phase is weak.

4. **Territory drives everything.** In every loss analyzed, the largest signal drop was territory (drops of 30–173 over last 10 turns). In wins, territory swings of +100-175 precede victory. Games are decided by sudden 1-2 turn territory flips.

5. **Eval swings are sudden and massive.** Typical pattern: eval goes from +20 to -190 in one turn (or vice versa). This means the search can't see these territory flips coming — they're beyond BRS horizon.

6. **Signals analysis (wins vs losses avg):** Territory=-1.07 in wins, +0.61 in losses. LenAdvantage=+0.12 in wins, -0.95 in losses. This is counterintuitive — the winner often had a territory *disadvantage* for most of the game, then won via a sudden flip. LenAdvantage is the strongest differentiator between wins and losses.

7. **OppConfinement is the kill signal.** Most wins show confinement jumping to +50 (0 safe moves) 1-2 turns before victory. Territory squeezes create confinement which creates the kill.

**Synthesis:**
- Losses are NOT due to eval gaps — both sides have the same eval. The deciding factor is who gets caught in a sudden territory flip.
- The sudden territory flips (200+ eval swing in 1 turn) suggest the engine can't foresee these positional crises far enough in advance.
- **Next iteration direction: phase-gate bottleneck detection.** Since all games are decided by territory and the engine already has strong territory eval, the best lever is getting more search depth to see territory flips earlier. Skip Tarjan's AP in early game to reclaim ~2 plies.

---

### Candidate iterations (pending Iter 25 analysis)

These are potential next steps. Which one (if any) depends on what the trace analysis reveals.

**Phase-gate bottleneck detection:**
Skip Tarjan's AP in early game (`lateBlend < 0.1`) to reclaim ~2 search plies. Best if losses aren't concentrated in a specific eval gap.

**Late-game survival signals:**
Space-to-length ratio, partition food planning, opponent space crisis. Best if trace shows late-game territory collapse as the dominant death pattern.

**New eval signals (TBD):**
Whatever the trace analysis reveals. Could be something we haven't considered yet.

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
