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
| `logic/search.go` | `BestMoveIterative` (BRS), `BestMove` (paranoid minimax) |
| `logic/eval.go` | `Evaluate`, `isSafeDir`, `safeMoveCount` |
| `logic/voronoi.go` | `VoronoiTerritory` → `VoronoiResult` (territory, food, partitions) |
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
| Voronoi territory | `1.0 - 0.2×early + 0.3×late` | Multi-source BFS territory difference |
| Length advantage | `2.0 + 1.0×early - 0.5×late` | Per-opponent length delta |
| Head-to-head pressure | `5.0 - 2.0×late` | Bonus/penalty when heads ≤2 Manhattan distance |
| Opponent confinement | 50.0 / 15.0 | Opponent has 0 / 1 safe moves |
| Food urgency | `0.5 × (threshold - health)` | Inverse distance to nearest food, gated by health |
| Food cluster value | `1.5 × early` | Distance-weighted food quality (sum 1/dist), early game |
| Food reach advantage | 0.5 | Opponent's closest food dist minus ours |
| Food denial | 2.0 | Bonus when opponent has 0 food and health < 40 |
| Starvation risk | 2.5 | Penalty when we have 0 food and health < 50 |
| Growth urgency | `0.3 × early` | Penalty when snake length < expected for turn |
| Bottleneck | `0.3 × (0.5 + 0.5×late)` | Territory behind live articulation points (Tarjan's) |

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

Zero-alloc (workspace pooled). ~2400ns per call (includes Tarjan's bottleneck detection).

## Performance

### Board Size Support

Supports 7×7, 11×11, and 19×19 boards (all standard Battlesnake sizes). Board dimensions come from the API at game start — no configuration needed.

All fixed-size arrays use `maxBoardCells = 361` (19×19). Loops iterate only `Width × Height` cells, so 11×11 games pay no cost for the larger arrays. Iterative deepening naturally adapts search depth to the time budget — on 19×19 with ~3× more cells, eval is slower so the engine reaches fewer plies (est. depth 6–8 vs 12–13 on 11×11), but still uses the full 300ms.

## Performance

Entire hot path is allocation-free (sync.Pool + stack arrays):

| Operation | Time | Allocs |
|-----------|------|--------|
| CloneFromPool | 19ns | 0 |
| Step | 49ns | 0 |
| Evaluate | ~2450ns | 0 |
| BRS node (Clone+Step+Eval) | ~2490ns | 0 |

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

### Key principle
Every past win came from deeper search or better eval. Search mechanics (pruning, ordering) are saturated at BF=4. The remaining lever is eval quality — but new signals must add genuinely new information, not restate what Voronoi territory already captures. Dominance-based weight modulation is also ineffective because both sides of self-play share the same eval.
