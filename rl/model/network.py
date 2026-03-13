"""CNN Actor-Critic network for Battlesnake."""

import torch
import torch.nn as nn


class BattlesnakeActorCritic(nn.Module):
    """Combined actor-critic network with CNN spatial encoder.

    MPS-compatible architecture (no AdaptiveAvgPool2d):
        Spatial: (11, 21, 21) → strided convs → 576
        Scalars: (8,) → Linear(64) → 64
        Merge: 640 → Linear(256) → ReLU
        Policy: → Linear(128) → ReLU → Linear(4)
        Value:  → Linear(128) → ReLU → Linear(1)
    """

    def __init__(self, spatial_channels: int = 11, num_scalars: int = 8):
        super().__init__()

        self.spatial_encoder = nn.Sequential(
            nn.Conv2d(spatial_channels, 32, kernel_size=5, padding=2),
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
            nn.Linear(num_scalars, 64),
            nn.ReLU(),
        )

        self.shared = nn.Sequential(
            nn.Linear(576 + 64, 256),
            nn.ReLU(),
        )

        self.policy_head = nn.Sequential(
            nn.Linear(256, 128),
            nn.ReLU(),
            nn.Linear(128, 4),
        )

        self.value_head = nn.Sequential(
            nn.Linear(256, 128),
            nn.ReLU(),
            nn.Linear(128, 1),
        )

    def forward(self, spatial: torch.Tensor, scalars: torch.Tensor):
        spatial_features = self.spatial_encoder(spatial)
        scalar_features = self.scalar_encoder(scalars)
        merged = torch.cat([spatial_features, scalar_features], dim=1)
        shared = self.shared(merged)
        logits = self.policy_head(shared)
        value = self.value_head(shared)
        return logits, value

    def forward_with_mask(
        self,
        spatial: torch.Tensor,
        scalars: torch.Tensor,
        action_mask: torch.Tensor,
    ):
        logits, value = self.forward(spatial, scalars)
        invalid = ~action_mask.bool()
        logits = logits.masked_fill(invalid, float("-inf"))
        return logits, value
