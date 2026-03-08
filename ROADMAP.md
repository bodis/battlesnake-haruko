# Haruko Battlesnake — Development Roadmap

> Active development plan. Completed iterations are archived in [ROADMAP_FINISHED.md](ROADMAP_FINISHED.md).
> Each iteration: implement → test → snapshot → compare → merge → move to finished → update ENGINE.md.
> Development follows a data-driven loop: trace games → analyze outcomes → identify root causes → design targeted fixes → verify with A/B comparison.

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iterations 1-20, 23-28, 30-31 (see ROADMAP_FINISHED.md) |
| **Dead ends** | Iter 21 (positional quality), Iter 22 (aggression), Iter 27 partial (full isSafeDir pruning: 32%), Iter 28 partial (tail-aware BRS pruning: 43%), Iter 29 (hybrid BRS+MCTS: 2–46%) |
| **Current** | v31 Eval diet: strip 3 dead signals + 6 Voronoi fields; 55% vs v17, 57% vs v28 (N=100); ~442 avg turns |
| **Key insight** | Multi-opponent validation gate works: TailChase appeared dead in trace data (δ=0.00) but removing it dropped v17 win rate from 55% to 44%. Low average signal contribution ≠ unimportant. |

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

## Iter 32 — Territory Quality: Connectivity + Squeeze Detection

**Goal:** Replace raw territory count with a quality-aware territory signal. The #1 differentiator between wins and losses is territory *trend* (wins: +10 to +30, losses: -47 to -62 in late game), not territory *count*. Current eval counts all territory cells equally — 30 cells in wide-open space and 30 cells behind a 1-cell corridor score the same. This iteration adds territory quality measurement to break the hill-climbing plateau.

**Why this should work:** Depth 14 analysis (Iter 31 traces) proved the depth hypothesis was wrong — we reach depth 14 in both wins and losses. The problem isn't search depth, it's eval quality in the late game. Late-game losses follow a universal pattern: gradual territory erosion → self-confinement → forced into wall/collision. This happens over 100+ turns, well beyond search horizon. Better eval scoring of territory *quality* should make the engine avoid narrow-corridor positions before the slow squeeze begins.

**Acceptable depth trade-off:** Losing 1 ply (depth 14→13) for a ~100-200ns territory quality signal is acceptable. All deaths happen well beyond the search horizon. The 14th ply can't save us from a 100-turn positional squeeze; better eval at depth 13 can.

### Evidence from Iter 31 Trajectory Analysis

**Traced 20 games each against v17, v28, v30. Key findings:**

1. **Early/mid game (turn 0-250) is identical** between wins and losses. Territory, eval, length advantage — all nearly the same. The game is decided purely in the late game.

2. **Late-game territory trend is the #1 predictor:**
   - vs v17: wins +29.8 trend, losses -61.9 trend (delta: 91.7)
   - vs v28: wins +10.7 trend, losses -47.3 trend (delta: 58.0)
   - vs v30: wins -18.2 trend, losses -61.4 trend (delta: 43.1)

3. **Self-confinement events (turns with ≤1 safe move) are 2x higher in losses:**
   - vs v17: wins 8.6/game, losses 15.0/game
   - vs v28: wins 9.7/game, losses 23.3/game
   - vs v30: wins 16.9/game, losses 28.1/game

4. **Partitioned games (fully cut off) are more common in losses:**
   - Wins: 19-21% of late-game turns partitioned
   - Losses: 24-33% of late-game turns partitioned

5. **100% of deaths are turn 300+, 100% are collision/wall-collision** — slow positional death, never sudden mistakes.

6. **All opponents show same pattern** — this is a structural weakness in our eval, not opponent-specific.

### Trace data location

Traced games from Iter 31 analysis are stored in:
- `traces/vs-v17/` — 20 games (10W/10L, N=20)
- `traces/vs-v28/` — 20 games (6W/14L, N=20)
- `traces/vs-v30/` — 20 games (12W/8L, N=20)

Analyze with: `go run ./cmd/analyze -mode trajectories traces/vs-v17/*.jsonl`

### Step 1: Territory Connectivity Signal

**Concept:** During the existing Voronoi territory counting loop (`voronoi.go`, territory counting section ~lines 327-340), for each cell we own, count how many of its 4 neighbors are also ours. Sum this and divide by territory count to get average connectivity.

- Wide-open territory: ~3.0-3.5 avg neighbors (interior cells have 4, edge cells have 2-3)
- Narrow corridor: ~2.0 avg neighbors (each cell has mostly 2 neighbors: forward and backward)
- Dead-end pocket: ~1.5 avg neighbors

**Implementation in `voronoi.go`:**

```go
// In VoronoiResult, add:
MyConnectivity  float64 // avg neighbor count for our territory cells (higher = wider territory)
OppConnectivity float64 // same for opponent

// In the territory counting loop:
var myNeighborSum, oppNeighborSum int
for i := 0; i < size; i++ {
    o := ws.owner[i]
    if o <= 0 { continue }
    // Count same-owner neighbors
    x := i % g.Width
    y := i / g.Width
    neighbors := 0
    if x > 0 && ws.owner[i-1] == o { neighbors++ }
    if x < g.Width-1 && ws.owner[i+1] == o { neighbors++ }
    if y > 0 && ws.owner[i-g.Width] == o { neighbors++ }
    if y < g.Height-1 && ws.owner[i+g.Width] == o { neighbors++ }
    if o == myTag {
        result.MyTerritory++
        myNeighborSum += neighbors
    } else {
        result.OppTerritory++
        oppNeighborSum += neighbors
    }
}
if result.MyTerritory > 0 {
    result.MyConnectivity = float64(myNeighborSum) / float64(result.MyTerritory)
}
// same for opp
```

**Cost:** 4 comparisons + 4 additions per territory cell. Estimated ~30-50ns per Voronoi call on 11x11 board. Very cheap.

**In `eval.go`:** Add a connectivity signal:
```go
// Territory quality: penalize narrow corridor territory, reward wide-open territory
// Connectivity ranges from ~1.5 (dead end) to ~3.5 (wide open)
// Neutral point around 2.5 — below that, territory is fragile
connectivityDelta := vr.MyConnectivity - vr.OppConnectivity
score += wConnectivity * lateBlend * connectivityDelta
```

Scale with `lateBlend` because early-game territory is fluid and connectivity doesn't matter yet. Start with `wConnectivity = 3.0-5.0` (similar weight to TailChase).

### Step 2: Squeeze Momentum Signal

**Concept:** Reward positions where our territory is stable/growing and opponent's is shrinking. This captures the "slow squeeze" pattern that decides games.

**Implementation:** This is tricky because eval sees only a single position. But within BRS search, each node's eval is compared against alpha/beta from parent nodes. The search already implicitly values positions where our territory grows.

**Alternative approach — territory ratio instead of delta:**
Instead of `MyTerritory - OppTerritory` (current), use a ratio-based signal: `MyTerritory / (MyTerritory + OppTerritory)`. This naturally captures squeeze dynamics:
- 50/50 territory = 0.5 (neutral)
- 60/40 territory = 0.6 (dominant)
- 30/70 territory = 0.3 (losing)

The ratio signal is more meaningful than absolute delta because a 10-cell advantage on a crowded board is worth more than on an open board. This is almost free — just changes the arithmetic in eval, no new data needed.

**In `eval.go`:** Replace or supplement the territory delta:
```go
totalTerritory := vr.MyTerritory + vr.OppTerritory
if totalTerritory > 0 {
    territoryRatio := float64(vr.MyTerritory) / float64(totalTerritory)
    // 0.5 = neutral, >0.5 = advantage, <0.5 = disadvantage
    // Scale to eval range: ratio 0.6 → score +10, ratio 0.4 → score -10
    score += wTerritoryRatio * (territoryRatio - 0.5) * 100
}
```

### Step 3: Head Escape Routes

**Concept:** From our head, count how many of the 4 directions lead to cells we own (our Voronoi territory). If only 1 direction leads to our territory, we're in a corridor facing a dead end. If 3-4 directions lead to our territory, we have options.

**Implementation in `eval.go`** (not Voronoi — uses the VoronoiResult owner data indirectly):

Actually, this can be approximated cheaply: after Voronoi, check how many safe moves lead to cells in our territory. This combines `isSafeDir` with Voronoi ownership. But we don't have per-cell ownership in eval...

**Alternative: use `safeMoveCount` as a proxy.** We already compute `safeMoveCount(g, me)` — this counts moves that don't hit walls or bodies. In late game with large snakes, safe moves that lead into our own territory are the ones that matter. The current self-confinement signal already captures this (0 or 1 safe moves = penalty).

**Better approach:** Enhance the self-confinement signal to be more granular. Currently it's binary (0 moves → big penalty, 1 move → medium penalty, 2+ → nothing). Make it continuous and scale with territory connectivity:
```go
safeMoves := safeMoveCount(g, me)
if safeMoves <= 2 {
    selfConfinePenalty := (3.0 - float64(safeMoves)) * (5.0 + 10.0*lateBlend)
    if vr.MyConnectivity < 2.5 { // narrow territory amplifies confinement danger
        selfConfinePenalty *= 1.5
    }
    score -= selfConfinePenalty
}
```

### Step 4: Implementation Order

1. **Add `MyConnectivity`/`OppConnectivity` to `VoronoiResult`** and compute in territory loop
2. **Add connectivity signal to `Evaluate()`** — weighted by `lateBlend`
3. **Benchmark** — verify eval cost increase is <200ns
4. **A/B test connectivity alone** at N=100 vs v17, v28, v30
5. **If connectivity passes:** add territory ratio signal, test again
6. **If connectivity fails:** try head escape routes or enhanced self-confinement instead
7. **Multi-opponent validation gate (same as Iter 31):** must beat v17 >55% AND v28 >55%

### Step 5: Tuning Approach

Test connectivity weight at 3.0, 5.0, 8.0 against v17 (not v30 — breaks hill-climbing trap). The weight that beats v17 most consistently is correct.

If territory ratio signal is added, test independently first, then combined.

### Step 6: Iteration Completion

Standard workflow:
1. `make compare PREV=snapshots/haruko-ea8e2d7 N=100` — vs v17 (must >55%)
2. `make compare PREV=snapshots/haruko-c77fa1f N=100` — vs v28 (must >55%)
3. Move iteration to ROADMAP_FINISHED.md
4. Update ENGINE.md, CLAUDE.md

### Critical Files

| File | Changes |
|------|---------|
| `logic/voronoi.go` | Add `MyConnectivity`/`OppConnectivity` to `VoronoiResult`, compute in territory loop |
| `logic/eval.go` | Add connectivity signal, possibly territory ratio, enhanced self-confinement |
| `logic/eval.go` | Update `EvaluateDetailed` and `EvalBreakdown` to include new signals |
| `trace.go` | Add new signal fields to `traceRecord` |
| `cmd/analyze/main.go` | Add new signals to analysis |

### Constraints
- Territory connectivity must be computed in the existing territory loop (no second pass)
- Total eval cost increase must be <200ns (acceptable: depth 14→13)
- Do NOT change search mechanics
- Multi-opponent validation gate is mandatory
- `cmd/analyze -mode trajectories` is available for per-phase analysis of results

### What NOT to try (known dead ends)
- Explicit positional signals (center/edge) — Iter 21 proved these double-count Voronoi (37-48%)
- Aggression modulation — Iter 22 proved ineffective in self-play (42-49%)
- Full Tarjan's AP — Iter 30 proved anti-correlated with wins
- Body-collision pruning — Iter 27/28 proved harmful to search (32-43%)

---

## Future Directions (After Iter 32)

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
