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

## Current state (Iter 20, Iter 21+22 dead ends)
Food strategy signals: distance-weighted food value, reach advantage, denial/starvation, growth urgency. BRS depth up to 14. Voronoi: ~1025ns/0 allocs. Evaluate: ~1130ns/0 allocs. BRS node: ~1.2µs/0 allocs. Self-play avg ~443 turns.

## Direction
Search mechanics are saturated (pruning, ordering, QS all failed — see ENGINE.md dead ends). The remaining lever is **eval quality** — but new signals must add genuinely new information, not restate what Voronoi already captures (Iter 21), and dominance-based weight modulation is ineffective in self-play (Iter 22). Next: late-game survival, weight calibration.

## Go LSP (gopls)
`gopls` v0.21.1 at `/Users/bodist/go/bin/gopls`. Use for type checking (`gopls check`), references, definition lookup, rename, hover, symbols.
