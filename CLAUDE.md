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
- `logic/bench_test.go` — microbenchmarks

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

## Current state (Iter 23, Iter 21+22 dead ends)
Territory bottleneck detection (Tarjan's AP on territory subgraph). BRS depth ~12-13. Voronoi: ~2400ns/0 allocs. Evaluate: ~2450ns/0 allocs. BRS node: ~2.5µs/0 allocs. 58% vs v20.

## Direction
Eval quality > search depth. Tarjan's AP adds genuinely new structural info. Next: weight calibration (Iter 24), late-game survival signals.

## Go LSP (gopls)
`gopls` v0.21.1 at `/Users/bodist/go/bin/gopls`. Use for type checking (`gopls check`), references, definition lookup, rename, hover, symbols.
