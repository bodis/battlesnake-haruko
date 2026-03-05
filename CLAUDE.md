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

## Conventions
- Logic package must not import main. API types convert via `coordsToLogic()`/`snakesToLogic()`.
- Hot path must be zero-alloc. Use `CloneFromPool`/`Release`, stack arrays, `sync.Pool`.
- All dev tools project-scoped via `go get -tool` + `go tool <name>`.

## Current state (Iter 17)
Phase-adaptive eval with continuous blend weights. BRS depth up to 14. Evaluate: ~1090ns/0 allocs. BRS node: ~1.1µs/0 allocs. Self-play avg ~451 turns.

## Direction
Search mechanics are saturated (pruning, ordering, QS all failed — see ENGINE.md dead ends). The remaining lever is **eval quality**. Next improvements should add new eval signals or refine existing weights.

## Go LSP (gopls)
`gopls` v0.21.1 at `/Users/bodist/go/bin/gopls`. Use for type checking (`gopls check`), references, definition lookup, rename, hover, symbols.
