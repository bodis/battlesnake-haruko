#!/bin/bash
# MC Rollout phase 2 sweep — confirm top configs at N=100
set -e
N=${1:-100}
PORT_A=8080
PORT_B=8081
PREV="snapshots/haruko-69c43bb"

go build -o haruko .

CONFIGS=(
    "w20-s05              20    0.05 0.1 100"
    "w18-s05              18    0.05 0.1 100"
    "w22-s05              22    0.05 0.1 100"
    "w20-s03              20    0.03 0.1 100"
    "w20-s07              20    0.07 0.1 100"
    "w20-s05-gate005      20    0.05 0.05 100"
    "w20-s05-turns75      20    0.05 0.1  75"
)

echo "===== MC Sweep Phase 2 (N=$N, vs v32) ====="
echo ""
FMT="%-30s | %4s | %6s | %4s"
printf "$FMT\n" "Config" "Wins" "Win%" "AvgT"
printf "$FMT\n" "------------------------------" "----" "------" "----"

for config in "${CONFIGS[@]}"; do
    read -r label weight spread gate turns <<< "$config"

    MC_WEIGHT=$weight MC_SPREAD=$spread MC_GATE=$gate MC_TURNS=$turns \
        PORT=$PORT_A ./haruko >/dev/null 2>&1 &
    PID_A=$!

    PORT=$PORT_B $PREV >/dev/null 2>&1 &
    PID_B=$!

    sleep 0.5

    OUTPUT=$(go run ./cmd/bench -games $N \
        -a-name "current" -a-url "http://localhost:$PORT_A" \
        -b-name "v32" -b-url "http://localhost:$PORT_B" \
        -save "scripts/mc2_${label// /}.jsonl" 2>/dev/null)

    WINS=$(echo "$OUTPUT" | grep "current (A)" | awk '{print $3}')
    WINPCT=$(echo "$OUTPUT" | grep "current (A)" | sed 's/.*(\s*//' | sed 's/%.*//')
    AVGTURNS=$(echo "$OUTPUT" | grep "Turns" | awk '{print $4}')

    DESC="$label w=$weight s=$spread g=$gate"
    printf "$FMT\n" "$DESC" "$WINS" "${WINPCT}%" "$AVGTURNS"

    kill $PID_A $PID_B 2>/dev/null || true
    wait $PID_A $PID_B 2>/dev/null || true
    sleep 0.3
done

echo ""
echo "===== Phase 2 Complete ====="
