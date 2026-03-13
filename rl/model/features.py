"""SB3 custom feature extractor for Battlesnake Dict observation space."""

import gymnasium as gym
import torch
import torch.nn as nn
from stable_baselines3.common.torch_layers import BaseFeaturesExtractor


class BattlesnakeFeatureExtractor(BaseFeaturesExtractor):
    """Extracts features from the Dict obs space for SB3's PPO.

    Architecture (MPS-compatible — no AdaptiveAvgPool2d):
        (11, 21, 21) → Conv(32, 5, pad=2) → ReLU                    → (32, 21, 21)
                     → Conv(64, 3, stride=2, pad=1) → ReLU           → (64, 11, 11)
                     → Conv(64, 3, stride=2, pad=1) → ReLU           → (64, 6, 6)
                     → Conv(64, 3, stride=2, pad=1) → ReLU           → (64, 3, 3)
                     → Flatten                                        → 576

        Scalars: (8,) → Linear(64) → ReLU                            → 64
        Merge: 640 → Linear(features_dim) → ReLU                     → 256
    """

    def __init__(self, observation_space: gym.spaces.Dict, features_dim: int = 256):
        super().__init__(observation_space, features_dim)

        self.spatial_encoder = nn.Sequential(
            nn.Conv2d(11, 32, kernel_size=5, padding=2),
            nn.ReLU(),
            nn.Conv2d(32, 64, kernel_size=3, stride=2, padding=1),
            nn.ReLU(),
            nn.Conv2d(64, 64, kernel_size=3, stride=2, padding=1),
            nn.ReLU(),
            nn.Conv2d(64, 64, kernel_size=3, stride=2, padding=1),
            nn.ReLU(),
            nn.Flatten(),
        )  # 64 * 3 * 3 = 576

        self.scalar_encoder = nn.Sequential(
            nn.Linear(8, 64),
            nn.ReLU(),
        )

        self.merge = nn.Sequential(
            nn.Linear(576 + 64, features_dim),
            nn.ReLU(),
        )

    def forward(self, observations: dict) -> torch.Tensor:
        spatial = observations["spatial"]
        scalars = observations["scalars"]
        spatial_features = self.spatial_encoder(spatial)
        scalar_features = self.scalar_encoder(scalars)
        merged = torch.cat([spatial_features, scalar_features], dim=1)
        return self.merge(merged)
