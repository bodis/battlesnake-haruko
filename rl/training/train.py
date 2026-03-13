"""Main training entry point for Battlesnake RL."""

import argparse
import os
import sys
import time
from pathlib import Path

import numpy as np
import torch
from stable_baselines3 import PPO
from stable_baselines3.common.callbacks import (
    BaseCallback,
    CheckpointCallback,
    CallbackList,
)

# Add rl/ to path for imports.
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from env.battlesnake_env import BattlesnakeVecEnv
from model.features import BattlesnakeFeatureExtractor


class ActionMaskWrapper:
    """Wraps the VecEnv to apply action masking in SB3's predict step.

    SB3 doesn't natively support action masking with PPO (only MaskablePPO
    from sb3-contrib). Instead, we store masks and apply them via a callback.
    For the initial implementation, we let the agent learn from invalid
    action penalties. We can add sb3-contrib MaskablePPO later.
    """
    pass


class EpisodeLogCallback(BaseCallback):
    """Logs episode stats to TensorBoard."""

    def __init__(self, verbose=0):
        super().__init__(verbose)
        self._episode_rewards = []
        self._episode_lengths = []

    def _on_step(self) -> bool:
        # SB3 VecEnv: infos is a list of dicts, one per env.
        dones = self.locals.get("dones", [])
        infos = self.locals.get("infos", [])

        if isinstance(infos, list):
            for i, done in enumerate(dones):
                if done and i < len(infos) and isinstance(infos[i], dict):
                    turn = infos[i].get("turn", 0)
                    self.logger.record("episode/turns", turn)

        return True


class BestModelCallback(BaseCallback):
    """Saves the best model based on mean episode reward."""

    def __init__(self, save_path: str, check_freq: int = 10000, verbose=0):
        super().__init__(verbose)
        self.save_path = save_path
        self.check_freq = check_freq
        self.best_mean_reward = -float("inf")
        self.n_calls_since_check = 0

    def _on_step(self) -> bool:
        self.n_calls_since_check += 1
        if self.n_calls_since_check >= self.check_freq:
            self.n_calls_since_check = 0
            # Check rollout buffer for mean reward.
            if hasattr(self.model, "ep_info_buffer") and self.model.ep_info_buffer:
                rewards = [ep["r"] for ep in self.model.ep_info_buffer]
                if rewards:
                    mean_reward = np.mean(rewards)
                    if mean_reward > self.best_mean_reward:
                        self.best_mean_reward = mean_reward
                        path = os.path.join(self.save_path, "best_model")
                        self.model.save(path)
                        if self.verbose:
                            print(
                                f"New best model: mean_reward={mean_reward:.2f}, "
                                f"saved to {path}"
                            )
        return True


def make_env(args) -> BattlesnakeVecEnv:
    """Create the vectorized environment connected to the Go server."""
    return BattlesnakeVecEnv(
        server_addr=args.server_addr,
        num_envs=args.num_envs,
        board_width=args.board_width,
        board_height=args.board_height,
        opponent_type=args.opponent,
        max_turns=args.max_turns,
        brs_budget_ms=args.brs_budget_ms,
    )


def make_model(env, args) -> PPO:
    """Create the PPO model with our custom feature extractor."""
    policy_kwargs = {
        "features_extractor_class": BattlesnakeFeatureExtractor,
        "features_extractor_kwargs": {"features_dim": 256},
        "net_arch": {"pi": [128], "vf": [128]},
    }

    model = PPO(
        "MultiInputPolicy",
        env,
        learning_rate=args.lr,
        n_steps=args.n_steps,
        batch_size=args.batch_size,
        n_epochs=args.n_epochs,
        gamma=args.gamma,
        gae_lambda=args.gae_lambda,
        clip_range=args.clip_range,
        ent_coef=args.ent_coef,
        verbose=1,
        tensorboard_log=args.log_dir,
        policy_kwargs=policy_kwargs,
        device=args.device,
    )
    return model


def load_or_create_model(env, args) -> PPO:
    """Load a checkpoint if specified, otherwise create a new model."""
    if args.resume:
        print(f"Resuming from checkpoint: {args.resume}")
        model = PPO.load(
            args.resume,
            env=env,
            device=args.device,
            tensorboard_log=args.log_dir,
        )
        return model
    return make_model(env, args)


def main():
    parser = argparse.ArgumentParser(description="Train Battlesnake RL agent")
    # Environment.
    parser.add_argument("--server-addr", default="localhost:50051")
    parser.add_argument("--num-envs", type=int, default=64)
    parser.add_argument("--board-width", type=int, default=11)
    parser.add_argument("--board-height", type=int, default=11)
    parser.add_argument("--opponent", default="none", choices=["none", "random", "brs"])
    parser.add_argument("--max-turns", type=int, default=500)
    parser.add_argument("--brs-budget-ms", type=int, default=50)
    # Training.
    parser.add_argument("--total-timesteps", type=int, default=1_000_000)
    parser.add_argument("--lr", type=float, default=3e-4)
    parser.add_argument("--n-steps", type=int, default=2048)
    parser.add_argument("--batch-size", type=int, default=256)
    parser.add_argument("--n-epochs", type=int, default=4)
    parser.add_argument("--gamma", type=float, default=0.99)
    parser.add_argument("--gae-lambda", type=float, default=0.95)
    parser.add_argument("--clip-range", type=float, default=0.2)
    parser.add_argument("--ent-coef", type=float, default=0.01)
    # Output.
    parser.add_argument("--log-dir", default="logs/")
    parser.add_argument("--checkpoint-dir", default="checkpoints/")
    parser.add_argument("--checkpoint-freq", type=int, default=50000)
    parser.add_argument("--device", default="auto")
    # Resume.
    parser.add_argument("--resume", default=None, help="Path to checkpoint .zip to resume from")
    # Run name.
    parser.add_argument("--name", default=None, help="Run name for TensorBoard and checkpoints")

    args = parser.parse_args()

    # Set up directories.
    run_name = args.name or f"solo_{int(time.time())}"
    checkpoint_dir = os.path.join(args.checkpoint_dir, run_name)
    os.makedirs(checkpoint_dir, exist_ok=True)
    os.makedirs(args.log_dir, exist_ok=True)

    print(f"=== Battlesnake RL Training ===")
    print(f"Run: {run_name}")
    print(f"Envs: {args.num_envs}, Board: {args.board_width}x{args.board_height}")
    print(f"Opponent: {args.opponent}, Max turns: {args.max_turns}")
    print(f"Total timesteps: {args.total_timesteps}")
    print(f"Checkpoints: {checkpoint_dir}")
    print()

    env = make_env(args)
    model = load_or_create_model(env, args)

    callbacks = CallbackList(
        [
            CheckpointCallback(
                save_freq=max(args.checkpoint_freq // args.num_envs, 1),
                save_path=checkpoint_dir,
                name_prefix="battlesnake",
            ),
            BestModelCallback(
                save_path=checkpoint_dir,
                check_freq=10000,
                verbose=1,
            ),
            EpisodeLogCallback(),
        ]
    )

    try:
        model.learn(
            total_timesteps=args.total_timesteps,
            callback=callbacks,
            tb_log_name=run_name,
            reset_num_timesteps=args.resume is None,
        )
    except KeyboardInterrupt:
        print("\nTraining interrupted. Saving checkpoint...")

    # Save final model.
    final_path = os.path.join(checkpoint_dir, "final_model")
    model.save(final_path)
    print(f"Final model saved: {final_path}.zip")

    # Also save just the PyTorch state dict for inference server.
    torch_path = os.path.join(checkpoint_dir, "final_policy.pt")
    torch.save(
        {
            "policy_state_dict": model.policy.state_dict(),
            "spatial_channels": 11,
            "spatial_size": 21,
            "num_scalars": 8,
            "num_actions": 4,
        },
        torch_path,
    )
    print(f"PyTorch policy saved: {torch_path}")

    env.close()


if __name__ == "__main__":
    main()
