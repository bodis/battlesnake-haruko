# Claude context — haruko

Battlesnake AI in Go. Module: `github.com/bodist/haruko`. Server port: 8080.

## Key files
- `main.go` — `info/start/end/move` handlers, `GameSession` map (mutex-protected with `sync.RWMutex`)
- `logic/board.go` — `FastBoard`: flat `[]uint8` 1D grid, `GetIndex(x,y)=y*Width+x`, cell constants `CellEmpty/Food/Hazard/Snake/MySnake`
- `Makefile` — `make local` is the main dev loop (build → start server → 1v1 self-game → stop)

## Rules CLI (game engine)
Tracked as a project tool in `go.mod` (`tool github.com/BattlesnakeOfficial/rules/cli/battlesnake`).
Run via `go tool battlesnake` — no global install, nothing in `~/go/bin`, no `$PATH` changes.

**Policy: keep all tooling project-scoped.** Never suggest `go install` for dev tools — use `go get -tool` instead and invoke with `go tool <name>`.

## Type bridge (main.go)
API types (`Coord`, `Battlesnake` in `models.go`) are converted to `logic.Coord` / `logic.Snake` via `coordsToLogic()` and `snakesToLogic()` before being passed to `FastBoard.Update()`. Keep this pattern when adding new logic-package functions — the logic package must not import main.

## Ports
- `:8080` — current snake (all normal targets)
- `:8081` — previous snapshot (`make compare`)

## Current state
Phase 1 done: server runs, FastBoard updates each turn, basic wall/body collision avoidance, random safe-move selection.

## Bench / version comparison
- `make bench [N=10]` — self-play; turns are the meaningful metric (A/B split is noise)
- `make snapshot` / `make compare PREV=... [N=50]` — version diff on :8080 vs :8081
- `-save FILE` flag writes JSONL: `{"n":1,"winner":"A","turns":42,"seed":123}` — seed replays exact game with `--seed`
- Speed: ~100 games in 4s with 16 workers; all local, no network overhead

Baseline (v1, random safe-move): avg ~68 turns.
`FastBoard.Update()` does a full cell reset every turn — acceptable now, worth revisiting if move generation becomes the bottleneck.
`make bench` manages the server lifecycle automatically; `go run ./cmd/bench` requires a server already running on the target port.

**Next:** flood-fill space counting → Minimax with alpha-beta pruning.

## Go LSP (gopls)
`gopls` v0.21.1 is available at `/Users/bodist/go/bin/gopls`. Use it when appropriate:
- **Type checking / diagnostics:** `gopls check <file.go>` — catch errors before building
- **Find references:** `gopls references <file.go>:#offset` — where is a symbol used
- **Definition lookup:** `gopls definition <file.go>:#offset` — jump to declaration
- **Rename:** `gopls rename <file.go>:#offset NewName` — safe cross-file rename
- **Hover / signature:** `gopls hover <file.go>:#offset` — type info, doc strings
- **Symbols:** `gopls symbols <file.go>` — list all declared symbols in a file
- **Workspace symbols:** `gopls workspace_symbol <query>` — search symbols across the module
