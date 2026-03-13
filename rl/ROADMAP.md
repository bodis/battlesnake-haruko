# RL Training Pipeline — Roadmap

## Current state (v0.3 — solo mode solved)
- Solo training converged: 500 avg turns (max), 7.3M total steps
- Best model: `checkpoints/solo_v2b/final_model.zip`
- Pipeline: Go gRPC server → mmap shared memory → Python PPO (SB3)
- Next: self-play training (Iter 4)

---

## Iter 4: Self-play training

Train against a frozen copy of the latest model. Only one snake is trained (agent); the opponent runs inference with the frozen model and does not update weights.

### Architecture

The Go server currently supports opponent types (none, random, brs) via `OpponentPolicy` interface. For self-play, the opponent's action must come from a Python-side frozen model inference, since the model lives in Python/PyTorch.

**Approach: extend mmap with opponent observations + actions**

1. **Go server (shm.go):** Add opponent spatial, scalars, mask, and action arrays to the mmap layout. Extend the header with new offsets.
2. **Go server (env_manager.go):** Add `oppSelf` type. When stepping with `oppSelf`:
   - Read opponent action from mmap (instead of calling `OpponentPolicy`)
   - After step + reset, write BOTH agent and opponent observations to mmap
3. **Go server (observation.go):** `computeObservation(g, oppIdx)` already works for any snake index — no changes needed.
4. **Python (battlesnake_env.py):** In self-play mode, map the new opponent mmap arrays. Expose opponent observations so the training loop can run the frozen model. Write opponent actions to mmap before `StepSignal`.
5. **Python (train.py):** Add `--opponent self --self-play-model <path>` flags. Load the frozen model (SB3 `PPO.load()` or raw `state_dict`). Each step:
   - Read opponent observations from mmap
   - Run frozen model forward pass → opponent action
   - Write opponent action to mmap
   - Normal PPO step for agent

### Reward

Simple competitive reward (no shaping):
- Agent death: **-1.0**
- Opponent death (kill): **+1.0**
- Per turn alive: **+0.01**

No territory, food, or confinement shaping — let RL discover its own strategy.

### Training plan

- Start from solo_v2b model (`checkpoints/solo_v2b/final_model.zip`)
- Opponent = frozen copy of same model
- 256 envs, MPS, 750K steps per round
- 3 rounds initially, check for learning signal (win rate, episode length changes)

### Success criteria

- Agent learns to win >55% vs the frozen opponent (asymmetric improvement)
- Episode length decreases from 500 (solo survival) as agent learns to kill

### What to watch

- Win rate (agent kills vs agent deaths) — main metric
- Episode length — should drop from 500 as combat emerges
- Reward trend — should climb above 0 (kills outweigh deaths)
- Policy collapse — if agent always dies or always ties, adjust reward or LR

---

## Iter 5: Action masking via MaskablePPO

- Switch from standard PPO to sb3-contrib's `MaskablePPO`
- Pass action_mask from observation to the policy
- Currently using server-side `correctAction()` as workaround — works but limits exploration
- MaskablePPO lets the agent learn the mask naturally via masked logit gradients
- `sb3-contrib` already installed in pyproject.toml

---

## Iter 6: Double-buffered inference/stepping

- While Python runs forward pass on batch N, Go prepares observations for batch N+1
- Two mmap observation buffers, swap on each step
- Overlaps GPU inference with env stepping
- Only matters if GPU inference becomes the bottleneck (currently gRPC is)

---

## Iter 7: Inference server + live comparison

- `inference/server.py` serves Battlesnake HTTP API with trained model
- Load `.pt` checkpoint, run PyTorch inference per move
- Compatible with `go tool battlesnake play` (browser visualization)
- Use `make compare` infrastructure for A/B testing RL vs BRS engine
- ONNX export for production deployment (smaller, faster)

---

## Future ideas
- **Larger model:** Current CNN is small (332K params). Deeper ResNet if training data is sufficient.
- **Multi-agent:** Train all snakes simultaneously (population-based training).
- **Hybrid:** Use RL policy for early/mid game, BRS for endgame (when search depth matters most).
- **Distillation:** Train a fast RL policy, then distill into the BRS eval function.
- **Self-play curriculum:** After Iter 4, periodically update the frozen opponent with the latest trained model (league training).
