# Claude context — haruko

Battlesnake AI in Go. Module: `github.com/bodist/haruko`. Server port: 8080.

## Key files
- `main.go` — `info/start/end/move` handlers, `GameSession` map (mutex-protected with `sync.RWMutex`)
- `logic/sim.go` — `GameSim`: full game state simulator with `Clone`, `Step`, `MoveSnakes`, `IsOver`
- `logic/eval.go` — `Evaluate(g, myID)`: composite eval (Voronoi territory + length advantage + h2h pressure + opponent confinement + food urgency)
- `logic/voronoi.go` — `VoronoiTerritory(g, myID)`: multi-source BFS territory counting
- `logic/search.go` — `BestMoveIterative(myID, budget)`: iterative deepening with time management; paranoid minimax with alpha-beta pruning
- `logic/types.go` — shared types: `Coord`, `Snake`, `Direction`, `AllDirections`
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

## Current state (Iter 9)
Iterative deepening with 300ms time budget, max depth 5. Searches depth 1, 2, 3, ... within budget, always has a valid move from at least depth 1. Composite evaluation: Voronoi territory (dominant), length advantage, head-to-head pressure, opponent confinement, and food urgency. 76% win rate vs Iter 8 (N=100).

## Bench / version comparison
- `make bench [N=10]` — self-play; turns are the meaningful metric (A/B split is noise)
- `make snapshot` / `make compare PREV=... [N=50]` — version diff on :8080 vs :8081
- `-save FILE` flag writes JSONL: `{"n":1,"winner":"A","turns":42,"seed":123}` — seed replays exact game with `--seed`
- Speed: ~100 games in 4s with 16 workers; all local, no network overhead

Baselines (self-play avg turns): v1 ~68, v5 ~87, v6 ~328, v8 ~330, v9 ~306.
`make bench` manages the server lifecycle automatically; `go run ./cmd/bench` requires a server already running on the target port.

**Next:** move ordering + killer heuristic, snake appearance tuning.

## Go LSP (gopls)
`gopls` v0.21.1 is available at `/Users/bodist/go/bin/gopls`. Use it when appropriate:
- **Type checking / diagnostics:** `gopls check <file.go>` — catch errors before building
- **Find references:** `gopls references <file.go>:#offset` — where is a symbol used
- **Definition lookup:** `gopls definition <file.go>:#offset` — jump to declaration
- **Rename:** `gopls rename <file.go>:#offset NewName` — safe cross-file rename
- **Hover / signature:** `gopls hover <file.go>:#offset` — type info, doc strings
- **Symbols:** `gopls symbols <file.go>` — list all declared symbols in a file
- **Workspace symbols:** `gopls workspace_symbol <query>` — search symbols across the module
