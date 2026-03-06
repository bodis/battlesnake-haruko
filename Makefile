BINARY   := haruko
PORT     := 8080
BS       := go tool battlesnake

.PHONY: build run play local clean trace analyze

## build: compile the snake server binary
build:
	go build -o $(BINARY) .

## run: start the snake server on $(PORT)
run: build
	PORT=$(PORT) ./$(BINARY)

## play: run a local 1v1 game in the terminal (server must already be running)
play:
	$(BS) play \
		--name 'Haruko' --url http://localhost:$(PORT) \
		--name 'Haruko2' --url http://localhost:$(PORT) \
		-W 11 -H 11 --browser

## local: build, start server in background, run a game, then shut down
local: build
	@echo "Starting snake server on port $(PORT)..."
	@PORT=$(PORT) ./$(BINARY) & echo $$! > .snake.pid
	@sleep 1
	@$(BS) play \
		--name 'Haruko' --url http://localhost:$(PORT) \
		--name 'Haruko2' --url http://localhost:$(PORT) \
		-W 11 -H 11 || true
	@echo "Stopping snake server..."
	@kill $$(cat .snake.pid) 2>/dev/null && rm -f .snake.pid || true

## bench N=10: self-play N games, report win/draw/turn stats
bench: build
	@PORT=$(PORT) ./$(BINARY) & echo $$! > .snake.pid
	@sleep 0.5
	go run ./cmd/bench -games $(or $(N),10) -a-name Haruko -b-name Haruko \
		-a-url http://localhost:$(PORT) -b-url http://localhost:$(PORT)
	@kill $$(cat .snake.pid) 2>/dev/null && rm -f .snake.pid || true

## snapshot: save current binary to snapshots/ tagged by git hash
snapshot: build
	@mkdir -p snapshots
	cp $(BINARY) snapshots/$(BINARY)-$$(git rev-parse --short HEAD)
	@echo "Saved snapshots/$(BINARY)-$$(git rev-parse --short HEAD)"

## compare PREV=snapshots/haruko-abc N=50: current vs a previous snapshot
compare: build
	@test -n "$(PREV)" || (echo "Usage: make compare PREV=snapshots/haruko-<hash>"; exit 1)
	@PORT=$(PORT) ./$(BINARY) & echo $$! > .snake.pid
	@PORT=8081 $(PREV) & echo $$! >> .snake.pid
	@sleep 0.5
	go run ./cmd/bench -games $(or $(N),50) \
		-a-name "current" -a-url http://localhost:$(PORT) \
		-b-name "prev"    -b-url http://localhost:8081
	@kill $$(cat .snake.pid) 2>/dev/null && rm -f .snake.pid || true

## trace N=10: self-play with tracing enabled
trace: build
	@mkdir -p traces
	@HARUKO_TRACE=1 PORT=$(PORT) ./$(BINARY) & echo $$! > .snake.pid
	@sleep 0.5
	go run ./cmd/bench -games $(or $(N),10) -a-name Haruko -b-name Haruko \
		-a-url http://localhost:$(PORT) -b-url http://localhost:$(PORT)
	@kill $$(cat .snake.pid) 2>/dev/null && rm -f .snake.pid || true

## analyze MODE=summary: run analysis on trace files
analyze:
	go run ./cmd/analyze -mode $(or $(MODE),summary) traces/*.jsonl

## clean: remove build artifacts
clean:
	rm -f $(BINARY) .snake.pid
