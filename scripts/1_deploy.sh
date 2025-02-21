#!/usr/bin/bash
set -e

usage() {
  printf 'Usage: %s <n_workers> \n' "$0"
  printf '      n_workers: number of workers to set up \n'
}

if [ $# -ne 1 ]; then
  usage
  exit 1
fi

cd terraform || exit 1
terraform init
terraform apply -auto-approve -var "n_workers=$1"

echo "Deployed scheduler and workers!"
