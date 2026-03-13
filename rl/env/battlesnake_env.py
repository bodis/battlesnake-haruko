"""Gymnasium/SB3 environment wrapping the Go gRPC env server."""

import json
import mmap
import os
import struct
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

    When the Go server provides a shared memory path, observations and rewards
    are read directly from mmap (zero-copy) instead of gRPC protobuf payloads.
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
        self._turn_counts = np.zeros(num_envs, dtype=np.int32)
        self._step_id = 0

        # Set up shared memory if available.
        self._shm_fd = None
        self._shm_mm = None
        self._shm_spatial = None
        self._shm_scalars = None
        self._shm_masks = None
        self._shm_rewards = None
        self._shm_dones = None
        self._shm_actions = None
        self._use_shm = False

        if cfg_resp.shm_path:
            self._init_shm(cfg_resp.shm_path, num_envs)

    def _init_shm(self, path: str, n: int) -> None:
        """Open the shared memory file and create numpy views."""
        self._shm_fd = os.open(path, os.O_RDWR)
        file_size = os.fstat(self._shm_fd).st_size
        self._shm_mm = mmap.mmap(self._shm_fd, file_size)

        # Read header offsets (stored by Go server).
        header = self._shm_mm[:64]
        magic = header[0:4]
        assert magic == b"HSHM", f"Bad shm magic: {magic}"

        spatial_off = struct.unpack_from("<I", header, 24)[0]
        scalars_off = struct.unpack_from("<I", header, 28)[0]
        masks_off = struct.unpack_from("<I", header, 32)[0]
        rewards_off = struct.unpack_from("<I", header, 36)[0]
        dones_off = struct.unpack_from("<I", header, 40)[0]
        actions_off = struct.unpack_from("<I", header, 44)[0]

        ch, h, w = self._spatial_shape
        spatial_count = n * ch * h * w

        self._shm_spatial = np.frombuffer(
            self._shm_mm, dtype=np.float32, offset=spatial_off, count=spatial_count
        ).reshape(n, ch, h, w)
        self._shm_scalars = np.frombuffer(
            self._shm_mm, dtype=np.float32, offset=scalars_off, count=n * self._num_scalars
        ).reshape(n, self._num_scalars)
        self._shm_masks = np.frombuffer(
            self._shm_mm, dtype=np.uint8, offset=masks_off, count=n * 4
        ).reshape(n, 4)
        self._shm_rewards = np.frombuffer(
            self._shm_mm, dtype=np.float32, offset=rewards_off, count=n
        )
        self._shm_dones = np.frombuffer(
            self._shm_mm, dtype=np.uint8, offset=dones_off, count=n
        )
        self._shm_actions = np.frombuffer(
            self._shm_mm, dtype=np.int32, offset=actions_off, count=n
        )

        self._use_shm = True

    def _read_shm_obs(self) -> dict:
        """Read observations from shared memory views (copy to avoid mmap aliasing)."""
        return {
            "spatial": self._shm_spatial.copy(),
            "scalars": self._shm_scalars.copy(),
            "action_mask": self._shm_masks.astype(np.int8).copy(),
        }

    def reset(self) -> dict:
        resp = self._stub.Reset(pb.ResetRequest(env_ids=self._env_ids_i32))
        self._turn_counts[:] = 0
        if self._use_shm:
            return self._read_shm_obs()
        return self._parse_batch_obs(resp)

    def step_async(self, actions: np.ndarray) -> None:
        self._actions_buf[:] = actions

    def step_wait(self) -> tuple:
        if self._use_shm:
            return self._step_wait_shm()
        return self._step_wait_grpc()

    def _step_wait_shm(self) -> tuple:
        """Step using shared memory for data transfer."""
        # Write actions into mmap.
        self._shm_actions[:] = self._actions_buf

        # Signal the Go server to step.
        self._step_id += 1
        self._stub.StepSignal(pb.StepSignalRequest(step_id=self._step_id))

        # Read results from mmap.
        obs = self._read_shm_obs()
        rewards = self._shm_rewards.copy()
        dones = self._shm_dones.astype(bool)

        # Update turn counts and build infos.
        self._turn_counts += 1
        infos = []
        for i in range(self.num_envs):
            info = {"turn": int(self._turn_counts[i])}
            if dones[i]:
                info["terminal_observation"] = {
                    "spatial": np.zeros(self._spatial_shape, dtype=np.float32),
                    "scalars": np.zeros(self._num_scalars, dtype=np.float32),
                    "action_mask": np.ones(4, dtype=np.int8),
                }
                self._turn_counts[i] = 0
            infos.append(info)

        return obs, rewards, dones, infos

    def _step_wait_grpc(self) -> tuple:
        """Step using gRPC protobuf payloads (fallback)."""
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
        # Release numpy views before closing mmap.
        self._shm_spatial = None
        self._shm_scalars = None
        self._shm_masks = None
        self._shm_rewards = None
        self._shm_dones = None
        self._shm_actions = None
        if self._shm_mm is not None:
            self._shm_mm.close()
            self._shm_mm = None
        if self._shm_fd is not None:
            os.close(self._shm_fd)
            self._shm_fd = None

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
