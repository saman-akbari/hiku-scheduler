#!/usr/bin/bash
set -e
#set -x

usage() {
  printf 'Usage: %s [-n n_iterations] [-m mode] <n_workers> <n_copies> \n' "$0"
  printf '      n_iterations: number of iterations to run the experiment (default: 1) \n'
  printf '      mode: local or cloud (default: local) \n'
  printf '      n_workers: number of workers to set up \n'
  printf '      n_copies: number of copies of each benchmark to deploy \n'
  exit 1
}

cleanup() {
  ./scripts/3_clean.sh -m "$mode"
  exit 1
}

trap cleanup SIGINT ERR

#-------------------#
# Timer start
start_time=$(date +%s)
minutes=0

# Parse parameters
n_iterations=1
mode="local"
while getopts ":n:m:" opt; do
  case ${opt} in
    n )
      n_iterations=$OPTARG
      ;;
    m )
      mode=$OPTARG
      ;;
    \? )
      usage
      ;;
  esac
done
shift $((OPTIND -1))

if [ $# -lt 2 ] || [ "$mode" != "local" ] && [ "$mode" != "cloud" ]; then
  usage
fi

n_workers=$1
n_copies=$2

# Clean up previous runs
./scripts/3_clean.sh -m "$mode"

# Run experiments
for n in $(seq 1 "$n_iterations"); do
  timestamp=$(date +"%Y-%m-%d_%H-%M-%S")
  seed=$(date +%s)

  for balancer in pull-based hashing-bounded least-connections random; do
    if [ "$mode" == "cloud" ]; then
      ./scripts/1_deploy.sh "$n_workers"

      echo "Waiting for instances to start..."
      sleep 20
    fi

    # Get worker and scheduler URL
    if [ "$mode" == "cloud" ]; then
      cd terraform || exit 1
      scheduler_url=$(terraform output -raw scheduler_url)
      worker_url_output=$(terraform output worker_url | tr -d '[]\n'\''" ')
      cd ..

      worker_url_list=()
      IFS=',' read -r -a worker_url_list <<< "$worker_url_output"
    else
      scheduler_url="localhost"
    fi

    # Set up workers
    ./scripts/2_setup.sh -m "$mode" "$n_workers" "$n_copies"
    mkdir -p "results/${balancer}/${timestamp}/"

    # Set up scheduler
    if [ "$mode" == "cloud" ]; then
      ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/id_rsa ec2-user@"$scheduler_url" \
        "mkdir -p results/${balancer}/${timestamp};
        ./hiku start --config=\"worker_config/${balancer}.json\" > results/${balancer}/${timestamp}/balancer.log 2>&1 &"
    else
      src/bin/hiku start --config="worker_config/${balancer}.json" > "results/${balancer}/${timestamp}/balancer.log" 2>&1 &
    fi

    # Run load test
    k6 run -e N_COPIES="${n_copies}" -e SEED="${seed}" -e SCHEDULER_DNS="${scheduler_url}" \
      --out json="results/${balancer}/${timestamp}/load_test.json" \
      evaluation/load_test.js

    # Forward logs
    if [ "$mode" == "cloud" ]; then
      scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/id_rsa \
          ec2-user@"$scheduler_url:results/${balancer}/${timestamp}/balancer.log" \
          "results/${balancer}/${timestamp}/"

      for worker_url in "${worker_url_list[@]}"; do
        worker_name=$(ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/id_rsa ec2-user@"$worker_url" "ls -d worker-*")
        scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/id_rsa \
          ec2-user@"$worker_url:${worker_name}/worker.out" \
          "results/${balancer}/${timestamp}/${worker_name}.log"
      done
    else
      for worker_dir in $(ls -d worker-*); do
        worker_name=$(basename "$worker_dir")
        cp "${worker_name}/worker.out" "results/${balancer}/${timestamp}/${worker_name}.log"
      done
    fi

    ./scripts/3_clean.sh -m "$mode"
  done
done

# Timer end
end_time=$(date +%s)
elapsed_time=$((end_time - start_time))
minutes=$((elapsed_time / 60))

echo "Experiment completed in $minutes minutes"
