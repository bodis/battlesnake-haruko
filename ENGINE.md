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

**Best-Reply Search (BRS)** — 2-player minimax where only the "best replier" opponent moves each ply. Branching factor: 4×4=16 per ply pair. Iterative deepening up to depth 14 within 300ms.

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
| Food control | `1.5 × early` | Food cells in our Voronoi territory (early game only) |

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

Zero-alloc (workspace pooled). ~1025ns per call.

## Performance

Entire hot path is allocation-free (sync.Pool + stack arrays):

| Operation | Time | Allocs |
|-----------|------|--------|
| CloneFromPool | 19ns | 0 |
| Step | 49ns | 0 |
| Evaluate | 1090ns | 0 |
| BRS node (Clone+Step+Eval) | ~1130ns | 0 |

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

### Key principle
Every past win came from deeper search or better eval. Search mechanics (pruning, ordering) are saturated at BF=4. The remaining lever is eval quality.
