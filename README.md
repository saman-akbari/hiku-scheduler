# Hiku: OpenLambda Scheduler

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Hiku is an extensible, lightweight HTTP request scheduler for serverless functions
in [OpenLambda](https://github.com/open-lambda/open-lambda), written in Go. Given a cluster of workers, the scheduler
routes incoming requests to the most suitable worker, supporting various load balancing strategies for optimal request
distribution.

**Note**: Hiku is not limited to OpenLambda and can be used with any serverless platform that supports HTTP requests.
To use it with other platforms, you will need to modify the API endpoints to match your platform's specifications.

## Features

- **Extensible:** Easily integrates with custom load balancing strategies.
- **Lightweight:** Minimal resource overhead.
- **Ready to Use:** Provides scripts for automated deployment and evaluation on AWS and local environments.


## Paper

S. Akbari and M. Hauswirth, ‘Hiku: Pull-Based Scheduling for Serverless Computing’, in 2025 IEEE 25th International Symposium on Cluster, Cloud and Internet Computing (CCGrid), 2025, pp. 450–461.

[![DOI](https://zenodo.org/badge/doi/10.1109/CCGRID64434.2025.00034.svg)](https://dx.doi.org/10.1109/CCGRID64434.2025.00034)
[![arXiv](https://img.shields.io/badge/arXiv-2502.15534-B31B1B.svg)](https://arxiv.org/abs/2502.15534)

### Citation

```bibtex
@inproceedings{akbari2025hiku,
  title={Hiku: Pull-Based Scheduling for Serverless Computing},
  author={Akbari, Saman and Hauswirth, Manfred},
  booktitle={2025 IEEE 25th International Symposium on Cluster, Cloud and Internet Computing (CCGrid)},
  pages={450--461},
  year={2025},
  organization={IEEE}
}
```

## Prerequisites

### Software Requirements

- **Go** 1.22+

Optional dependencies:

- **Docker** for setup of OpenLambda workers
- **Terraform** for cloud deployment
- **k6** for load testing
- **Python** for evaluation
- **LaTeX** (optional, for generating evaluation plots)

### Installation

To set up the project, run:

```bash
go get github.com/saman-akbari/hiku-scheduler
```

## Usage

### Starting the Scheduler

The scheduler proxies incoming HTTP requests to a cluster of workers. You need to set up and start the workers
separately, but we provide scripts for that (more information below). You can configure and start the scheduler using a
JSON file or via the Go API.

#### JSON Configuration

Create a configuration file, for example `config.json`:

```json
{
  "host": "localhost",
  "port": 9020,
  "balancer": "pull-based",
  "workers": [
    "http://localhost:5000",
    "http://localhost:5001"
  ]
}
```

Start the scheduler:

```bash
hiku start -c config.json
```

#### Go API

If you prefer using Go, you can configure and start the scheduler programmatically:

```go
package main

import (
	"net/url"
	"hiku/balancer"
	"hiku/config"
	"hiku/server"
)

func main() {
	cfg := config.CreateDefaultConfig()
	cfg.Port = 9020

	workerUrls := []url.URL{
		{Scheme: "http", Host: "localhost:5000"},
		{Scheme: "http", Host: "localhost:5001"},
	}
	cfg.Balancer = balancer.NewPullBased(workerUrls)

	server.Start(cfg)
}
```

### Health Check

You can check the health of the scheduler:

```bash
curl <scheduler_url>/status
```

### Managing Workers

You can add or remove workers to the cluster at runtime using the admin API.
The input should be a comma separated list of worker URLs.

- **Add workers:**
  ```bash
  curl <scheduler_url>/admin/workers/add?workers=<worker_url_list>
  ```
  Example: `curl localhost:9020/admin/workers/add?workers=http://localhost:5002,http://localhost:5003`

- **Remove workers:**
  ```bash
  curl <scheduler_url>/admin/workers/remove?workers=<worker_url_list>
  ```
  Example: `curl localhost:9020/admin/workers/remove?workers=http://localhost:5002,http://localhost:5003`

### Sending Requests

To send a request to the scheduler:

  ```bash
  curl <scheduler_url>/run/<function_name> -H "Content-Type: application/json" -d <json_payload>
  ```

Example: `curl localhost:9020/run/gzip_compression-1 -H "Content-Type: application/json" -d '{"file_size": 5}'`

## Evaluation and Benchmarking

We provide code for automated deployment, experimentation, and evaluation. You can run experiments on AWS or locally
and plot the results using Python. The evaluation includes benchmarks from
the [FunctionBench](https://github.com/ddps-lab/serverless-faas-workbench) suite. The evaluation scripts are located
in the `scripts/` directory, and the benchmark code is in the `evaluation/` directory.

### Scripts Overview

- For cloud deployment, configure and deploy AWS infrastructure:
   ```bash
   aws configure
   ./scripts/1_deploy.sh <n_workers>
   ```
    - `n_workers`: Number of worker instances

- Set up workers and scheduler:
   ```bash
   ./scripts/2_setup.sh <n_workers> <n_copies>
   ```
    - `n_workers`: Number of worker instances
    - `n_copies`: Number of function replicas

- Clean up and teardown:
   ```bash
   ./scripts/3_clean.sh -m <mode>
   ```
    - `mode`: `local` or `cloud`.

- To run an automated experiment (deployment, setup, cleanup) and export results:
    ```bash
    ./scripts/run_experiments.sh  -m <mode>
                                  -n <n_iterations>
                                  <n_workers> <n_copies>
    ```
    - `mode` (optional): `local` or `cloud`, default is `local`
    - `n_iterations` (optional): Number of iterations, default is 1
    - `n_workers`: Number of worker instances
    - `n_copies`: Number of function replicas

### Load Testing

You can perform load testing using the `k6` tool:

```bash
k6 run  -e N_COPIES=<n_copies>
        -e SEED=<seed>
        -e SCHEDULER_DNS=<scheduler_url>
        --out json=<result_path>
        evaluation/load_test.js
```

### Plot Experimental Results

1. Install Python and set up a virtual environment:
   ```bash
   sudo apt-get install python3 python3-venv
   python3 -m venv .venv
   source .venv/bin/activate
   ```

2. Install required Python packages:
   ```bash
   pip install -r requirements.txt
   ```

3. Plot the experimental results:
    ```bash
    python3 evaluation/plot.py
    ```

You can also find our raw experimental results in the `results.zip` file.

### Supported Load Balancing Strategies

- **Pull-Based:** Idle workers proactively request new tasks
- **Consistent Hashing with Bounded Loads**: Distributes requests based on a hash function while limiting maximum load
  per worker.
- **Random**: Routes requests to a random worker.
- **Least Connections**: Routes requests to the worker with the fewest active connections at the time.

### Changes to OpenLambda

We did the following changes to OpenLambda: (i) added endpoint configuration for the scheduler, (ii) introduced a
notification system for sandbox destruction, and (iii) fixes related to cloud deployment and package pulling (see
[open-lambda-mod](open-lambda-mod) directory). For convenience, we already provide an executable binary
for `GOOS=linux GOARCH=amd64` with these changes. If your target platform differs, or you wish to reproduce the
executable that we provide, follow the usage instructions in [open-lambda-mod/README.md](open-lambda-mod/README.md).

## License

This project is licensed under the [Apache License 2.0](LICENSE).

## Acknowledgments

This project extends the [olscheduler](https://github.com/disel-espol/olscheduler)
for [OpenLambda](https://github.com/open-lambda/open-lambda) and integrates benchmarks from
[FunctionBench](https://github.com/ddps-lab/serverless-faas-workbench), which are all licensed under
the [Apache License 2.0](LICENSE).
