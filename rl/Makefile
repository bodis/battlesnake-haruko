# RL Training Pipeline — Makefile
# All commands run from the rl/ directory.

GOENV_PORT ?= 50051
NUM_ENVS ?= 64
BOARD ?= 11
OPPONENT ?= none
TIMESTEPS ?= 1000000
DEVICE ?= auto
INFERENCE_PORT ?= 8082

.PHONY: proto server train train-solo train-vs-random train-vs-brs serve tensorboard clean

# Generate Python proto stubs.
proto:
	uv run python -m grpc_tools.protoc \
		--proto_path=proto \
		--python_out=proto \
		--grpc_python_out=proto \
		proto/battlesnake_env.proto

# Start the Go gRPC env server.
server:
	cd .. && go run ./cmd/rlenv/ -port $(GOENV_PORT) -num-envs $(NUM_ENVS)

# Generic train command.
train:
	uv run python training/train.py \
		--server-addr localhost:$(GOENV_PORT) \
		--num-envs $(NUM_ENVS) \
		--board-width $(BOARD) --board-height $(BOARD) \
		--opponent $(OPPONENT) \
		--total-timesteps $(TIMESTEPS) \
		--device $(DEVICE) \
		$(EXTRA_ARGS)

# Phase 1: Solo survival training.
train-solo:
	$(MAKE) train OPPONENT=none NAME=solo TIMESTEPS=$(TIMESTEPS)

# Phase 2: vs Random opponent.
train-vs-random:
	$(MAKE) train OPPONENT=random NAME=vs_random TIMESTEPS=$(TIMESTEPS)

# Phase 3: vs BRS engine.
train-vs-brs:
	$(MAKE) train OPPONENT=brs NAME=vs_brs TIMESTEPS=$(TIMESTEPS)

# Resume training from a checkpoint.
resume:
	uv run python training/train.py \
		--server-addr localhost:$(GOENV_PORT) \
		--num-envs $(NUM_ENVS) \
		--board-width $(BOARD) --board-height $(BOARD) \
		--opponent $(OPPONENT) \
		--total-timesteps $(TIMESTEPS) \
		--device $(DEVICE) \
		--resume $(CHECKPOINT)

# Run inference server with a trained model.
serve:
	uv run python inference/server.py \
		--model $(MODEL) \
		--port $(INFERENCE_PORT) \
		--device $(DEVICE)

# TensorBoard.
tensorboard:
	uv run tensorboard --logdir logs/

# Clean generated files and checkpoints.
clean:
	rm -rf checkpoints/ logs/ proto/*_pb2*.py proto/__pycache__/
