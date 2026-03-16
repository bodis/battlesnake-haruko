# Haruko Engine Description

Battlesnake AI in Go. Iterative deepening Best-Reply Search with phase-adaptive evaluation.

## Architecture

```
HTTP request (GameState JSON)
  → main.go: convert API types to logic types
  → GameSim.BestMoveIterative(300ms budget)
    → iterative deepening: depth 1..14
      → BRS (Best-Reply Search): 2-player minimax variant
        → alpha-beta with TT + killer heuristics
        → Evaluate(): Voronoi + phase-weighted signals
  → respond with direction
```

### Files

| File | Role |
|------|------|
| `main.go` | HTTP handlers, API→logic type bridge |
| `logic/sim.go` | `GameSim`: state, `Clone`/`CloneFromPool`, `Step`, `IsOver` |
| `logic/search.go` | `BestMoveIterative` (BRS), `BestMove` (paranoid minimax), `BRSResult`/`bestMoveBRS` |
| `logic/eval.go` | `Evaluate`, `isSafeDir` (tail-aware), `safeMoveCount` |
| `logic/voronoi.go` | `VoronoiTerritory` → `VoronoiResult` (territory, food, partitions) |
| `logic/mcts.go` | `mctsSearch` — flat MCTS with UCB1 (infrastructure, not used in production) |
| `logic/zobrist.go` | Zobrist hashing for TT |
| `logic/tt.go` | Transposition table (1M entries, generation-based) |
| `logic/types.go` | `Coord`, `Direction`, `MoveSet`, `MaxSnakes=4` |

## Search

**Best-Reply Search (BRS)** — 2-player minimax where only the "best replier" opponent moves each ply. Branching factor: 4×4=16 per ply pair. Iterative deepening up to depth 14 within 300ms. Depth adapts automatically to board size via time budget.

**Move ordering:** PV/TT move → killer heuristics → static fallback. This is sufficient for BF=4.

**Transposition table:** Zobrist hash, 1M entries, probe/store with generation invalidation. Hit rate ~8% at depth 5, ~25% at depth 6. Singleton to avoid GC pressure.

**Paranoid minimax** (`BestMove`) retained for multi-opponent scenarios but degrades at depth 7+.

## Evaluation

`Evaluate(g, myIdx)` returns a float64 score. Terminal: -1000 (dead), +1000 (all opponents dead).

### Signals

| Signal | Weight | Description |
|--------|--------|-------------|
| Voronoi territory | `1.5 - 0.3×early + 0.45×late` | Multi-source BFS territory difference |
| Length advantage | `3.0 + 1.5×early - 0.75×late` | Per-opponent length delta |
| Head-to-head pressure | `8.0 - 3.2×late` | Bonus/penalty when heads ≤2 Manhattan distance |
| Opponent confinement | `(50+25×late)` / `(15+10×late)` | Opponent has 0 / 1 safe moves |
| Food urgency | `0.5 × (threshold - health)` | Inverse distance to nearest food, gated by health |
| Food cluster value | `1.0 × early` | Distance-weighted food quality (sum 1/dist), early game |
| Growth urgency | `0.3 × early` | Penalty when snake length < expected for turn |
| Tail chase | `5.0 × late` | Reward proximity to own tail when space is tight |
| Territory connectivity | `5.0 × late` | Absolute MyConnectivity — avg same-owner neighbors per territory cell (wide=3.5, corridor=2.0) |
| ~~Food reach~~ | ~~removed (Iter 31)~~ | ~~Opponent's closest food dist minus ours~~ — near zero contribution |
| ~~Food denial~~ | ~~removed (Iter 31)~~ | ~~Bonus when opponent has 0 food~~ — completely dead |
| ~~Starvation risk~~ | ~~removed (Iter 31)~~ | ~~Penalty when we have 0 food~~ — completely dead |
| ~~Bottleneck~~ | ~~removed (Iter 30)~~ | ~~Territory behind live APs (Tarjan's)~~ — anti-correlated with wins |

### Game Phase

Continuous blend factors, not discrete phases:
- **earlyBlend** (0.0–1.0): `max(lenBased, turnBased)`. 1.0 when length ≤ 4 or turn ≤ 15; fades to 0.0 by length 8 / turn 35.
- **lateBlend** (0.0–1.0): board fill ratio. 0.0 at 30% fill, 1.0 at 50%+. Boosted to 0.5 when Voronoi detects partition.

Early game boosts length and food. Late game boosts territory and reduces h2h.

### Voronoi

Multi-source BFS from all alive heads. Body segments block, tails are passable. Returns:
- `MyTerritory`, `OppTerritory` — cell counts
- `MyFood`, `OppFood` — food ownership
- `IsPartitioned` — our wavefront never met opponent's
- `MyFoodValue` — sum of 1/dist for owned food (cluster quality)
- `MyConnectivity`, `OppConnectivity` — avg same-owner neighbors per territory cell (quality metric)
- `MyThreatenedTerritory`, `OppThreatenedTerritory` — cells behind live articulation points (Tarjan's, gated by `skipBottleneck`)

Zero-alloc (workspace pooled). ~1055ns per call. Bottleneck detection (Tarjan's AP) is available but always skipped in eval since Iter 30 (anti-correlated with wins). Infrastructure retained for future experimentation. Iter 31 stripped unused fields (centroids, depth, tail reachability, closest food distances).

## Performance

### Board Size Support

Supports 7×7, 11×11, and 19×19 boards (all standard Battlesnake sizes). Board dimensions come from the API at game start — no configuration needed.

All fixed-size arrays use `maxBoardCells = 361` (19×19). Loops iterate only `Width × Height` cells, so 11×11 games pay no cost for the larger arrays. Iterative deepening naturally adapts search depth to the time budget — on 19×19 with ~3× more cells, eval is slower so the engine reaches fewer plies (est. depth 6–8 vs 12–13 on 11×11), but still uses the full 300ms.

### Hot Path

Entire hot path is allocation-free (sync.Pool + stack arrays):

| Operation | Time | Allocs |
|-----------|------|--------|
| CloneFromPool | 19ns | 0 |
| Step | 49ns | 0 |
| Evaluate (early game) | ~1270ns | 0 |
| Evaluate (late game) | ~208ns | 0 |
| BRS node (Clone+Step+Eval) | ~1199ns early | 0 |

## Version History

| Iter | What | Result |
|------|------|--------|
| 1 | Random safe move | baseline ~68 turns |
| 5 | 1-ply paranoid minimax | ~87 turns |
| 6 | Depth-3 minimax + alpha-beta | ~328 turns |
| 8 | Composite eval (Voronoi + h2h + confinement) | 88% vs v7 |
| 9 | Iterative deepening | 76% vs v8 |
| 10 | PV + killer heuristics | 54% vs v9 |
| 11 | TT + Zobrist hashing | 65% vs v10 |
| 12 | Best-Reply Search | 59% vs v11 |
| 14 | Zero-alloc hot path (sync.Pool) | 56% vs v12 |
| 16 | VoronoiResult infrastructure | (infra only) |
| 17 | Game-phase adaptive eval | 59% vs v16, ~451 avg turns |
| 19 | Voronoi strategic extraction | (infra only) |
| 20 | Food strategy signals | 54% vs v19, ~443 avg turns |
| 23 | Territory bottleneck detection (Tarjan's AP) | 58% vs v20 |
| 24 | Weight calibration (6/12 weights improved) | 61% vs v23 |
| 25 | Win/loss trace analysis | (analysis only, no code change) |
| 26 | Phase-gate bottleneck detection | 67% vs v24, ~316 avg turns |
| 27 | Wall-only move pruning in BRS | 62% vs v26, ~329 avg turns |
| 28 | Tail-aware isSafeDir (eval only) | 61% vs v27, ~436 avg turns |
| 29 | Hybrid BRS+MCTS root-level vote | ❌ Dead end (2–46%) |
| 30 | Remove bottleneck + phase-dependent confinement | 61% vs v28, ~433 avg turns |
| 31 | Eval diet: strip 3 dead signals + 6 Voronoi fields | 55% vs v17, 57% vs v28, ~442 avg turns |
| 32 | Territory connectivity signal (absolute MyConnectivity) | 56-61% vs v31, ~443-451 avg turns |
| 34 | Bottleneck-aware routing (head-side AP region) | ❌ Dead end (40-57%, depth regression) |
| 35 | Tail reachability / loopability diagnostic signals | ❌ Dead end (signals too late, 1-turn death collapse) |
| 36 | MC rollout with random play policy | ❌ Dead end (32-52%, random policy too noisy — same flaw as MCTS Iter 29) |
| 36b | MC rollout v2: smart policy + top-2 focused | ❌ Dead end (47%, 0 overrides in 7504 turns — MC structurally unable to influence decisions) |

## Dead Ends

Things that don't work for BRS at branching factor 4. Do not retry without new preconditions.

### Search pruning (Iter 15): LMR, NMP, extensions — all ≤50%
BRS tree is already narrow. LMR reduces 50% of moves (too aggressive at BF=4). NMP has no meaningful "null move" in Battlesnake. Extensions steal time from iterative deepening. These techniques require high-BF games.

### Quiescence search (Iter 13): 41–51%
QS nodes cost the same as regular BRS nodes (Clone+Step+Eval). Each extension steals a full ply from main search. Would need incremental move/unmove to be viable.

### Constant-weight food control (Iter 16): 28–51%
Food value changes across game phases. Flat weights average early benefit and late harm to neutral/negative. Solved by phase-adaptive eval in Iter 17.

### Partition short-circuit (Iter 16): 39%
Root-level `oppIdx=-1` disables opponent modeling for entire search tree, but body partitions are transient (tails retract). Would need per-node partition check (expensive).

### Heuristic move ordering (Iter 18): 47–51.5%
isSafeDir-based ordering at BRS call sites. TT+killers already handle the 1-2 best moves; reordering the remaining 2-3 has negligible cutoff impact. Center proximity tiebreaker actively misleads.

### Positional quality signals (Iter 21): 37–48%
Edge/corner penalty, territory depth adequacy, center-of-mass advantage. All three individually harmful. Voronoi territory already captures positional quality implicitly — center positions get more territory, edge positions get less. Explicit positional signals double-count and confuse BRS. Depth adequacy (MyTerritoryDepth < snake length) is misleading because Voronoi partitions fluctuate turn-to-turn and depth < length is normal, not a crisis. Tried halving weights (48%), individual isolation (37–45%), normalized center (43%). All negative.

### Opponent pressure & aggression (Iter 22): 42–49%
Dominance score (length+territory+food composite), H2H range expansion, confinement scaling, health pressure, directional pressure (push to edge). Tested 7 variants isolating each signal: full plan (42%), no directional + reduced scaling (49%), H2H scaling instead of range (47%), confinement+health only (48%), health pressure only (43%), dominance-scaled food denial (46%). All negative. Root cause: in self-play, both sides use the same eval, so aggression modulation doesn't give asymmetric advantage. The search already implicitly finds aggressive moves when they lead to better territory/length/confinement positions. Explicit aggression signals add noise that confuses BRS.

### Full unsafe move pruning (Iter 27 partial): 32%
Full `isSafeDir` pruning (wall + body collision) in `brsMax`/`brsMin`. `isSafeDir` is a static check on the current board — it doesn't account for tail retraction. Pruning an opponent move that appears to hit a body segment (but won't after the tail retracts) removes the min's actual best response, causing the engine to overestimate positions. Wall-only pruning (a subset) succeeded at 62% because walls never move.

### Tail-aware body pruning in BRS (Iter 28 partial): 43%
Even with correct tail-retraction logic in `isSafeDir`, using it for BRS move pruning is harmful. The search benefits from exploring body-collision moves to find optimal responses — removing them narrows the search tree in ways that lose strategically important lines. The eval-only approach (tail-aware confinement scoring) succeeded at 61%. Body-collision pruning in BRS is fundamentally flawed regardless of tail-awareness correctness.

### MC rollout with smart policy + top-2 focused (Iter 36b): 47%
Smart rollout policy (flood-count BFS + chase/flee + food-seek, ~1730ns/move) replacing random (~50ns), focused on BRS top-2 directions only (skip if BRS gap ≥ 5). Tested at N=100 vs v32: 47%. Deep trace analysis (20 games, 7504 turns) revealed three compounding structural failures: (1) **Phase gate kills 98% of MC decisions** — `lateBlend ≥ 0.1` occurs in only 1.2% of turns; 2176/2220 MC fires are in early game where `applyMCBias` returns BRS unchanged. (2) **When phase gate passes, spread too low** — only 4 turns had `lateBlend ≥ 0.1` AND spread ≥ 0.10. (3) **MC adjustment too weak** — max MC adjustment ever: 1.47 points vs BRS gaps of 0-1.5; zero overrides in 7504 turns. Removing phase gate entirely (simulation): MC disagrees with BRS ~47% of turns, but loss rate when ignoring MC is 50.8% (coin flip). At high confidence (spread ≥ 0.15, gap < 2): loss rate when ignoring MC drops to 41.5% — **MC is anti-predictive**. Winners have lower MC survival (0.396 vs 0.447 for losers). Smart policy gives same wrong signal as random (conservative = anti-correlated with winning). RL as rollout policy infeasible: 2.1M CNN too slow (~500μs), tiny MLP (5K params) can't learn territorial strategy, survival RL anti-correlated with winning by design. MC rollouts are a closed dead end for this engine.

### Hybrid BRS+MCTS root-level vote (Iter 29): 2–46%
Flat depth-1 MCTS (UCB1, random opponent moves, xorshift64 PRNG, ~52K sims/50ms) combined with BRS at root level. Tested 6 configurations: (1) Sequential 70/30 budget split with 0.7/0.3 weighted combination: 2%. (2) Sequential 95/5 budget split: 30% (pure budget loss — BRS is extremely sensitive to even 5% budget reduction). (3) Exact-tie-only tiebreaker with 1ms MCTS: 46%. (4) Concurrent goroutines: data race on sync.Pool (pooledGameSim.poolRef). Root causes: (a) BRS depth is the engine's primary strength and is hypersensitive to budget reduction — even 15ms less budget causes measurable depth regression. (b) Depth-1 MCTS with random opponents produces systematically wrong move preferences against optimal play — it favors moves good against weak play, which are bad against strong opponents. (c) Even as a tiebreaker on exact BRS score ties, MCTS adds noise that hurts. (d) Concurrent execution hits data races on the shared gameSimPool. Infrastructure retained: `BRSResult`, `bestMoveBRS()`, `mctsSearch()`, `mctsRoot` — available for future experimentation.

### Bottleneck routing via Tarjan's in leaf eval (Iter 34): 40-57% (depth regression)
Head-side flood fill from snake head through non-AP territory cells, penalizing when head is on the small side of an AP. Signal concept is directional (guides toward larger region, not general narrowness penalty). Failed because Tarjan's AP detection costs ~2150ns/eval — 3x the Voronoi baseline. At every BRS leaf node, this causes 1-2 plies of depth regression. Phase-gating (running Tarjan's only in late game) reduces frequency but late-game depth is the most critical. N=200 confirmation showed gate=0.5/w=10 was 50.5% (initial N=100 of 57% was noise). Any eval signal requiring Tarjan's is not viable for leaf evaluation. Infrastructure retained for diagnostic/root-only use.

### MC rollout with random play policy (Iter 36): 32–52% (random policy too noisy)
Timed MC rollouts from root position after BRS completes (separate budget, no BRS depth loss). Round-robin across valid directions, `randomSafeMove` (isSafeDir + wall fallback), xorshift64 PRNG. Phase-scaled `applyMCBias` adjusts BRS scores by survival spread. 22 configurations tested across weight (0-30), spread threshold (0.03-0.20), late gate (0.05-0.3), rollout turns (50-200). No configuration reliably beats v32 at N=100. Root cause: same flaw as Iter 29 MCTS — random opponents don't model territory collapse (deaths from optimal corridor closing), MC favors conservative play while squeezing wins. Trace analysis: MC disagrees with BRS 44% of turns (52% early, 33% late), median spread 0.078, pure noise. Infrastructure retained for v2 (smart rollout policy + top-2 focused budget).

### Tail reachability / loopability as death predictor (Iter 35): signals too late
Instrumented trace-only diagnostic signals: tail reachability (BFS from head to tail through owned territory + body), loopability (tail reachable AND territory >= snake length), TurnsToDeathEst (escape reachability when not loopable). 100-game self-play analysis found: (1) Tail is always reachable (100% of all turns, wins and losses) — zero discriminative value. (2) Loopability drops only 1-2 turns before death (48% at 1 turn, 19% at 5 turns, 6% at 20 turns) — lagging indicator, not early warning. (3) Territory collapses from 20-50 to 1 in a single turn in 90% of non-starvation deaths — the opponent closes the last corridor exit in one move. (4) Loopability asymmetry is symmetric in self-play (mine-only 3.9% in wins vs 2.4% in losses). (5) TurnsToDeathEst has MAE=36 turns, bias=-27.3 (massive underestimate). The fundamental problem: deaths are instantaneous 1-turn territory collapses beyond BRS horizon, not gradual squeezes. By the time any survival signal fires, it's already too late. These signals cannot work as eval inputs.

### Key principle
Every past win came from deeper search or better eval. Search mechanics (pruning, ordering) are saturated at BF=4. The remaining lever is eval quality — but new signals must add genuinely new information, not restate what Voronoi territory already captures. Dominance-based weight modulation is also ineffective because both sides of self-play share the same eval. Sound pruning (wall-only) is a valid third lever: it reduces BF without losing information.

## Findings

Insights from successful iterations that inform future development.

### Eval > search depth (Iter 5-7, 23)
Deeper search with a bad eval is counterproductive. Iter 23 doubled eval cost but won 58% — better eval beats deeper search.

### Weight sensitivity (Iter 20, 24)
Eval weights are highly sensitive. Iter 20: halving food strategy weights flipped 47% → 54%. Iter 24: systematic calibration found 6/12 weights undertuned, yielding 61%. Key results:
- Core weights were all too low: Territory 1.0→1.5, Length 2.0→3.0, H2H 5.0→8.0 (H2H was single strongest at 64%)
- Food weights benefit from reduction: FoodCluster 1.5→1.0, StarvationRisk 2.5→1.5
- TailChase 3.0→5.0 improved late-game survival
- ~~Bottleneck 0.3 is well-calibrated~~ Removed in Iter 30 — anti-correlated with wins (aggressive squeezers have fragile territory)
- GrowthUrgency 0.3 is important — halving it lost badly (38%)

### Search mechanics saturated at BF=4 (Iter 13, 15, 18)
Pruning (LMR, NMP), ordering (heuristic), and extensions (QS) all ≤51%. BRS already narrow. The winning lever is eval quality (Iter 8: 88%, Iter 17: 59%, Iter 20: 54%, Iter 23: 58%, Iter 24: 61%).

### Self-play limitations (Iter 22)
Both sides share the same eval, so dynamic modulation (aggression, dominance scaling) gives no asymmetric advantage. Static weight tuning works because it improves absolute position assessment.

### Territory decides everything (Iter 25 analysis)
100% of wins are territory-squeeze. Games are decided by sudden 1-2 turn territory flips (200+ eval swing). The winner often has a territory *disadvantage* for most of the game, then wins via a sudden confinement kill. LenAdvantage is the strongest differentiator between wins and losses (not territory). Deaths are 50% collision, 25% wall-collision, evenly split between mid-game and late-game. Implication: more search depth to foresee territory flips is the next lever.

### Sound pruning vs heuristic pruning (Iter 27)
Wall-only move pruning (skip moves that go off-board) is sound and wins 62% vs v26. Full `isSafeDir` pruning (wall + body) loses at 32% because body-collision checks are static and don't account for tail movement. The distinction: walls are immutable board boundaries, body segments are dynamic. Sound pruning reduces BF without information loss; heuristic pruning at BF=4 removes critical information.

### Eval accuracy vs search pruning (Iter 28)
Tail-aware `isSafeDir` (skipping retracting tails) improves eval accuracy → 61% vs v27. But using the same function to prune BRS moves loses at 43%. The distinction: eval benefits from accurate position assessment at every node, while search benefits from exploring the full move space including "bad" moves. Body-collision pruning removes the opponent's actual best responses from consideration, causing position overestimation. This confirms that BRS pruning should only remove provably impossible moves (walls), not merely "bad" ones.

### MCTS is not viable for this engine (Iter 29)
Flat MCTS with random opponent moves produces systematically wrong move preferences. At BF=4 with BRS reaching depth 12-15, MCTS can't match tactical depth — it would need 4^12 ≈ 16M paths vs ~1M sims/sec capacity. BRS's budget sensitivity is extreme: even 5% budget reduction (15ms at 300ms) causes measurable depth regression and win-rate loss. The only viable path for MCTS-family algorithms would be AlphaZero-style with a trained neural network for both policy and value, but that's a fundamentally different architecture. Concurrent BRS+MCTS via goroutines is blocked by data races on the shared gameSimPool.

### Bottleneck signal anti-correlated with wins (Iter 30)
Trace analysis of 20 games (120 perspectives) revealed the bottleneck signal (territory behind articulation points) was anti-correlated: winners averaged -0.09 contribution, losers +0.09. The aggressive squeezer pushes into narrow corridors to cut off the opponent, naturally creating more fragile territory for itself. The signal penalized this winning playstyle. Removing it (53%) + adding phase-dependent confinement weights (61%) was the winning combination. Phase-dependent confinement (stronger in late game) targets the dominant failure mode: 70% of deaths are late-game, and confinement is the kill signal.

### Trace statistics can mislead about signal importance (Iter 31)
TailChase showed δ=0.00 between wins and losses in trace analysis, suggesting it was dead. Removing it dropped v17 win rate from 55% to 44%. The signal has low *average* contribution but high *conditional* importance in specific late-game survival situations. Multi-opponent A/B testing is essential — aggregate trace statistics alone can't identify signals that matter only in critical moments.

### Absolute signals beat delta signals in self-play (Iter 32)
Connectivity delta (MyConnectivity - OppConnectivity) was neutral in self-play (50-52%), just like Iter 22's aggression modulation. Absolute MyConnectivity (rewarding our own wide territory regardless of opponent) won 56-61%. In self-play where both sides share the same eval, delta signals cancel out symmetrically. Absolute signals change move preferences asymmetrically because positions differ.

### Tarjan's AP is not viable for leaf evaluation (Iter 34)
Tarjan's articulation point detection adds ~2150ns per Voronoi call (1055ns → 3200ns = 3x cost). Running it at every BRS leaf node causes 1-2 plies of depth regression. Phase-gating (only in late game) doesn't help enough — late-game depth is the most critical for survival. The bottleneck routing signal was directional (head-side region analysis, not a general narrowness penalty) and conceptually sound, but N=200 confirmation showed 50.5% vs v32. Any future bottleneck-based signal must avoid Tarjan's in the hot path — either compute once at root per move, or use a cheaper proxy (e.g., corridor-cell-based detection instead of full AP decomposition).

### Phase-gating eval cost for depth (Iter 26)
Skipping Tarjan's AP in early game (`lateBlend < 0.1`) recovers 57% of Voronoi cost (2414ns → 1051ns), translating to ~2 extra search plies. Result: 67% vs v24 — the strongest single-iteration improvement since Iter 8. The skipped signal contributes at most ~1.65 eval points at the threshold — well below noise floor. This confirms that early-game search depth is critical: the engine needs to see territory flips coming, and cheaper eval = more depth = better foresight.
