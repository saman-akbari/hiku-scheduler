#!/usr/bin/bash
set -e
#set -x

RESULTS_FILE="results/cold_and_warm_start_latency.json"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" >&2
}

BENCHMARKS=(
    "chameleon"
    "dd"
    "float_operation"
    "gzip_compression"
    "json_dumps_loads"
    "linpack"
    "matmul"
    "pyaes"
)
RUNS=20

deploy_worker() {
    log "Deploying worker..."
    ./scripts/1_deploy.sh 1
    log "Worker deployed successfully."
}

get_worker_url() {
    cd terraform || exit 1
    worker_url_output=$(terraform output worker_url | tr -d '[]\n'\''" ')
    cd ..
    worker_url_output=${worker_url_output::-1}
    echo "$worker_url_output"
}

setup_worker() {
    num_workers=$1
    copies=$2
    log "Setting up $num_workers worker with $copies copies..."
    ./scripts/2_setup.sh -m cloud "$num_workers" "$copies"
    log "Worker setup complete."
}

call_benchmark() {
    worker_url=$1
    benchmark=$2
    copy=$3
    local payload

    case $benchmark in
        "chameleon")
            payload=$(jq -n --arg rows "250" --arg cols "250" '{num_of_rows: $rows | tonumber, num_of_cols: $cols | tonumber, metadata: ""}')
            ;;
        "dd")
            payload=$(jq -n --arg bs "1024" --arg count "100000" '{bs: $bs, count: $count}')
            ;;
        "float_operation")
            payload=$(jq -n --arg n "100000" '{n: $n | tonumber, metadata: ""}')
            ;;
        "gzip_compression")
            payload=$(jq -n --arg file_size "5" '{file_size: $file_size | tonumber}')
            ;;
        "json_dumps_loads")
            payload=$(jq -n --arg link "https://api.nobelprize.org/2.1/nobelPrizes" '{link: $link}')
            ;;
        "linpack")
            payload=$(jq -n --arg n "100" '{n: $n | tonumber, metadata: ""}')
            ;;
        "matmul")
            payload=$(jq -n --arg n "100" '{n: $n | tonumber, metadata: ""}')
            ;;
        "pyaes")
            payload=$(jq -n --arg length_of_message "100" --arg num_of_iterations "100" '{length_of_message: $length_of_message | tonumber, num_of_iterations: $num_of_iterations | tonumber, metadata: ""}')
            ;;
        *)
            log "Error: Invalid benchmark $benchmark"
            exit 1
            ;;
    esac

    url="$worker_url:5000/run/${benchmark}-${copy}"
    response=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" -d "$payload" "$url")

    if [ "$response" -ne 200 ]; then
        log "Error: Received HTTP response $response for $benchmark"
    fi
}

cold_start() {
    worker_url=$1
    declare -A cold_start_latencies

    for benchmark in "${BENCHMARKS[@]}"; do
        latencies=()
        for ((i = 0; i < RUNS; i++)); do
            start_time=$(date +%s%3N)
            call_benchmark "$worker_url" "$benchmark" "$i"
            end_time=$(date +%s%3N)
            time_taken=$((end_time - start_time))
            latencies+=("$time_taken")

            log "Benchmark: $benchmark, Run: $((i + 1)), Time: $time_taken milliseconds"
        done

        cold_start_latencies["$benchmark"]="[$(IFS=,; echo "${latencies[*]}")]"
    done

    echo "{"
    for benchmark in "${BENCHMARKS[@]}"; do
        if [ "$benchmark" == "${BENCHMARKS[-1]}" ]; then
            echo "  \"$benchmark\": ${cold_start_latencies[$benchmark]}"
        else
            echo "  \"$benchmark\": ${cold_start_latencies[$benchmark]},"
        fi
    done
    echo "}"
}

warm_start() {
    worker_url=$1
    declare -A warm_start_latencies

    for benchmark in "${BENCHMARKS[@]}"; do
        latencies=()

        call_benchmark "$worker_url" "$benchmark" 0

        for ((i = 0; i < RUNS; i++)); do
            start_time=$(date +%s%3N)
            call_benchmark "$worker_url" "$benchmark" 0
            end_time=$(date +%s%3N)
            time_taken=$((end_time - start_time))
            latencies+=("$time_taken")

            log "Benchmark: $benchmark, Run: $((i + 1)), Time: $time_taken milliseconds"
        done

        warm_start_latencies["$benchmark"]="[$(IFS=,; echo "${latencies[*]}")]"
    done

    echo "{"
    for benchmark in "${BENCHMARKS[@]}"; do
        if [ "$benchmark" == "${BENCHMARKS[-1]}" ]; then
            echo "  \"$benchmark\": ${warm_start_latencies[$benchmark]}"
        else
            echo "  \"$benchmark\": ${warm_start_latencies[$benchmark]},"
        fi
    done
    echo "}"
}

cleanup() {
    ./scripts/3_clean.sh -m cloud
    exit 0
}

trap cleanup SIGINT ERR
deploy_worker
setup_worker 1 "$RUNS"
worker_url=$(get_worker_url)

for benchmark in "${BENCHMARKS[@]}"; do
    call_benchmark "$worker_url" "$benchmark" 0
done
sleep 10

cold_start_latencies=$(cold_start "$worker_url")
warm_start_latencies=$(warm_start "$worker_url")

echo "{" > "$RESULTS_FILE"
echo "  \"cold_start\": $cold_start_latencies," >> "$RESULTS_FILE"
echo "  \"warm_start\": $warm_start_latencies" >> "$RESULTS_FILE"
echo "}" >> "$RESULTS_FILE"

./scripts/3_clean.sh -m cloud

log "Results saved to $RESULTS_FILE."
