# RL Training Pipeline — Finished Iterations

## Iter 1: Replace gRPC with shared memory (mmap)

Implemented mmap shared memory (`/tmp/haruko-rl-shm`). Layout: separate contiguous arrays (spatial, scalars, masks, rewards, dones, actions) with 64-byte header containing offsets. Python reads header, creates numpy views, zero-copy reads. gRPC `StepSignal` RPC carries only step_id. Old `Step` RPC preserved as fallback.

---

## Iter 2: Solo reward function redesign

Simplified to 3 signals: death (-1.0), alive (+0.01/turn), hunger gradient (-0.005 x deficit when health < 30). Removed food reward, death cliff (-10 to -1), survival bonus, and late health penalty.

---

## Iter 3: Solo training with mmap + new rewards

**Setup:** 256 envs, MPS, mmap mode, 750K steps per round.

**Results (solo_v2b, 6 rounds from Round 2 peak checkpoint):**

| Round | Avg Turns (last iter) | vs Baseline (146) | Cumulative Steps |
|-------|----------------------|-------------------|-----------------|
| 1 | 160 | +9.6% | ~2.07M |
| 2 | 277 | +89.7% | ~3.12M |
| 3 | 430 | +194.5% | ~4.17M |
| 4 | 196 (dip) | +34% | ~5.22M |
| 5 | **500** | **+242%** | ~6.27M |
| 6 | 455 | +211% | ~7.31M |

**Solo mode effectively solved** — snake hits 500-turn max consistently. Training diagnostics at convergence: explained_variance=0.86, value_loss=0.0073, entropy=-1.19.

**Key observations:**
- Round 2 peak (335 turns) was best checkpoint to resume from — Round 3 of initial run collapsed (162 turns)
- Policy stability improved with longer rounds (750K vs 250K steps)
- Training converged around ~6M total steps

**Final model:** `checkpoints/solo_v2b/final_model.zip` (127 intermediate checkpoints preserved)
