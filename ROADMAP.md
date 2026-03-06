# Haruko Battlesnake — Development Roadmap

> Active development plan. Completed iterations are archived in [ROADMAP_FINISHED.md](ROADMAP_FINISHED.md).
> Each iteration: implement → test → snapshot → compare → merge → move to finished → update ENGINE.md.

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iterations 1-20 (see ROADMAP_FINISHED.md) |
| **Dead ends** | Iter 21 (positional quality — Voronoi already captures it) |
| **Next** | Iteration 22 |
| **Current** | v20 Food strategy signals; BRS depth 14; ~443 avg turns; Evaluate ~1130ns/0 allocs |
| **Key insight** | Search mechanics exhausted at BF=4. The remaining lever is **eval quality** — but new signals must add genuinely new information, not restate what Voronoi territory already captures. |

---

## Phase 8: Strategic Board Understanding

> **The gap:** The current eval is a snapshot scorer — it measures "how good is this position right now"
> but has no strategic awareness. It can't reason about food routes, spatial opportunity, resource
> denial, or whether the current game phase demands growth vs aggression vs survival.
>
> The Voronoi BFS already computes `owner[]` and `dist[]` for every cell but we discard most of it.
> These iterations extract strategic signals from existing data and teach the eval to reason about
> the whole board, not just count cells.

### Iteration 21 — Positional Quality ❌ DEAD END

All three signals (edge/corner penalty, territory depth adequacy, center-of-mass advantage) individually harmful (37–48%). Voronoi territory already captures positional quality implicitly. See ENGINE.md dead ends.

---

### Iteration 22 — Opponent Pressure & Aggression Mode

**Status:** TODO
**Depends on:** Iteration 20

**Goal:** Adapt play style based on relative strength. When we're dominant (longer + better food access + more territory), play aggressively to close out the game. When we're weaker, play defensively to survive and find food.

**Problem:** Current eval rewards being in good positions but doesn't shift *strategy* based on who's winning. A snake that's 5 cells longer should actively cut off the opponent, not just passively maintain territory. A snake that's shorter should avoid confrontation zones entirely.

**New signals:**

1. **Dominance score** (composite): Combine length ratio, territory ratio, and food access into a single continuous dominance factor (-1.0 to +1.0).
   ```
   lenRatio = clamp(float64(me.Length - opp.Length) / 5.0, -1, 1)
   terrRatio = clamp(float64(vr.MyTerritory - vr.OppTerritory) / 20.0, -1, 1)
   foodRatio = clamp(float64(vr.MyFood - vr.OppFood) / 3.0, -1, 1)
   dominance = 0.4*lenRatio + 0.4*terrRatio + 0.2*foodRatio
   ```

2. **Aggression modulation**: When dominant (dominance > 0.3), boost h2h pressure range (activate at dist ≤ 4 instead of ≤ 2), increase confinement weight. When losing (dominance < -0.3), reduce h2h range to 1 (avoid fights), boost food urgency threshold.
   ```
   h2hRange = 2 + int(2 * max(dominance, 0))      // 2 normally, up to 4 when dominant
   wConfinement = baseConfinement * (1 + dominance) // amplify when winning
   ```

3. **Opponent health exploitation**: When opponent health < 30 AND we control more food, the opponent is in a resource crisis. Bonus proportional to their desperation — they'll be forced into risky moves.
   ```
   if opp.Health < 30 && vr.MyFood > vr.OppFood:
       score += wHealthPressure * float64(30 - opp.Health) / 30.0
   ```

4. **Directional pressure**: When dominant, prefer positions that push the opponent toward edges/corners (reduce their centroid distance to board edge). Use opponent centroid from Iter 19.
   ```
   if dominance > 0.2:
       oppEdgeness = boardCenterDist(oppCenter)
       score += wPushToEdge * dominance * oppEdgeness
   ```

**Phase interaction:** Aggression mode primarily mid-game (earlyBlend low, lateBlend low). In early game, always prioritize growth. In late game / partition, aggression is irrelevant.

**Files:**
| File | Action |
|------|--------|
| `logic/eval.go` | Dominance computation, aggression modulation, health exploitation |
| `logic/eval_test.go` | Test dominance factor, h2h range expansion, health pressure |

**Verify:** `make compare N=100` — target: >55%. This should be a significant improvement because it changes how the snake plays, not just how it scores.

---

### Iteration 23 — Late-Game Survival Intelligence

**Status:** TODO
**Depends on:** Iteration 19

**Goal:** When the board is filling up or we're partitioned, the game becomes about space efficiency and not running into our own tail. The current eval has a basic tail-chase bonus but no deeper understanding of confined play.

**Problem:** Late-game deaths are primarily from:
- Running out of space (territory too small for body length)
- Inefficient coiling (body wastes space by not following walls/edges)
- Health depletion when partitioned with no food

**New signals:**

1. **Space-to-length ratio**: Territory cells vs body length. Below 1.5x = danger. Below 1.0x = critical (guaranteed death soon without opponent dying).
   ```
   spaceRatio = float64(vr.MyTerritory) / float64(me.Length)
   if spaceRatio < 1.5:
       score -= wSpaceCrisis * lateBlend * (1.5 - spaceRatio) * 20.0
   ```

2. **Partition food planning**: When partitioned (`IsPartitioned`), food in our territory becomes survival-critical. Score based on food count relative to how many turns we can survive.
   ```
   if vr.IsPartitioned:
       turnsToStarve = me.Health  // 1 health/turn
       if vr.MyFood == 0:
           score -= wPartitionStarve * float64(max(100-turnsToStarve, 0)) / 100.0
       else:
           score += wPartitionFood * float64(vr.MyFood)
   ```

3. **Tail accessibility**: Beyond simple tail distance, check if our tail is actually *reachable* (not blocked by our own body). Use the Voronoi `dist[]` — if the tail cell is in our territory, it's reachable.
   ```
   tailIdx = tail.Y*width + tail.X
   if owner[tailIdx] == myTag:
       score += wTailReachable * lateBlend * 5.0
   else:
       score -= wTailBlocked * lateBlend * 3.0  // can't chase our own tail
   ```
   Note: This requires exposing tail reachability from the Voronoi workspace, either as a new VoronoiResult field or by checking `dist[]` for the tail cell.

4. **Opponent space crisis detection**: If the opponent is in a worse space crisis than us (their spaceRatio < ours), we're likely to outlast them. Bonus.
   ```
   oppSpaceRatio = float64(vr.OppTerritory) / float64(opp.Length)
   if spaceRatio > oppSpaceRatio:
       score += wOutlast * lateBlend * (spaceRatio - oppSpaceRatio) * 5.0
   ```

**Phase interaction:** All signals gated by `lateBlend` — they're irrelevant early/mid game. Partition signals additionally gated by `IsPartitioned`.

**Files:**
| File | Action |
|------|--------|
| `logic/eval.go` | Add late-game survival signals (tail reachability already in `VoronoiResult` from Iter 19) |
| `logic/eval_test.go` | Test space crisis, partition planning, tail reachability |

**Verify:** `make compare N=100` — target: >53%. Late-game improvements may not show huge numbers in winrate since many games are decided before late game.

---

### Iteration 24 — Weight Calibration

**Status:** TODO
**Depends on:** Iterations 20-23

**Goal:** Systematically tune all eval weights now that the eval has rich, meaningful signals from Iter 19-23. Many weights were set by intuition during development.

**Weights to tune (~15-18):**

| Category | Weights |
|----------|---------|
| Existing | wTerritory coefficients, wLen, wH2H, confinement (50/15), tail chase (3.0), food urgency (0.5) |
| Food strategy (Iter 20) | wFoodCluster, wFoodReach, wFoodDenial, wStarvationRisk |
| Positional (Iter 21) | ❌ Dead end — no weights to tune |
| Aggression (Iter 22) | dominance blend ratios, wHealthPressure, wPushToEdge, h2hRange scaling |
| Late-game (Iter 23) | wSpaceCrisis, wPartitionStarve, wPartitionFood, wTailReachable, wOutlast |

**Approach:**
1. One weight at a time: 2x it, 0.5x it, compare N=100 against current best
2. Start with new signals (never tuned) — most likely to have wrong magnitudes
3. If >55%, keep. If <50%, revert. If 50-55%, noise — skip.
4. After individual tuning, test 2-3 "theme" combinations (e.g., all aggression weights up 50%)
5. ~20-25 compare runs total

**Files:**
| File | Action |
|------|--------|
| `logic/eval.go` | Adjust weights based on A/B results |

---

### Iteration 25 — Territory Shape Quality (Optional)

**Status:** TODO
**Depends on:** Iteration 19

**Goal:** Detect corridor-shaped territory (many thin cells with ≤1 owned neighbor) and penalize it. This was the original Iter 18 plan from the old roadmap.

**Why optional / last:** The other iterations (20-24) target higher-impact strategic signals. Corridor detection requires an extra scan of the owner array (checking 4 neighbors per owned cell) with uncertain ROI. If Iter 20-24 already achieve strong results, this may not be worth the eval cost.

**Approach:**
1. After Voronoi BFS, scan owned cells: count cells with ≤1 neighbor also owned by us ("thin cells")
2. `corridorRatio = thinCells / myTerritory`
3. `score -= wCorridor * corridorRatio * lateBlend` — penalize corridor shapes, more in late game

**Risk:** If thin-cell counting adds >200ns to Voronoi, it may not be worth the eval cost. Alternatively, `MyTerritoryDepth` from Iter 19 may already capture this (deep territory ≈ compact territory).

**Files:**
| File | Action |
|------|--------|
| `logic/voronoi.go` | Add thin-cell count to result loop |
| `logic/eval.go` | Add corridor penalty |

**Verify:** `make compare N=100`. If <50%, this is a dead end — note in ENGINE.md and skip.

---

## Snapshot Log (Active)

Continues from ROADMAP_FINISHED.md snapshot log.

| Iteration | Snapshot | Avg Turns | Notes |
|-----------|----------|-----------|-------|
| 19 | | | Voronoi strategic extraction (infra) |
| 20 | `snapshots/haruko-a989fbb` | ~443 | Food strategy signals; 54% vs v19 |
| 21 | — | — | ❌ Dead end (37–48%) |
| 22 | | | Opponent pressure & aggression mode |
| 23 | | | Late-game survival intelligence |
| 24 | | | Weight calibration |
| 25 | | | Territory shape quality (optional) |
