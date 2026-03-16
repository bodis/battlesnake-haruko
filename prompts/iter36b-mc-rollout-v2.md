# Iter 36b — MC Rollout v2: Smart Policy + Top-2 Focused

## Context

Read `ROADMAP.md` (Iter 36 section — the dead end analysis and improvement ideas), `CLAUDE.md`, `ENGINE.md`.

**Key background:** Iter 36 implemented MC rollouts with `randomSafeMove` policy and tested 22 parameter configurations (32-52% vs v32, N=50+N=100). All failed because random rollouts have the same flaw as MCTS (Iter 29): random opponents don't model territory collapse, MC favors conservative play over aggressive squeezing. MC disagreed with BRS on 44% of turns — pure noise.

The infrastructure is solid. The problems are: (1) rollout policy is too dumb, (2) MC wastes budget on directions BRS already rejected. This iteration fixes both.

## What to implement

Two changes to the existing MC rollout system. Implement in order — test each independently.

### Change 1: Top-2 Focused Rollouts

**File:** `logic/rollout.go` — modify `applyMCBias` and add `MCRolloutTop2Timed`

Instead of running rollouts across all 3-4 valid directions, only run rollouts for BRS's top-2 scored directions. This:
- Focuses all rollout budget on the actual decision (right vs up, not right vs up vs down vs left)
- Gets 2x more rollouts per direction → stronger statistical signal
- MC can only swap top-1 and top-2 — never pulls toward a BRS-rejected direction
- Only activates when BRS top-2 gap is small → MC as tiebreaker, not override

```go
// MCRolloutTop2Timed runs MC rollouts only on the two best BRS directions.
// Returns result with only those two directions populated.
// If BRS gap > scoreGap, returns empty result (BRS is confident, skip MC).
func MCRolloutTop2Timed(g *GameSim, myIdx, oppIdx int, brsResult BRSResult,
    maxTurns int, budget time.Duration, scoreGap float64) MCRolloutResult
```

**Logic:**
1. From `brsResult.Scores[d]` and `brsResult.HasScore[d]`, find the top-2 scored directions
2. If only 1 valid direction or gap >= `scoreGap` → return empty result (skip MC)
3. Run timed rollouts round-robin between only those 2 directions (reuse `runOneRollout`)
4. All budget goes to 2 directions → ~2500 rollouts per direction in 20ms (vs ~1200 when split across 4)

**Wire into `BestMoveIterative`** (replace current `MCRolloutTimed` call):
```go
// Replace:
//   mcResult = MCRolloutTimed(g, myIdx, oppIdx, RolloutMaxTurns, mcBudget)
// With:
//   mcResult = MCRolloutTop2Timed(g, myIdx, oppIdx, result, RolloutMaxTurns, mcBudget, MCScoreGap)
```

**New tunable parameter:**
```go
var MCScoreGap = 5.0 // skip MC if BRS top-2 gap >= this
```
Also add env var `MC_GAP` in `main.go` init().

**Update `applyMCBias`:** when top-2 mode returns only 2 valid directions, the bias should only choose between those two. The existing code already handles this correctly (it checks `mc.Valid[d]` and `result.HasScore[d]`).

### Change 2: Smart Rollout Policy

**File:** `logic/rollout.go` — add `smartRolloutMove`, replace `randomSafeMove` calls in `runOneRollout`

Replace the random move selection with a lightweight heuristic that plays like a competent beginner. The goal: rollout snakes should actively use space efficiently and chase/flee, so territory collapse happens in rollouts too (not just in optimal play).

```go
// smartRolloutMove picks a heuristic-guided direction for rollout play.
// ~200-500ns per call (vs ~50ns for randomSafeMove). 10x slower but
// rollout snakes play territory-aware, producing meaningful survival signal.
func smartRolloutMove(g *GameSim, snakeIdx, oppIdx int, rng *xorshift64) Direction
```

**Heuristic layers (evaluated in order, first match wins):**

1. **Collect safe moves** using existing `isSafeDir` (same as current `randomSafeMove`). If only 0-1 safe moves, return that (no decision to make).

2. **Food-seek when hungry** (~10ns): if `health < 25`, find nearest food among safe directions using Manhattan distance. If a safe direction moves closer to food, pick it. Prevents starvation noise.

3. **Quick flood count** (~200-400ns): for each safe direction, do a bounded BFS from `head.Move(d)` counting reachable empty cells (not occupied by any snake body). Use stack-based BFS with a small visited bitset (stack array `[maxBoardCells]bool`). Limit to `maxFloodSteps = 12` cells (not full board — just enough to detect dead ends vs open space).
   - Score each direction by reachable cell count
   - This is the primary signal: "which direction has more space?"

4. **Chase/flee bias** (~10ns): if our `Length > opp.Length`, add +2 bonus to the direction(s) that move toward opponent head (Manhattan). If `Length < opp.Length`, add +2 to direction(s) moving away. This makes rollout snakes actively squeeze when bigger.

5. **Select with randomness**: pick the direction with the highest combined score. On ties, pick randomly among tied directions. Additionally, with 20% probability (`rng.next() % 5 == 0`), ignore the heuristic and pick a random safe move instead. This randomness is critical — without it, all rollouts from the same position play identically and you get 0 statistical variance.

**Important constraints:**
- Zero allocations. Use stack arrays only (visited bitset: `[maxBoardCells]bool`, BFS queue: `[maxBoardCells]int16`).
- Do NOT use `sync.Pool` for flood workspace — the rollout already has a pooled `GameSim` clone, adding another pool increases contention. Stack arrays are fine for 361 cells (2.9KB).
- Do NOT call `VoronoiTerritory` or `Evaluate` — way too expensive.
- The flood count does NOT need to be accurate. It's a quick space estimate. 12-step BFS is enough to distinguish "3 cells dead end" from "20+ cells open area".

**Update `runOneRollout`:** replace `randomSafeMove` calls for BOTH snakes with `smartRolloutMove`. Both snakes should play smart — if only our snake plays smart, we'd bias survival upward (same self-play symmetry issue).

### Flood count implementation detail

```go
// quickFloodCount counts cells reachable from 'start' without crossing snake bodies.
// Stops after maxSteps cells. Returns count of reachable cells (0 to maxSteps).
// Zero-alloc: uses stack arrays.
func quickFloodCount(g *GameSim, start Coord, maxSteps int) int {
    // Build body occupancy set (iterate all alive snake bodies, mark cells)
    // BFS from start, count reachable non-body cells up to maxSteps
    // Use g.Width*g.Height as board size, index = y*width + x
}
```

Note: building the body occupancy set each call (~50ns for 2 snakes with ~10 body segments each) is fine. Alternative: pass a pre-built occupancy grid, but that adds complexity for minimal gain at this scale.

## Testing

### Unit tests (`logic/rollout_test.go`)

1. **TestSmartRolloutMoveBasic**: snake in open position, verify it returns a safe direction
2. **TestSmartRolloutMoveDeadEnd**: snake with 1 safe direction, verify it picks that direction
3. **TestSmartRolloutMovePreferSpace**: snake at T-junction where one direction has 3 cells, other has 20+ cells — verify it prefers the open direction (run 100 times, check >70% picks open)
4. **TestQuickFloodCount**: verify flood count on known positions (corner=few, center=many)
5. **TestMCRolloutTop2**: verify that only 2 directions get rollouts, others have Total=0

### Benchmarks

```go
func BenchmarkSmartRolloutMove(b *testing.B) // target: <500ns
func BenchmarkQuickFloodCount(b *testing.B)  // target: <300ns
func BenchmarkMCRolloutTop2(b *testing.B)    // 20ms budget, report rollout count
```

### A/B comparison

After implementation, run parameter sweep against v32:

```bash
# First test with Change 1 only (top-2 focused, random policy still)
make snapshot  # save current as baseline
# Then test with Change 1+2 (top-2 + smart policy)

# Key configs to test (N=100 each):
# gap=3, gap=5, gap=8, gap=10
# w=10, w=15, w=20
# spread=0.05, spread=0.10, spread=0.15
```

Use the existing `scripts/mc_sweep.sh` infrastructure (add MC_GAP env var).

## What NOT to do

- Do NOT change `Evaluate()` or eval weights
- Do NOT change `bestMoveBRS()` or BRS search logic
- Do NOT add goroutine parallelism yet (that's a separate improvement)
- Do NOT use `VoronoiTerritory` or any expensive analysis in rollout moves
- Do NOT make the flood count exact — 12-step bounded BFS is enough
- Do NOT remove `randomSafeMove` — keep it for fallback and tests. Add `smartRolloutMove` alongside it.
- Do NOT change the trace diagnostic MC (in `trace.go`) — it can keep using the old `MCRolloutTimed` for comparison

## Success criteria

1. `BenchmarkSmartRolloutMove` < 500ns (10x budget over randomSafeMove's ~50ns)
2. `BenchmarkMCRolloutTop2` produces 1000+ rollouts per direction in 20ms
3. At least one parameter configuration beats v32 at **N=100 with win rate > 55%**
4. MC disagreement with BRS drops from 44% to < 15% (top-2 eliminates most noise)

## After implementation

Run the comparison sweep and show the results. If a config beats v32 at N=100 > 55%, confirm at N=200. Then we document and commit.
