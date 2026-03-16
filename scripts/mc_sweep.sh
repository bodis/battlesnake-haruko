#!/bin/bash
# MC Rollout parameter sweep — test configurations against v32 snapshot.
# Usage: bash scripts/mc_sweep.sh [N=50]

set -e
N=${1:-50}
PORT_A=8080
PORT_B=8081
PREV="snapshots/haruko-69c43bb"  # v32 snapshot
RESULTS_FILE="scripts/mc_sweep_results.txt"

if [ ! -f "$PREV" ]; then
    echo "ERROR: v32 snapshot not found at $PREV"
    exit 1
fi

go build -o haruko .

# Each config: "label MC_WEIGHT MC_SPREAD MC_GATE MC_TURNS"
CONFIGS=(
    "mc-off                0    0.10 0.1 100"
    "w10-s10              10    0.10 0.1 100"
    "w15-s10              15    0.10 0.1 100"
    "w20-s10              20    0.10 0.1 100"
    "w25-s10              25    0.10 0.1 100"
    "w30-s10              30    0.10 0.1 100"
    "w15-s05              15    0.05 0.1 100"
    "w15-s15              15    0.15 0.1 100"
    "w15-s20              15    0.20 0.1 100"
    "w20-s05              20    0.05 0.1 100"
    "w20-s15              20    0.15 0.1 100"
    "w15-gate02           15    0.10 0.2 100"
    "w15-gate03           15    0.10 0.3 100"
    "w15-turns50          15    0.10 0.1  50"
    "w15-turns200         15    0.10 0.1 200"
)

HEADER="===== MC Rollout Parameter Sweep (N=$N per config, vs v32) ====="
echo "$HEADER"
echo "$HEADER" > "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

FMT="%-30s | %4s | %6s | %4s"
printf "$FMT\n" "Config" "Wins" "Win%" "AvgT" | tee -a "$RESULTS_FILE"
printf "$FMT\n" "------------------------------" "----" "------" "----" | tee -a "$RESULTS_FILE"

for config in "${CONFIGS[@]}"; do
    read -r label weight spread gate turns <<< "$config"

    # Start current (port A) with MC params
    MC_WEIGHT=$weight MC_SPREAD=$spread MC_GATE=$gate MC_TURNS=$turns \
        PORT=$PORT_A ./haruko >/dev/null 2>&1 &
    PID_A=$!

    # Start v32 (port B)
    PORT=$PORT_B $PREV >/dev/null 2>&1 &
    PID_B=$!

    sleep 0.5

    # Run bench, save per-game data
    OUTPUT=$(go run ./cmd/bench -games $N \
        -a-name "current" -a-url "http://localhost:$PORT_A" \
        -b-name "v32" -b-url "http://localhost:$PORT_B" \
        -save "scripts/mc_${label// /}.jsonl" 2>/dev/null)

    # Parse results
    WINS=$(echo "$OUTPUT" | grep "current (A)" | awk '{print $3}')
    WINPCT=$(echo "$OUTPUT" | grep "current (A)" | sed 's/.*(\s*//' | sed 's/%.*//')
    AVGTURNS=$(echo "$OUTPUT" | grep "Turns" | awk '{print $4}')

    DESC="$label w=$weight s=$spread g=$gate"
    printf "$FMT\n" "$DESC" "$WINS" "${WINPCT}%" "$AVGTURNS" | tee -a "$RESULTS_FILE"

    kill $PID_A $PID_B 2>/dev/null || true
    wait $PID_A $PID_B 2>/dev/null || true
    sleep 0.3
done

echo "" | tee -a "$RESULTS_FILE"
echo "===== Sweep Complete =====" | tee -a "$RESULTS_FILE"
echo "Results saved to $RESULTS_FILE"
