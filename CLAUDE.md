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
- `cmd/analyze/main.go` — trace analysis (7 modes: summary, turning-points, deaths, wins, signals, trajectories, correlation, decision-points)

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

## Current state (Iter 33 planned, Iter 32 current)
v32: Territory connectivity signal (absolute MyConnectivity). 56-61% vs v31. Diagnostic infrastructure added: VoronoiResult depth profile, EscapeReachability, trace diagnostic fields, analyze correlation + decision-points modes. Iter 33 planned: escape routes + far territory eval signals.

## Direction
Iter 33: Add escape routes + far territory signals to eval. Data shows eval is blind to positional collapse (92% detectable at search leaf but not scored). Test vs v32, v31, v28. See ROADMAP.md for full plan + data.

## Go LSP (gopls)
`gopls` v0.21.1 at `/Users/bodist/go/bin/gopls`. Use for type checking (`gopls check`), references, definition lookup, rename, hover, symbols.
