"""Battlesnake HTTP inference server using a trained RL model.

Implements the standard Battlesnake API:
  GET  /          → info
  POST /start     → start game
  POST /move      → choose move
  POST /end       → end game

Compatible with `go tool battlesnake play` and `make compare`.
"""

import argparse
import json
import sys
from pathlib import Path

import numpy as np
import torch
from flask import Flask, request, jsonify

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from model.network import BattlesnakeActorCritic

app = Flask(__name__)

# Global model state.
_model: BattlesnakeActorCritic | None = None
_device: torch.device = torch.device("cpu")

SPATIAL_SIZE = 21
CENTER = 10
CHANNELS = 11


def load_model(path: str, device: str = "cpu"):
    """Load a trained model from a PyTorch checkpoint."""
    global _model, _device
    _device = torch.device(device)

    checkpoint = torch.load(path, map_location=_device, weights_only=False)

    # Handle both raw state dicts and SB3 policy state dicts.
    if "policy_state_dict" in checkpoint:
        # SB3 policy — we need to extract just the feature extractor and heads.
        # For simplicity, create our standalone model and load matching keys.
        _model = BattlesnakeActorCritic()
        _load_from_sb3_policy(checkpoint["policy_state_dict"])
    elif "state_dict" in checkpoint:
        _model = BattlesnakeActorCritic()
        _model.load_state_dict(checkpoint["state_dict"])
    else:
        # Assume the checkpoint IS the state dict.
        _model = BattlesnakeActorCritic()
        _model.load_state_dict(checkpoint)

    _model.to(_device)
    _model.eval()
    print(f"Model loaded from {path} on {_device}")


def _load_from_sb3_policy(sb3_state: dict):
    """Map SB3 policy state dict keys to our BattlesnakeActorCritic."""
    our_state = _model.state_dict()
    mapped = {}

    # SB3 key patterns:
    # features_extractor.spatial_encoder.{0,3,6}.{weight,bias} → spatial_encoder
    # features_extractor.scalar_encoder.0.{weight,bias} → scalar_encoder
    # features_extractor.merge.0.{weight,bias} → shared
    # mlp_extractor.policy_net.0.{weight,bias} → policy_head.0
    # action_net.{weight,bias} → policy_head.2
    # mlp_extractor.value_net.0.{weight,bias} → value_head.0
    # value_net.{weight,bias} → value_head.2

    key_map = {
        "features_extractor.spatial_encoder": "spatial_encoder",
        "features_extractor.scalar_encoder": "scalar_encoder",
        "features_extractor.merge": "shared",
    }

    for sb3_key, val in sb3_state.items():
        our_key = None

        # Feature extractor mappings.
        for sb3_prefix, our_prefix in key_map.items():
            if sb3_key.startswith(sb3_prefix):
                suffix = sb3_key[len(sb3_prefix):]
                our_key = our_prefix + suffix
                break

        # Policy network.
        if our_key is None and sb3_key.startswith("mlp_extractor.policy_net."):
            suffix = sb3_key[len("mlp_extractor.policy_net."):]
            our_key = "policy_head." + suffix
        if our_key is None and sb3_key.startswith("action_net."):
            suffix = sb3_key[len("action_net."):]
            our_key = "policy_head.2." + suffix

        # Value network.
        if our_key is None and sb3_key.startswith("mlp_extractor.value_net."):
            suffix = sb3_key[len("mlp_extractor.value_net."):]
            our_key = "value_head." + suffix
        if our_key is None and sb3_key.startswith("value_net."):
            suffix = sb3_key[len("value_net."):]
            our_key = "value_head.2." + suffix

        if our_key is not None and our_key in our_state:
            if our_state[our_key].shape == val.shape:
                mapped[our_key] = val

    if mapped:
        _model.load_state_dict(mapped, strict=False)
        print(f"Loaded {len(mapped)}/{len(our_state)} parameters from SB3 policy")
    else:
        print("WARNING: Could not map any SB3 policy parameters!")


def game_state_to_observation(data: dict) -> tuple[np.ndarray, np.ndarray, np.ndarray]:
    """Convert Battlesnake API game state to model observation."""
    board = data["board"]
    you = data["you"]
    turn = data.get("turn", 0)

    my_head = you["body"][0]
    hx, hy = my_head["x"], my_head["y"]
    w, h = board["width"], board["height"]

    spatial = np.zeros((CHANNELS, SPATIAL_SIZE, SPATIAL_SIZE), dtype=np.float32)

    def to_obs(cx, cy):
        ox = cx - hx + CENTER
        oy = cy - hy + CENTER
        if 0 <= ox < SPATIAL_SIZE and 0 <= oy < SPATIAL_SIZE:
            return ox, oy
        return None, None

    # Ch 0: Walls.
    for oy in range(SPATIAL_SIZE):
        for ox in range(SPATIAL_SIZE):
            bx = ox - CENTER + hx
            by = oy - CENTER + hy
            if bx < 0 or bx >= w or by < 0 or by >= h:
                spatial[0, oy, ox] = 1.0

    # Our body (ch 1, 2, 7, 9).
    my_body = you["body"]
    body_len = len(my_body)
    for seg_i, seg in enumerate(my_body):
        ox, oy = to_obs(seg["x"], seg["y"])
        if ox is None:
            continue
        spatial[1, oy, ox] = 1.0
        if seg_i == 0:
            spatial[2, oy, ox] = 1.0
        if body_len > 1:
            spatial[7, oy, ox] = max(
                spatial[7, oy, ox], seg_i / (body_len - 1)
            )
        # Stacking: check if this segment overlaps with the next.
        if seg_i < body_len - 1:
            next_seg = my_body[seg_i + 1]
            if seg["x"] == next_seg["x"] and seg["y"] == next_seg["y"]:
                spatial[9, oy, ox] = 1.0

    # Opponent snakes (ch 3, 4, 8, 10).
    for snake in board["snakes"]:
        if snake["id"] == you["id"]:
            continue
        opp_body = snake["body"]
        opp_len = len(opp_body)
        for seg_i, seg in enumerate(opp_body):
            ox, oy = to_obs(seg["x"], seg["y"])
            if ox is None:
                continue
            spatial[3, oy, ox] = 1.0
            if seg_i == 0:
                spatial[4, oy, ox] = 1.0
            if opp_len > 1:
                spatial[8, oy, ox] = max(
                    spatial[8, oy, ox], seg_i / (opp_len - 1)
                )
            if seg_i < opp_len - 1:
                next_seg = opp_body[seg_i + 1]
                if seg["x"] == next_seg["x"] and seg["y"] == next_seg["y"]:
                    spatial[10, oy, ox] = 1.0

    # Food (ch 5).
    for food in board.get("food", []):
        ox, oy = to_obs(food["x"], food["y"])
        if ox is not None:
            spatial[5, oy, ox] = 1.0

    # Hazards (ch 6).
    for haz in board.get("hazards", []):
        ox, oy = to_obs(haz["x"], haz["y"])
        if ox is not None:
            spatial[6, oy, ox] = 1.0

    # Scalars.
    my_health = you.get("health", 100)
    my_length = you.get("length", len(my_body))
    opp_health, opp_length = 0, 0
    for snake in board["snakes"]:
        if snake["id"] != you["id"]:
            opp_health = snake.get("health", 0)
            opp_length = snake.get("length", len(snake["body"]))
            break

    scalars = np.array(
        [
            my_health / 100.0,
            my_length / 30.0,
            opp_health / 100.0,
            opp_length / 30.0,
            turn / 500.0,
            w / 19.0,
            h / 19.0,
            len(board.get("food", [])) / 10.0,
        ],
        dtype=np.float32,
    )

    # Action mask: block 180-degree reversal.
    mask = np.ones(4, dtype=np.float32)
    if len(my_body) >= 2:
        neck = my_body[1]
        dx = my_head["x"] - neck["x"]
        dy = my_head["y"] - neck["y"]
        # Up=0, Down=1, Left=2, Right=3
        if dy == 1:   # facing up, block down
            mask[1] = 0
        elif dy == -1: # facing down, block up
            mask[0] = 0
        elif dx == -1: # facing left, block right
            mask[3] = 0
        elif dx == 1:  # facing right, block left
            mask[2] = 0

    return spatial, scalars, mask


DIRECTION_NAMES = ["up", "down", "left", "right"]


@app.route("/", methods=["GET"])
def info():
    return jsonify(
        {
            "apiversion": "1",
            "author": "haruko-rl",
            "color": "#4488FF",
            "head": "default",
            "tail": "default",
        }
    )


@app.route("/start", methods=["POST"])
def start():
    return jsonify({})


@app.route("/move", methods=["POST"])
def move():
    data = request.get_json()
    spatial, scalars, mask = game_state_to_observation(data)

    with torch.no_grad():
        spatial_t = torch.from_numpy(spatial).unsqueeze(0).to(_device)
        scalars_t = torch.from_numpy(scalars).unsqueeze(0).to(_device)
        mask_t = torch.from_numpy(mask).unsqueeze(0).to(_device)

        logits, _ = _model.forward_with_mask(spatial_t, scalars_t, mask_t)
        probs = torch.softmax(logits, dim=-1)
        action = torch.argmax(probs, dim=-1).item()

    return jsonify({"move": DIRECTION_NAMES[action]})


@app.route("/end", methods=["POST"])
def end():
    return jsonify({})


def main():
    parser = argparse.ArgumentParser(description="Battlesnake RL inference server")
    parser.add_argument("--model", required=True, help="Path to model checkpoint")
    parser.add_argument("--port", type=int, default=8082)
    parser.add_argument("--device", default="cpu")
    args = parser.parse_args()

    load_model(args.model, args.device)
    print(f"Starting Battlesnake HTTP server on :{args.port}")
    app.run(host="0.0.0.0", port=args.port)


if __name__ == "__main__":
    main()
