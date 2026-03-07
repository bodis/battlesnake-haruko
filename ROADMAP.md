# Haruko Battlesnake — Development Roadmap

> Active development plan. Completed iterations are archived in [ROADMAP_FINISHED.md](ROADMAP_FINISHED.md).
> Each iteration: implement → test → snapshot → compare → merge → move to finished → update ENGINE.md.
> Development follows a data-driven loop: trace games → analyze outcomes → identify root causes → design targeted fixes → verify with A/B comparison.

---

## Current State

| Metric | Value |
|--------|-------|
| **Completed** | Iterations 1-20, 23-24 (see ROADMAP_FINISHED.md) |
| **Dead ends** | Iter 21 (positional quality), Iter 22 (aggression), Iter 25 (superseded by 23) |
| **Next** | Iteration 25 |
| **Current** | v24 Weight calibration; BRS depth ~12-13; Evaluate ~2450ns/0 allocs; 61% vs v23 |
| **Key insight** | After weight calibration, the engine's death/win patterns may have shifted. Before adding new features or optimizations, we need fresh data to guide the next move. |

---

## Phase 10: Data-Driven Analysis

> **The situation:** v24 changed 6 eval weights significantly (H2H nearly doubled, territory +50%, food weights reduced).
> This likely shifted how the engine wins and loses. Before building the next feature (phase-gating, new signals, etc.),
> we need to understand the current engine's behavior: what kills us, what kills the opponent, and where the biggest
> improvement opportunities lie. Analysis first, design second.

### Iteration 25 — Win/Loss Trace Analysis

**Status:** TODO
**Depends on:** Iteration 24

**Goal:** Comprehensive analysis of v24's game outcomes — both losses AND wins. Understand not just how we die, but how we win: what signals drive our victories, what opponent mistakes we exploit, and where our turning points happen. This informs whether the next iteration should be eval speed (phase-gating), new signals, or something else entirely.

**Step 1: Collect trace data**
```
rm -f traces/*.jsonl        # clear old traces
make trace N=20             # self-play with full eval tracing
```

Note: self-play traces capture BOTH perspectives (we play both sides), so N=20 games = 40 trace files. Each game has a winner and loser from both perspectives.

**Step 2: Loss analysis — how do we die?**
```
make analyze MODE=summary           # overall win/loss/draw + death causes
make analyze MODE=deaths -top 20    # detailed last-10-turns for each loss
make analyze MODE=turning-points    # largest eval swings (negative = our collapse)
```

Questions to answer:
- **Death cause distribution**: starvation vs head-collision vs body-collision vs wall? Which dominates?
- **Death phase**: do we die early (turn <50), mid (50-200), or late (200+)? Early deaths suggest opening weakness; late deaths suggest endgame weakness.
- **Pre-death pattern**: what signal collapses before death? Territory drop (getting cornered)? H2H loss (losing head-to-head)? Starvation (failing to find food)?
- **Preventability**: was there a turning point where eval swung negative? How many turns before death? Could deeper search have seen it?

**Step 3: Win analysis — how does the opponent die?**
```
make analyze MODE=signals           # signal averages in wins vs losses
make analyze MODE=turning-points    # largest eval swings (positive = our breakthrough)
```

Add a new `wins` analysis mode to `cmd/analyze/main.go`:
```
make analyze MODE=wins -top 20      # detailed last-10-turns before opponent dies
```

Questions to answer:
- **Win cause distribution**: do we kill opponents via territory strangulation, H2H domination, or do they self-destruct (walk into walls/bodies)?
- **Win phase**: early kills (aggressive H2H) vs late kills (territory squeeze)?
- **Winning signals**: which eval signals are strongest in wins but weakest in losses? These are our most effective weapons.
- **Opponent self-destruction**: how often does the opponent lose without us doing anything special? (e.g., opponent walks into our body, opponent starves in open space) — these wins don't teach us anything, but filtering them out shows our "real" win rate.
- **Turning points in wins**: when does the eval swing positive? What signal drives it? This shows our strategic "moment of advantage".

**Step 4: Synthesize findings**

Classify outcomes into actionable categories:
1. **Fixable losses** — deaths where a clear eval signal gap exists (territory collapse we didn't see, starvation we could've avoided)
2. **Unavoidable losses** — opponent played better, no signal gap (these set our ceiling)
3. **Active wins** — we created advantage through territory/H2H/food control
4. **Passive wins** — opponent self-destructed (wall, starvation, bad move)

Based on the distribution, decide next iteration:
- If >30% of losses have a fixable signal gap → new eval signal targeting that gap
- If losses are mostly unavoidable + wins are mostly active → phase-gate bottleneck (Iter 25-old) to get more depth
- If many wins are passive (opponent self-destructs) → our self-play winrate overstates real strength; consider testing against external opponents
- If a specific death phase dominates → target that phase (early: opening, late: endgame)

**Step 5: Build `wins` analysis mode**

Add `modeWins` to `cmd/analyze/main.go` — mirror of `modeDeaths` but for wins:
- Show last 10 turns before opponent death
- Track which signal was highest at the moment of victory
- Classify wins: "territory squeeze" (territory signal dominant), "H2H kill" (H2H + confinement dominant), "starvation kill" (food denial dominant), "self-destruct" (our eval was flat/negative before opponent died)

**Files:**
| File | Action |
|------|--------|
| `cmd/analyze/main.go` | Add `wins` mode, enhance `summary` with phase breakdown |
| `ROADMAP.md` | Document findings, decide next iteration |
| `ENGINE.md` | Update with analysis insights |

**Verify:** Qualitative — findings should clearly point to a next iteration, or confirm we're at a local optimum.

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
