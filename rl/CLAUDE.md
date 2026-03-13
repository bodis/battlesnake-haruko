# Claude context — rl/

Reinforcement learning training pipeline for Battlesnake Haruko.
Python (uv-managed) + Go gRPC env server.

## Architecture

```
Python (rl/)                    mmap + gRPC              Go (cmd/rlenv/)
┌────────────────┐          ┌──────────────┐          ┌─────────────────┐
│ SB3 PPO        │◄─mmap───►│ /tmp/haruko  │◄─mmap───►│ EnvManager      │
│ CNN extractor  │          │ -rl-shm      │          │ N×GameSim       │
│ MPS device     │◄─gRPC───►│ Configure    │◄─gRPC───►│ shm.go          │
└────────────────┘          │ Reset        │          │ observation.go  │
                            │ StepSignal   │          │ reward.go       │
                            │ Step (compat)│          │ food_spawn.go   │
                            └──────────────┘          │ opponent.go     │
                                                      └─────────────────┘
```

## Key files

### Go server (`cmd/rlenv/`)
- `main.go` — gRPC server, flags: `-port 50051`, `-num-envs 256`, `-shm-path /tmp/haruko-rl-shm`
- `env_manager.go` — `EnvManager`, parallel `Step()`/`StepSignal()`, auto-reset, mmap obs writing
- `shm.go` — mmap shared memory: layout, create/close, read/write helpers
- `observation.go` — head-centered 21×21 spatial (11ch CHW), 8 scalars, action mask, `correctAction()`
- `reward.go` — solo rewards (death -1, alive +0.01, hunger gradient) and competitive rewards
- `food_spawn.go` — 15% chance/turn, min 1
- `opponent.go` — `RandomPolicy`, `BRSPolicy`
- `envpb/` — generated protobuf code

### Python (`rl/`)
- `env/battlesnake_env.py` — `BattlesnakeVecEnv` (SB3 `VecEnv`, mmap + gRPC fallback)
- `model/features.py` — `BattlesnakeFeatureExtractor` (CNN, MPS-compatible, no AdaptiveAvgPool)
- `model/network.py` — `BattlesnakeActorCritic` (standalone, for inference server)
- `training/train.py` — PPO entry point, checkpointing, resume, TensorBoard
- `inference/server.py` — Battlesnake HTTP API with trained model
- `proto/` — protobuf definition + generated Python stubs

## Observation space
- **Spatial:** 11 channels × 21 × 21 (head-centered, walls as OOB channel)
  - Ch 0-6: wall, our body, our head, opp body, opp head, food, hazard
  - Ch 7-10: body ages (normalized), stacking flags
- **Scalars:** 8 values (health, length, opp health/length, turn, board size, food count — all normalized)
- **Action mask:** 4 bools, blocks 180° reversal + unsafe dirs

## Dev workflow
```bash
# Terminal 1: Start Go env server
cd <repo-root> && go run ./cmd/rlenv/ -port 50051 -num-envs 256

# Terminal 2: Train
cd rl/
uv run python training/train.py --device mps --num-envs 256 --name solo_v1

# Resume from checkpoint
uv run python training/train.py --device mps --num-envs 256 --resume checkpoints/solo_v1/final_model.zip

# Run inference server
uv run python inference/server.py --model checkpoints/solo_v1/final_policy.pt --port 8082
```

Or use `make` targets: `make server`, `make train-solo`, `make serve MODEL=...`

## Model saving — 3 formats
1. **`final_model.zip`** — Full SB3 checkpoint (optimizer, replay buffer). `PPO.load()` to resume.
2. **`final_policy.pt`** — PyTorch state dict + metadata dict. For inference server.
3. **`battlesnake_*_steps.zip`** — Periodic checkpoints every 50K steps.

## Performance (M4 Max)
- Env stepping: ~200K fps (mmap), was ~4100 fps (gRPC protobuf) — 50x improvement
- Training throughput: ~408 steps/s (MPS forward/backward is now the bottleneck)
- ~250K steps per 10-minute training chunk
- 256 envs, 2048 n_steps, 256 batch_size
- v0.1 baseline: 146 avg turns after 500K steps (old reward, gRPC)

## Conventions
- All Python runs via `uv run` from `rl/` directory
- Proto stubs regenerate with `make proto`
- Go server imports `logic/` directly (same Go module)
- `logic.IsSafeDir()` and `logic.SafeMoveCount()` are exported wrappers added for this
- Server-side `correctAction()` prevents instant death from untrained policies
