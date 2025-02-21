#!/usr/bin/bash
set -e
#set -x

usage() {
  printf 'Usage: %s [-m mode] <n_workers> <n_copies> \n' "$0"
  printf '      mode: local or cloud (default: local) \n'
  printf '      n_workers: number of workers to set up \n'
  printf '      n_copies: number of copies of each benchmark to deploy \n'
  exit 1
}

setup_scheduler() {
  (
  cd src || exit 1
  ./build.sh
  )
  chmod +x src/bin/hiku

  if [ "$mode" == "cloud" ]; then
    scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/id_rsa src/bin/hiku ec2-user@"$scheduler_url":~
  fi
}

setup_workers() {
  random_str=$1
  worker_url=$2
  scheduler_url=$3
  index=$4
  mode=$5

  if [ "$mode" == "cloud" ]; then
    yum install -y docker
    systemctl start docker
    gpasswd -a $USER docker
    docker load -i ol-min.tar

    # https://grafana.com/docs/k6/latest/testing-guides/running-large-tests/
    sysctl -w net.ipv4.tcp_tw_reuse=1
    sysctl -w net.ipv4.tcp_timestamps=1
    ulimit -n 250000
  fi

  chmod +x openlambda
  ./openlambda worker init -p "worker-${random_str}" -i ol-min

  # Update worker config
  if ! command -v jq &> /dev/null; then
    if [ "$mode" == "cloud" ]; then
      sudo yum install -y jq
    else
      sudo apt-get install -y jq
    fi
  fi
  jq ".worker_url = \"$worker_url\"" "worker-${random_str}/config.json" > "worker-${random_str}/tmp_config.json" && mv "worker-${random_str}/tmp_config.json" "worker-${random_str}/config.json"
  jq ".worker_port = \"$((5000 + index))\"" "worker-${random_str}/config.json" > "worker-${random_str}/tmp_config.json" && mv "worker-${random_str}/tmp_config.json" "worker-${random_str}/config.json"
  jq ".scheduler_url = \"$scheduler_url\"" "worker-${random_str}/config.json" > "worker-${random_str}/tmp_config.json" && mv "worker-${random_str}/tmp_config.json" "worker-${random_str}/config.json"
  jq ".scheduler_port = \"9020\"" "worker-${random_str}/config.json" > "worker-${random_str}/tmp_config.json" && mv "worker-${random_str}/tmp_config.json" "worker-${random_str}/config.json"
  jq ".trace.memory = true" "worker-${random_str}"/config.json > "worker-${random_str}/tmp_config.json" && mv "worker-${random_str}/tmp_config.json" "worker-${random_str}"/config.json
  jq ".trace.latency = true" "worker-${random_str}"/config.json > "worker-${random_str}/tmp_config.json" && mv "worker-${random_str}/tmp_config.json" "worker-${random_str}"/config.json
  jq ".limits.max_runtime_default = 60" "worker-${random_str}"/config.json > "worker-${random_str}/tmp_config.json" && mv "worker-${random_str}/tmp_config.json" "worker-${random_str}"/config.json
  jq ".limits.mem_mb = 512" "worker-${random_str}"/config.json > "worker-${random_str}/tmp_config.json" && mv "worker-${random_str}/tmp_config.json" "worker-${random_str}"/config.json
  jq ".features.import_cache = \"\"" "worker-${random_str}"/config.json > "worker-${random_str}/tmp_config.json" && mv "worker-${random_str}/tmp_config.json" "worker-${random_str}"/config.json
  jq ".import_cache_tree = \"\"" "worker-${random_str}"/config.json > "worker-${random_str}/tmp_config.json" && mv "worker-${random_str}/tmp_config.json" "worker-${random_str}"/config.json

  # Start worker
  ./openlambda worker up -d -p "worker-${random_str}" -i ol-min

  # Set permissions on worker
  chmod -R a+rwx "worker-${random_str}" 2>/dev/null
}

enable_cgroup_nesting() {
  # cgroup v2: enable nesting
  # https://github.com/moby/moby/issues/43093
  if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
    # move the processes from the root group to the /init group,
    # otherwise writing subtree_control fails with EBUSY.
    # An error during moving non-existent process (i.e., "cat") is ignored.
    mkdir -p /sys/fs/cgroup/init
    xargs -rn1 < /sys/fs/cgroup/cgroup.procs > /sys/fs/cgroup/init/cgroup.procs || :
    # enable controllers
    sed -e "s/ / +/g" -e "s/^/+/" < /sys/fs/cgroup/cgroup.controllers > /sys/fs/cgroup/cgroup.subtree_control
  fi
}

#-------------------#
# Parse optional parameter
mode="local"
while getopts "m:" opt; do
  case $opt in
    m )
      mode=$OPTARG
      ;;
    \? )
      usage
      ;;
  esac
done
shift $((OPTIND -1))

if [ "$mode" != "local" ] && [ "$mode" != "cloud" ]; then
  usage
fi

# Parse required parameters
if [ $# -lt 2 ]; then
  usage
fi

n_workers=$1
n_copies=$2

BENCHMARK_DIR="evaluation/benchmarks"

# Create worker docker image
if [ -z "$(docker images -q ol-min 2> /dev/null)" ]; then
  systemctl start docker
  git clone https://github.com/open-lambda/open-lambda.git
  cd open-lambda || exit 1
  git checkout 0a834cee321fda36767775653394e0d6b5f00a2c
  make imgs/ol-min
  cd ..
  rm -rf open-lambda
fi

# Retrieve worker URL
if [ "$mode" == "cloud" ]; then
  cd terraform || exit 1
  scheduler_url=$(terraform output -raw scheduler_url)
  worker_url=$(terraform output worker_url | tr -d '[]\n'\''" ')
  cd ..

  worker_url_list=()
  IFS=',' read -r -a worker_url_list <<< "$worker_url"
else
  scheduler_url="localhost"
  worker_url_list=()
  for (( i=0; i<n_workers; i++ )); do
    worker_url_list+=("localhost")
  done
fi

setup_scheduler

# Setup workers
chmod +x openlambda
for (( i=0; i<n_workers; i++ )); do
  random_str=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 10 | head -n 1)

  {
    if [ "$mode" == "cloud" ]; then
      docker save ol-min > ol-min.tar
      scp -C -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/id_rsa openlambda ol-min.tar ec2-user@"${worker_url_list[i]}":~
      ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/id_rsa ec2-user@"${worker_url_list[i]}" \
        "sudo bash -c '$(declare -f setup_workers); setup_workers $random_str ${worker_url_list[i]} $scheduler_url $i $mode'"
    else
      enable_cgroup_nesting
      setup_workers "$random_str" "localhost" "localhost" "$i" "$mode"
    fi

    for benchmark in "$BENCHMARK_DIR/"*; do
      if [ "$benchmark" == "$BENCHMARK_DIR/__init__.py" ]; then
          continue
      fi

      if [ "$mode" == "local" ]; then
          for (( j=0; j<n_copies; j++ )); do
              mkdir -p "worker-${random_str}/registry/$(basename "$benchmark")-$j"
              cp -r "$benchmark"/. "worker-${random_str}/registry/$(basename "$benchmark")-$j"
          done
      fi

      if [ "$mode" == "cloud" ]; then
          # Copy the benchmark directory
          scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/id_rsa -r "$benchmark" ec2-user@"${worker_url_list[i]}":~/temp_benchmark

          # Create multiple copies
          ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/id_rsa ec2-user@"${worker_url_list[i]}" "
              for j in \$(seq 0 $((n_copies-1))); do
                  mkdir -p worker-${random_str}/registry/\$(basename $benchmark)-\$j
                  cp -r ~/temp_benchmark/. worker-${random_str}/registry/\$(basename $benchmark)-\$j
              done
              rm -rf ~/temp_benchmark
          "
      fi
    done
  } &  # run in parallel
done
wait

# Create balancer configs
mkdir -p worker_config
for balancer in hashing-bounded least-connections pull-based random; do
  echo "{
  \"host\": \"${scheduler_url}\",
  \"port\": 9020,
  \"balancer\": \"${balancer}\",
  \"workers\": [" >> "worker_config/${balancer}.json"

  # Generate worker list
  for ((i = 0; i < n_workers; i++)); do
    worker_url="http://${worker_url_list[i]}":"$((5000 + i))"

    if [[ $i -ne n_workers-1 ]]; then
      separator=","
    else
      separator=""
    fi

    echo -n "\"$worker_url\"$separator" >> "worker_config/${balancer}.json"
  done
  echo "]}" >> "worker_config/${balancer}.json"
done

if [ "$mode" == "cloud" ]; then
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/id_rsa ec2-user@"$scheduler_url" "mkdir worker_config"
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/id_rsa worker_config/* ec2-user@"$scheduler_url":~/worker_config/
fi

echo "Setup completed!"
