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
| Food reach advantage | 0.5 | Opponent's closest food dist minus ours |
| Food denial | 2.0 | Bonus when opponent has 0 food and health < 40 |
| Starvation risk | 1.5 | Penalty when we have 0 food and health < 50 |
| Growth urgency | `0.3 × early` | Penalty when snake length < expected for turn |
| Tail chase | `5.0 × late` | Reward proximity to own tail when space is tight |
| ~~Bottleneck~~ | ~~removed (Iter 30)~~ | ~~Territory behind live articulation points (Tarjan's)~~ — anti-correlated with wins |

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
- `MyClosestFoodDist`, `OppClosestFoodDist` — BFS distance to nearest owned food
- `MyFoodValue` — sum of 1/dist for owned food (cluster quality)
- `MyTerritoryDepth` — max BFS distance in our territory
- `MyCenterX/Y`, `OppCenterX/Y` — territory centroids
- `MyTailReachable` — tail cell in our Voronoi territory
- `MyThreatenedTerritory`, `OppThreatenedTerritory` — cells behind live articulation points (Tarjan's)

Zero-alloc (workspace pooled). ~1050ns per call. Bottleneck detection (Tarjan's AP) is available but always skipped in eval since Iter 30 (anti-correlated with wins). Infrastructure retained for future experimentation.

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
| Evaluate (early game) | ~1163ns | 0 |
| Evaluate (late game) | ~244ns | 0 |
| BRS node (Clone+Step+Eval) | ~1233ns early | 0 |

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

### Hybrid BRS+MCTS root-level vote (Iter 29): 2–46%
Flat depth-1 MCTS (UCB1, random opponent moves, xorshift64 PRNG, ~52K sims/50ms) combined with BRS at root level. Tested 6 configurations: (1) Sequential 70/30 budget split with 0.7/0.3 weighted combination: 2%. (2) Sequential 95/5 budget split: 30% (pure budget loss — BRS is extremely sensitive to even 5% budget reduction). (3) Exact-tie-only tiebreaker with 1ms MCTS: 46%. (4) Concurrent goroutines: data race on sync.Pool (pooledGameSim.poolRef). Root causes: (a) BRS depth is the engine's primary strength and is hypersensitive to budget reduction — even 15ms less budget causes measurable depth regression. (b) Depth-1 MCTS with random opponents produces systematically wrong move preferences against optimal play — it favors moves good against weak play, which are bad against strong opponents. (c) Even as a tiebreaker on exact BRS score ties, MCTS adds noise that hurts. (d) Concurrent execution hits data races on the shared gameSimPool. Infrastructure retained: `BRSResult`, `bestMoveBRS()`, `mctsSearch()`, `mctsRoot` — available for future experimentation.

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

### Phase-gating eval cost for depth (Iter 26)
Skipping Tarjan's AP in early game (`lateBlend < 0.1`) recovers 57% of Voronoi cost (2414ns → 1051ns), translating to ~2 extra search plies. Result: 67% vs v24 — the strongest single-iteration improvement since Iter 8. The skipped signal contributes at most ~1.65 eval points at the threshold — well below noise floor. This confirms that early-game search depth is critical: the engine needs to see territory flips coming, and cheaper eval = more depth = better foresight.
