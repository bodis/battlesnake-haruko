# Claude context — haruko

Battlesnake AI in Go. Module: `github.com/bodist/haruko`. Server port: 8080.
See [ENGINE.md](ENGINE.md) for full architecture, eval signals, version history, and dead ends.

## Key files
- `main.go` — HTTP handlers, `GameSession` map, API→logic type bridge
- `logic/sim.go` — `GameSim`: `CloneFromPool`/`Release`, `Step(MoveSet)`, `IsOver`
- `logic/eval.go` — `Evaluate(g, myIdx)`, `isSafeDir`, `safeMoveCount`
- `logic/voronoi.go` — `VoronoiTerritory(g, myIdx) VoronoiResult`
- `logic/search.go` — `BestMoveIterative(myID, budget)`: BRS + iterative deepening
- `logic/zobrist.go` — Zobrist hashing
- `logic/tt.go` — transposition table
- `logic/types.go` — `Coord`, `Direction`, `MoveSet`, `MaxSnakes=4`
- `logic/diagnostic.go` — `EscapeReachability` (allocating, trace-only)
- `logic/bench_test.go` — microbenchmarks
- `trace.go` — JSONL trace recording (diagnostic fields included)
- `cmd/analyze/main.go` — trace analysis (8 modes: summary, turning-points, deaths, wins, signals, trajectories, correlation, decision-points, survival)

## Dev workflow
- `make local` — build → start → 1v1 self-game → stop
- `make bench [N=10]` — self-play (turns are the metric)
- `make snapshot` / `make compare PREV=... [N=50]` — A/B comparison
- Rules CLI: `go tool battlesnake` (project-scoped, never `go install`)

## Iteration completion workflow
When finishing an iteration:
1. Run `make compare` to verify improvement
2. Move the completed iteration section from `ROADMAP.md` to `ROADMAP_FINISHED.md`
3. Update `ENGINE.md` — version history table, eval signals, and dead ends (if failed)
4. Update "Current State" in both `ROADMAP.md` and this file
5. Commit all doc changes with the iteration code

## Conventions
- Logic package must not import main. API types convert via `coordsToLogic()`/`snakesToLogic()`.
- Hot path must be zero-alloc. Use `CloneFromPool`/`Release`, stack arrays, `sync.Pool`.
- All dev tools project-scoped via `go get -tool` + `go tool <name>`.
- Board sizes: 7x7, 11x11, 19x19 all supported. `maxBoardCells=361`. Loops use `Width*Height`, no 11x11 cost.

## Current state (Iter 35 dead end, v32 current, Iter 36 next)
v32: Territory connectivity signal (absolute MyConnectivity). 56-61% vs v31. Iter 33-35 dead ends. Iter 35: tail reachability/loopability signals are lagging indicators — deaths are instantaneous 1-turn territory collapses. See ROADMAP.md for details.

## Direction
Iter 36: MC strategic rollout (diagnostic) — random game rollouts to detect long-horizon death traps. Iter 37: Survival mode (longest path in confined space). See ROADMAP.md.

## Go LSP (gopls)
`gopls` v0.21.1 at `/Users/bodist/go/bin/gopls`. Use for type checking (`gopls check`), references, definition lookup, rename, hover, symbols.
