# Claude context — haruko

Battlesnake AI in Go. Module: `github.com/bodist/haruko`. Server port: 8080.

## Key files
- `main.go` — `info/start/end/move` handlers, `GameSession` map (mutex-protected with `sync.RWMutex`)
- `logic/sim.go` — `GameSim`: full game state simulator with `Clone`, `CloneFromPool`/`Release`, `Step(MoveSet)`, `MoveSnakes(MoveSet)`, `IsOver`
- `logic/eval.go` — `Evaluate(g, myIdx int)`: composite eval (Voronoi territory + length advantage + h2h pressure + opponent confinement + food urgency)
- `logic/voronoi.go` — `VoronoiTerritory(g, myIdx int)`: multi-source BFS territory counting (workspace pooled)
- `logic/search.go` — `BestMoveIterative(myID, budget)`: iterative deepening with time management; index-based BRS + paranoid minimax; all hot-path cloning via sync.Pool
- `logic/zobrist.go` — `GameSim.Hash()`: Zobrist hashing (snake bodies + food)
- `logic/tt.go` — `TranspositionTable`: probe/store with generation-based invalidation
- `logic/types.go` — shared types: `Coord`, `Snake`, `Direction`, `AllDirections`, `MaxSnakes`, `MoveSet`
- `logic/bench_test.go` — microbenchmarks for Clone, Step, Evaluate, Voronoi, BRS node
- `Makefile` — `make local` is the main dev loop (build → start server → 1v1 self-game → stop)

## Rules CLI (game engine)
Tracked as a project tool in `go.mod` (`tool github.com/BattlesnakeOfficial/rules/cli/battlesnake`).
Run via `go tool battlesnake` — no global install, nothing in `~/go/bin`, no `$PATH` changes.

**Policy: keep all tooling project-scoped.** Never suggest `go install` for dev tools — use `go get -tool` instead and invoke with `go tool <name>`.

## Type bridge (main.go)
API types (`Coord`, `Battlesnake` in `models.go`) are converted to `logic.Coord` / `logic.Snake` via `coordsToLogic()` and `snakesToLogic()` before being passed to `GameSim`. Keep this pattern when adding new logic-package functions — the logic package must not import main.

## Ports
- `:8080` — current snake (all normal targets)
- `:8081` — previous snapshot (`make compare`)

## Current state (Iter 14)
Perf optimization: zero-alloc hot path. `MoveSet` replaces `map[string]Direction`, index-based search/eval/voronoi (no `SnakeByID` in hot path), `sync.Pool` for `GameSim` clones (4.4x faster, 0 allocs), pooled Voronoi workspace, stack-allocated arrays in `Step`. BRS node cost: ~1.1µs/0 allocs. 56% vs Iter 12 (N=100).

## Bench / version comparison
- `make bench [N=10]` — self-play; turns are the meaningful metric (A/B split is noise)
- `make snapshot` / `make compare PREV=... [N=50]` — version diff on :8080 vs :8081
- `-save FILE` flag writes JSONL: `{"n":1,"winner":"A","turns":42,"seed":123}` — seed replays exact game with `--seed`
- Speed: ~100 games in 4s with 16 workers; all local, no network overhead

Baselines (self-play avg turns): v1 ~68, v5 ~87, v6 ~328, v8 ~330, v9 ~306, v10 ~417, v11 ~197, v12 ~213, v13 ~200, v14 ~215.
`make bench` manages the server lifecycle automatically; `go run ./cmd/bench` requires a server already running on the target port.

**Note:** Paranoid minimax (retained in `BestMove`) degrades at depth 7+. BRS in `BestMoveIterative` breaks this ceiling.

**Next:** Iter 15 — search pruning/extensions (LMR, null move pruning, volatile position extensions). Then endgame detection (Iter 16), parameter tuning (Iter 17), QS retry (Iter 18 — precondition met: Clone+Step now ~4x cheaper).

## Failed experiments (do NOT retry without new preconditions)

### Quiescence search at BRS leaves (Iter 13)
Tried wiring `qsMax`/`qsMin` into `brsMax`/`brsMin` depth-0 returns. Tested multiple configurations — all performed ≤50% vs Iter 12 baseline (N=100 each):

| Config | Win rate vs Iter 12 |
|--------|-------------------|
| qsMaxDepth=2, isQuiet dist≤2, safeMoves≤1 | 41% |
| qsMaxDepth=1, same triggers | 51% |
| qsMaxDepth=1, tight triggers (dist≤1, safeMoves==0) | 48% |
| qsMin: all 4 opp dirs when no forcing moves | 48% |
| Eval hardening only (no QS at all) | 45% |

**Root cause:** Clone+Step+Evaluate per QS node is too expensive relative to the 300ms budget. Each QS extension costs the same as a regular BRS ply, so QS steals depth from the main search. The tactical benefit of resolving horizon effects doesn't compensate for the lost main-search depth.

**Preconditions to retry:**
1. ~~Clone+Step must become significantly cheaper (sync.Pool, arena allocation, or bitboard sim) — Iter 14 perf work~~ **MET in Iter 14**: CloneFromPool 4.4x faster (19ns/0 allocs), Step 0 allocs. BRS node ~1.1µs total.
2. Or: QS must avoid Clone+Step entirely (incremental move/unmove on the same GameSim)
3. Or: Budget must increase well beyond 300ms (different game mode / hardware)

**What to keep:** The `isQuiet`, `forcingMoves`, `safeMoveCount` helpers are useful independent of QS — `safeMoveCount` is already used by `Evaluate()`. Consider using `isQuiet` for search extensions (extend BRS depth by 1 in volatile positions) as a lighter alternative to full QS.

## Go LSP (gopls)
`gopls` v0.21.1 is available at `/Users/bodist/go/bin/gopls`. Use it when appropriate:
- **Type checking / diagnostics:** `gopls check <file.go>` — catch errors before building
- **Find references:** `gopls references <file.go>:#offset` — where is a symbol used
- **Definition lookup:** `gopls definition <file.go>:#offset` — jump to declaration
- **Rename:** `gopls rename <file.go>:#offset NewName` — safe cross-file rename
- **Hover / signature:** `gopls hover <file.go>:#offset` — type info, doc strings
- **Symbols:** `gopls symbols <file.go>` — list all declared symbols in a file
- **Workspace symbols:** `gopls workspace_symbol <query>` — search symbols across the module
