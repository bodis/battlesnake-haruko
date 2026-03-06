# Haruko Battlesnake — Development Roadmap

> Active development plan. Completed iterations are archived in [ROADMAP_FINISHED.md](ROADMAP_FINISHED.md).
> Each iteration: implement → test → snapshot → compare → merge → move to finished → update ENGINE.md.

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iterations 1-19 (see ROADMAP_FINISHED.md) |
| **Next** | Iteration 20 |
| **Current** | v19 Voronoi strategic extraction (infra); BRS depth 14; ~451 avg turns; Voronoi ~1025ns/0 allocs |
| **Key insight** | Search mechanics exhausted at BF=4. The remaining lever is **eval quality** — specifically, strategic board understanding that goes beyond snapshot scoring. |

---

## Phase 8: Strategic Board Understanding

> **The gap:** The current eval is a snapshot scorer — it measures "how good is this position right now"
> but has no strategic awareness. It can't reason about food routes, spatial opportunity, resource
> denial, or whether the current game phase demands growth vs aggression vs survival.
>
> The Voronoi BFS already computes `owner[]` and `dist[]` for every cell but we discard most of it.
> These iterations extract strategic signals from existing data and teach the eval to reason about
> the whole board, not just count cells.

### Iteration 20 — Food Strategy Signals

**Status:** TODO
**Depends on:** Iteration 19

**Goal:** Teach the eval to reason about food access quality, not just food count. "3 food nearby in my territory" should score very differently from "3 food scattered across the opponent's side."

**Problem:** Current food awareness is limited to:
- `vr.MyFood` count (how many food items in our territory — no distance weighting)
- `foodUrgency` (only activates below health threshold, only nearest single food)
- `1.5 * earlyBlend * vr.MyFood` (early-game food control — flat count, no quality)

A snake with 3 food at BFS distance 2-3 has a massive growth opportunity. A snake with 3 food at distance 8-10 effectively has nothing. The eval treats them the same.

**New signals:**

1. **Food cluster value** (`MyFoodValue` from Iter 19): Replace flat `vr.MyFood` in food control with distance-weighted value. Closer food = exponentially more valuable.
   ```
   score += wFoodCluster * earlyBlend * vr.MyFoodValue
   ```

2. **Food reach advantage**: When we can reach food faster than the opponent, that's a strategic asset regardless of health level.
   ```
   if vr.MyClosestFoodDist > 0 && vr.OppClosestFoodDist > 0:
       foodReachDelta = oppClosest - myClosest  // positive = we're closer
       score += wFoodReach * foodReachDelta
   ```

3. **Food denial**: When opponent has NO food in their territory (`OppFood == 0`) and their health is declining, we're winning the resource war. Conversely, when we have no food access, danger.
   ```
   if vr.OppFood == 0 && opp.Health < 40:
       score += wFoodDenial * (40 - opp.Health) / 40.0
   if vr.MyFood == 0 && me.Health < 50:
       score -= wStarvationRisk * (50 - me.Health) / 50.0
   ```

4. **Growth urgency**: Phase-aware signal that asks "am I long enough for this stage of the game?" Not just length delta vs opponent, but absolute adequacy.
   ```
   expectedLen = 3 + g.Turn/10  // rough: should be ~13 by turn 100
   if me.Length < expectedLen:
       growthNeed = earlyBlend * float64(expectedLen - me.Length)
       // boosts value of food access when we're behind schedule
   ```

**Phase interaction:** Food cluster + reach + denial strongest in early/mid game (earlyBlend and midgame). Growth urgency fades as lateBlend increases. Food denial relevant at all phases.

**Files:**
| File | Action |
|------|--------|
| `logic/eval.go` | Add food strategy signals, phase-weight them |
| `logic/eval_test.go` | Test food cluster scoring, denial detection, growth urgency |

**Verify:** `make snapshot` then `make compare PREV=<v19-snapshot> N=100` — target: >53%.

---

### Iteration 21 — Positional Quality

**Status:** TODO
**Depends on:** Iteration 19

**Goal:** Not all positions are equal even with the same territory count. A central position with deep territory is strategically dominant. An edge position with thin corridors is vulnerable. Teach the eval to distinguish.

**New signals:**

1. **Edge/corner vulnerability**: Head position near board edges has fewer escape routes. This is a permanent positional disadvantage independent of territory count.
   ```
   edgeDist = min(head.X, head.Y, width-1-head.X, height-1-head.Y)
   if edgeDist == 0: score -= wEdge * (1.0 - 0.5*earlyBlend)  // edges bad, less so early
   if edgeDist == 0 for both axes (corner): score -= wCorner
   ```
   Mirror for opponent: opponent near edge/corner = bonus.

2. **Territory depth adequacy**: `MyTerritoryDepth` (from Iter 19) tells us the longest path in our space. If it's shorter than our snake, we can't fit — death spiral. If it's much larger, we have room to maneuver.
   ```
   depthRatio = vr.MyTerritoryDepth / me.Length
   if depthRatio < 1.0: score -= wDepthCrisis * (1.0 - depthRatio)  // can't fit
   if depthRatio > 2.0: score += wDepthComfort * lateBlend          // plenty of room
   ```

3. **Center of mass advantage**: Territory centered near the board middle has more strategic flexibility than territory pushed to edges. Use centroids from Iter 19.
   ```
   boardCenter = (width-1)/2.0, (height-1)/2.0
   myCenterDist = manhattan(myCenter, boardCenter)
   oppCenterDist = manhattan(oppCenter, boardCenter)
   score += wCenterControl * (oppCenterDist - myCenterDist)  // we're more central = good
   ```

**Phase interaction:** Edge vulnerability matters most in mid/late game (when confrontation likely). Territory depth matters most in late game. Center control is a constant mild signal.

**Files:**
| File | Action |
|------|--------|
| `logic/eval.go` | Add positional quality signals |
| `logic/eval_test.go` | Test edge penalty, depth adequacy, center control |

**Verify:** `make compare N=100` — target: >53%.

---

### Iteration 22 — Opponent Pressure & Aggression Mode

**Status:** TODO
**Depends on:** Iteration 20, 21

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
| Positional (Iter 21) | wEdge, wCorner, wDepthCrisis, wDepthComfort, wCenterControl |
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
| 20 | | | Food strategy signals |
| 21 | | | Positional quality |
| 22 | | | Opponent pressure & aggression mode |
| 23 | | | Late-game survival intelligence |
| 24 | | | Weight calibration |
| 25 | | | Territory shape quality (optional) |
