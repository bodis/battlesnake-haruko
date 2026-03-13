"""Gymnasium/SB3 environment wrapping the Go gRPC env server."""

import json
from typing import Any

import grpc
import gymnasium as gym
import numpy as np
from stable_baselines3.common.vec_env import VecEnv

from proto import battlesnake_env_pb2 as pb
from proto import battlesnake_env_pb2_grpc as pb_grpc


class BattlesnakeVecEnv(VecEnv):
    """SB3-compatible vectorized env that batches all N envs through single gRPC calls.

    Uses SB3's VecEnv interface (not Gymnasium's VectorEnv) for direct
    compatibility with PPO and other SB3 algorithms.
    """

    def __init__(
        self,
        server_addr: str = "localhost:50051",
        num_envs: int = 128,
        board_width: int = 11,
        board_height: int = 11,
        opponent_type: str = "none",
        max_turns: int = 500,
        brs_budget_ms: int = 50,
    ):
        self._spatial_shape = (11, 21, 21)
        self._num_scalars = 8

        channel = grpc.insecure_channel(
            server_addr,
            options=[
                ("grpc.max_send_message_length", 64 * 1024 * 1024),
                ("grpc.max_receive_message_length", 64 * 1024 * 1024),
            ],
        )
        self._stub = pb_grpc.BattlesnakeEnvStub(channel)

        # Configure the Go server.
        cfg_resp = self._stub.Configure(
            pb.ConfigRequest(
                num_envs=num_envs,
                board_width=board_width,
                board_height=board_height,
                opponent_type=opponent_type,
                max_turns=max_turns,
                brs_budget_ms=brs_budget_ms,
            )
        )
        assert cfg_resp.num_envs == num_envs

        observation_space = gym.spaces.Dict(
            {
                "spatial": gym.spaces.Box(
                    low=0.0, high=1.0, shape=self._spatial_shape, dtype=np.float32
                ),
                "scalars": gym.spaces.Box(
                    low=0.0, high=100.0, shape=(self._num_scalars,), dtype=np.float32
                ),
                "action_mask": gym.spaces.MultiBinary(4),
            }
        )
        action_space = gym.spaces.Discrete(4)

        super().__init__(num_envs, observation_space, action_space)

        self._env_ids_i32 = list(range(num_envs))
        self._actions_buf = np.zeros(num_envs, dtype=np.int32)

    def reset(self) -> dict:
        resp = self._stub.Reset(pb.ResetRequest(env_ids=self._env_ids_i32))
        obs = self._parse_batch_obs(resp)
        return obs

    def step_async(self, actions: np.ndarray) -> None:
        self._actions_buf[:] = actions

    def step_wait(self) -> tuple:
        actions_list = [int(a) for a in self._actions_buf]
        resp = self._stub.Step(
            pb.StepRequest(env_ids=self._env_ids_i32, actions=actions_list)
        )

        obs = self._parse_batch_obs(resp.observations)
        rewards = np.array(resp.rewards, dtype=np.float32)
        dones = np.array(resp.dones, dtype=bool)

        # SB3 VecEnv info format: list of dicts, one per env.
        infos = []
        for i in range(self.num_envs):
            info = json.loads(resp.infos[i]) if i < len(resp.infos) else {}
            if dones[i]:
                # SB3 expects terminal_observation for done envs.
                # The Go server auto-resets, so current obs is post-reset.
                # Store a zero terminal obs — acceptable for PPO with GAE.
                info["terminal_observation"] = {
                    "spatial": np.zeros(self._spatial_shape, dtype=np.float32),
                    "scalars": np.zeros(self._num_scalars, dtype=np.float32),
                    "action_mask": np.ones(4, dtype=np.int8),
                }
            infos.append(info)

        return obs, rewards, dones, infos

    def close(self) -> None:
        pass

    def env_is_wrapped(self, wrapper_class, indices=None):
        return [False] * self.num_envs

    def env_method(self, method_name, *method_args, indices=None, **method_kwargs):
        raise NotImplementedError

    def get_attr(self, attr_name, indices=None):
        if attr_name == "render_mode":
            return [None] * self.num_envs
        raise AttributeError(f"No attribute {attr_name}")

    def set_attr(self, attr_name, value, indices=None):
        pass

    def seed(self, seed=None):
        pass

    def _parse_batch_obs(self, obs_msg) -> dict:
        n = self.num_envs
        ch, h, w = self._spatial_shape

        spatial = np.array(obs_msg.spatial, dtype=np.float32).reshape(n, ch, h, w)
        scalars = np.array(obs_msg.scalars, dtype=np.float32).reshape(
            n, self._num_scalars
        )
        action_mask = np.array(obs_msg.action_mask, dtype=np.int8).reshape(n, 4)

        return {
            "spatial": spatial,
            "scalars": scalars,
            "action_mask": action_mask,
        }
