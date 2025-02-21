#!/usr/bin/bash

usage() {
  printf 'Usage: %s [-m mode] \n' "$0"
  printf '      mode: local or cloud (default: local) \n'
  exit 1
}

#-------------------#
# Parse parameter
mode="local"
while getopts ":m:" opt; do
  case ${opt} in
    m )
      mode=$OPTARG
      ;;
    \? )
      usage
      ;;
  esac
done

if [ "$mode" != "local" ] && [ "$mode" != "cloud" ]; then
  usage
fi

exec 2>/dev/null

if [ "$mode" == "cloud" ]; then
  cd terraform || exit 1
  terraform destroy -auto-approve
  cd ..

  rm ol-min.tar
else
  # Find existing worker directories
  worker_dirs=$(ls -d worker-*)

  # Shutdown and remove workers
  count=0
  for worker_dir in $worker_dirs; do
    worker_name=$(basename "$worker_dir")

    kill $(cat "$worker_name"/worker/worker.pid)
    rm "$worker_name"/worker/worker.pid
    ./openlambda worker down -p "$worker_name" >/dev/null
    ./openlambda worker force-cleanup -p "$worker_name" >/dev/null

    kill -9 $(lsof -t -i:$((5000 + count)))

    count=$((count + 1))
  done

  # Shut down scheduler
  kill $(lsof -t -i:9020) 2>/dev/null

  # Remove worker directories
  rm -rf worker-*
fi

rm src/bin/hiku
rm -rf worker_config

echo "Cleaned up!"
