# haruko

A [Battlesnake](https://play.battlesnake.com) AI written in Go.

## Requirements

- Go 1.24+

The rules engine (game simulator) is tracked as a project tool in `go.mod` — no global install needed. Run it with `go tool battlesnake`.

## Run locally

```bash
git clone git@github.com:bodist/battlesnake-haruko.git
cd battlesnake-haruko
make local        # builds, starts server, runs a 1v1 self-game, stops server
```

Other targets:

```bash
make run          # just start the server on :8080
make play         # run a game against an already-running server
make build        # compile only
```

## Testing

`cmd/bench` runs N games in parallel and reports win rates and turn statistics.
~100 games take about 4 seconds on a modern machine (16 workers, all local).

```bash
make bench          # 10 self-play games, quick sanity check
make bench N=100    # 100 games
```

To save per-game records for later inspection, start the server first then run the bench tool directly:

```bash
make run   # terminal 1 — keeps server running

# terminal 2
go run ./cmd/bench -games 100 -workers 8 \
  -a-name A -a-url http://localhost:8080 \
  -b-name B -b-url http://localhost:8080 \
  -save results.jsonl
```

Each line in the file is a JSON record:

```json
{"n":3,"winner":"B","turns":94,"seed":1772574547356317000}
```

Quick ways to find interesting games in the file:

```bash
grep '"winner":"draw"' results.jsonl          # all draws
jq -s 'sort_by(.turns) | first' results.jsonl # shortest game
jq -s 'sort_by(.turns) | last'  results.jsonl # longest game
```

The `seed` field lets you replay any specific game exactly in the browser:

```bash
go tool battlesnake play \
  --name A --url http://localhost:8080 \
  --name B --url http://localhost:8080 \
  -W 11 -H 11 --seed 1772574547356317000 --browser
```

### Version comparison

```bash
make snapshot                              # save current binary → snapshots/haruko-<hash>
make compare PREV=snapshots/haruko-abc N=100   # current vs previous, 100 games
```

## Project structure

```
main.go          # snake logic + per-game session map
models.go        # Battlesnake API types
server.go        # HTTP server
logic/board.go   # FastBoard — 1D uint8 grid for fast state representation
cmd/bench/       # parallel game runner + stats reporter
snapshots/       # versioned binaries for comparison (not committed)
Makefile
```
