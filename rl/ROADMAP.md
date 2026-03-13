# RL Training Pipeline — Roadmap

## Current state (v0.1 — solo survival, gRPC)
- Pipeline working end-to-end: Go gRPC server → Python PPO (SB3) → model checkpoints
- MPS acceleration: ~4100 fps on M4 Max with 256 envs
- Solo survival: 146 avg turns after 500K steps (not converged)
- Model saving: SB3 .zip (resume) + PyTorch .pt (inference) + periodic checkpoints

---

## Iter 1: Replace gRPC with shared memory (mmap)

**Problem:** gRPC serialization is the main throughput bottleneck. Each step serializes 256 × 11 × 21 × 21 = 1.27M floats as protobuf. Encoding/decoding is single-threaded and accounts for most of the per-step wall time. The Go simulation itself is ~49ns/step — the gRPC overhead dwarfs actual compute.

**Solution:** Memory-mapped shared buffers for observation data. Keep gRPC only for lightweight control messages.

### Design

```
Shared memory layout (single mmap file, ~10MB):
┌─────────────────────────────────────────────────────────┐
│ Header (64 bytes)                                        │
│   magic, version, num_envs, spatial_shape, num_scalars   │
│   step_counter (atomic), ready_flag                      │
├─────────────────────────────────────────────────────────┤
│ Observation buffer (per env, contiguous):                 │
│   spatial: float32[11 × 21 × 21] = 19,404 bytes         │
│   scalars: float32[8] = 32 bytes                         │
│   action_mask: bool[4] = 4 bytes                         │
│   reward: float32 = 4 bytes                              │
│   done: bool = 1 byte                                    │
│   padding to 32-byte align                               │
│ Total per env: ~19,456 bytes (19KB)                      │
│ Total 256 envs: ~4.9MB                                   │
├─────────────────────────────────────────────────────────┤
│ Action buffer (Python → Go):                             │
│   actions: int32[256] = 1KB                              │
└─────────────────────────────────────────────────────────┘
```

### Flow
1. Python writes actions into action buffer in mmap
2. Python signals Go via a lightweight gRPC `StepSignal(step_id)` call (no payload)
3. Go reads actions from mmap, steps all envs in parallel, writes obs/rewards/dones to mmap
4. Go responds to gRPC with `StepDone(step_id)` (no payload)
5. Python reads observations directly via numpy memmap — zero copy, zero deserialization

### Implementation
- Go side: `mmap` the shared file, write observations as raw float32/bool in fixed layout
- Python side: `np.memmap` or `mmap` + `np.frombuffer` for zero-copy reads
- Fallback: keep current gRPC-with-payload mode for debugging/compatibility
- File: `/tmp/haruko-rl-shm` (or configurable path)

### Expected impact
- Eliminates ~80% of per-step overhead (protobuf encode/decode)
- Should push from ~4K fps to 10-20K fps
- Only ~1KB over gRPC per step (signal + ack) vs 5MB currently

### Alternative: cgo shared library
Compile Go `logic/` as C shared library (`go build -buildmode=c-shared`), call from Python via ctypes. Eliminates network layer entirely. Tradeoffs:
- **Pro:** Zero overhead, no IPC at all, single process
- **Con:** cgo build complexity, harder to debug, Go GC pauses affect Python, can't scale Go server independently
- **Verdict:** mmap is simpler, nearly as fast, and preserves the clean process separation. Try mmap first, cgo only if mmap isn't enough.

---

## Iter 2: Solo reward function redesign

**Problem:** Current reward function has structural biases that hurt learning:

1. **Food reward (+0.5) is too strong.** 50× the per-turn survival reward. Agent learns short aggressive food-chasing lives that are net-positive (+10 food - 10 death = profitable). Food pursuit in confined late-game spaces makes the snake *weaker* — longer body = less room to maneuver.
2. **Death penalty (-10.0) dwarfs value function.** With gamma=0.99, cumulative survival value is ~1.0. Death at -10.0 means the value head only learns "will I die?" not "what should I do?"
3. **Survive-500 bonus is unreachable.** Agent averages 146 turns — never sees the +10 bonus. Unreachable rewards create no learning signal.
4. **Low health penalty (-0.1 at health=0) is invisible.** Triggers too late (health < 20), too weak relative to food reward.

**Key insight from BRS engine (Iter 24 weight calibration):** Food/growth is important early game (H2H collision advantage) but neutral-to-harmful late game (body fills confined space). The reward shouldn't encourage eating — it should make hunger painful enough that eating emerges as survival behavior.

### New solo reward function

```go
func computeRewardSolo(g, prevG, myIdx) float32 {
    if !me.IsAlive() {
        return -1.0                           // death: ~100 turns of survival value
    }
    reward := 0.01                            // alive: main signal
    if me.Health < 30 {
        reward -= 0.005 * float32(30 - health) // hunger: max -0.15/turn at health=0
    }
    return reward
}
```

Three signals only:
- **Death (-1.0):** Reduced from -10. Now proportional to ~100 turns of survival, giving the value head a smooth learning surface instead of a cliff.
- **Per turn alive (+0.01):** Unchanged. This IS the survival signal — 500 turns = +5.0 cumulative.
- **Health < 30 (-0.005 × deficit):** Hunger pressure. Triggers at 30 (not 20) — gives agent time to react. Continuous gradient. At health=5: -0.125/turn. Agent learns to eat because hunger hurts, not because eating is rewarded.

**No food reward.** Agent discovers eating as a survival behavior from hunger pressure alone. Won't chase food into dangerous corridors when health is fine.

**No survival bonus.** Per-turn reward already incentivizes survival. No unreachable carrots.

### Verification
- Run 2M+ steps post-mmap, compare avg turns vs v0.1 (146 turns baseline)
- TensorBoard: episode length curve should climb faster with cleaner reward
- Watch for: agent learning to eat (health stays high) without food reward signal

---

## Iter 3: Longer solo training + convergence

- Run 5-10M timesteps with mmap + new rewards
- Target: 300+ avg turns in solo mode
- Tune hyperparams: lr schedule, entropy coefficient, clip range
- Add TensorBoard monitoring for episode length, reward curves
- If training stalls, try curriculum: start on 7×7, then 11×11

---

## Iter 4: Opponent training phases

### Phase 4a: vs Random
- Enable `--opponent random`
- Random opponent picks safe moves (already implemented in Go server)
- Target: >70% win rate vs random

### Phase 4b: vs BRS engine
- Enable `--opponent brs --brs-budget-ms 50`
- BRS opponent uses the Go BRS search engine (cloned game state)
- Start with 50ms budget (weaker), increase to 300ms (full strength)
- Target: competitive with BRS engine
- **Competitive reward: death -1, kill +1, alive +0.01 only.** No territory/confinement shaping — let RL discover its own strategy rather than mimicking BRS eval.

### Phase 4c: Self-play
- Train against pool of past model versions
- Rotate opponents every N steps
- Requires saving/loading opponent policy checkpoints
- Most complex phase — defer until vs-BRS works

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
