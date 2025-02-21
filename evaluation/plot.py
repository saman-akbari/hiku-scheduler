import logging
import math
import os
from collections import defaultdict
from datetime import datetime

import matplotlib.pyplot as plt
import numpy as np
import orjson

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
    handlers=[
        logging.StreamHandler()
    ]
)

plt.style.use("evaluation/scientific.mplstyle")

RESULTS_PATH = "results"
SAVE_PATH = "plots"
if not os.path.exists(SAVE_PATH):
    os.makedirs(SAVE_PATH)

SCHEDULING_STRATEGIES = [
    "pull-based",
    "hashing-bounded",
    "least-connections",
    "random"
]

LABELS = {
    "pull-based": "Pull-Based",
    "hashing-bounded": "CH-BL",
    "least-connections": "Least Connections",
    "random": "Random"
}


def average_latency():
    """
    Plots average response latencies.
    """
    average_latencies = {strategy: [] for strategy in SCHEDULING_STRATEGIES}

    for strategy in SCHEDULING_STRATEGIES:
        latencies = []
        timestamp_dirs = os.listdir(os.path.join(RESULTS_PATH, strategy))

        for timestamp_dir in timestamp_dirs:
            log_file = os.path.join(RESULTS_PATH, strategy, timestamp_dir, "load_test.json")

            with open(log_file, 'r') as f:
                for line in f:
                    try:
                        data = orjson.loads(line.strip())
                        if data["metric"] == "http_req_duration" and data["type"] == "Point" and \
                                data["data"]["tags"]["status"] == "200":
                            latencies.append(data["data"]["value"])
                    except orjson.JSONDecodeError:
                        logging.error(f"Failed to parse JSON data: {line}")

        average_latency = np.mean(latencies) if latencies else 0
        average_latencies[strategy].append(average_latency)

    fig, ax = plt.subplots()
    for strategy, avg_latency in average_latencies.items():
        ax.bar(strategy, avg_latency[0], color='#212427', width=0.4)
        ax.text(strategy, avg_latency[0] + 0.5, f"{int(round(avg_latency[0]))}", ha='center', va='bottom')

    ax.set_ylabel("Average Latency (ms)")
    labels = LABELS.copy()
    labels["least-connections"] = "Least\nConnections"

    plt.xticks(range(len(SCHEDULING_STRATEGIES)), [labels[s] for s in SCHEDULING_STRATEGIES])
    max_y = max([avg_latency[0] for avg_latency in average_latencies.values()])
    plt.yticks(range(0, math.ceil(max_y / 250) * 250 + 1, 250))

    plt.savefig(os.path.join(SAVE_PATH, "average_latency.pdf"))
    plt.show()


def latency_cdf():
    """
    Plots the cumulative distribution function of the response latency.
    """
    if not os.path.exists(SAVE_PATH):
        os.makedirs(SAVE_PATH)

    for strategy in SCHEDULING_STRATEGIES:
        latencies = []
        timestamp_dirs = os.listdir(os.path.join(RESULTS_PATH, strategy))

        for timestamp_dir in timestamp_dirs:
            with open(os.path.join(RESULTS_PATH, strategy, timestamp_dir, "load_test.json")) as f:
                for line in f:
                    try:
                        data = orjson.loads(line.strip())

                        if data["metric"] == "http_req_duration" and data["type"] == "Point" and \
                                data["data"]["tags"]["status"] == "200":
                            latencies.append(data["data"]["value"])
                    except orjson.JSONDecodeError:
                        logging.error(f"Failed to parse JSON data: {line}")

        latencies.sort()

        cdf = []
        for i, latency in enumerate(latencies):
            cdf.append((i + 1) / len(latencies))

        plt.plot(latencies, cdf, label=LABELS[strategy])

    plt.xlabel("Latency (ms)")
    plt.xscale('symlog', linthresh=1000)
    plt.xlim(0, 10000)
    plt.ylim(0, 1)
    plt.ylabel("Percent of Requests")

    plt.gca().yaxis.set_major_formatter(plt.FuncFormatter(lambda x, _: r'$%d$' % (x * 100)))
    plt.gca().xaxis.set_minor_locator(plt.FixedLocator([100, 200, 300, 400, 500, 600, 700, 800, 900,
                                                        1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000]))

    plt.gca().xaxis.set_major_formatter(plt.ScalarFormatter())
    plt.gca().xaxis.get_major_ticks()[0].label1.set_visible(False)

    plt.legend(fontsize='small')

    plt.savefig(os.path.join(SAVE_PATH, "latency_cdf.pdf"))
    plt.show()


def tail_latency(percentiles):
    """
    Plots the tail response latency.

    Parameters:
        percentiles (list): List of percentiles to calculate.
    """
    if not os.path.exists(SAVE_PATH):
        os.makedirs(SAVE_PATH)

    strategy_latencies = {strategy: [] for strategy in SCHEDULING_STRATEGIES}

    for strategy in SCHEDULING_STRATEGIES:
        latencies = []
        timestamp_dirs = os.listdir(os.path.join(RESULTS_PATH, strategy))

        for timestamp_dir in timestamp_dirs:
            with open(os.path.join(RESULTS_PATH, strategy, timestamp_dir, "load_test.json")) as f:
                for line in f:
                    try:
                        data = orjson.loads(line.strip())

                        if data["metric"] == "http_req_duration" and data["type"] == "Point" and \
                                data["data"]["tags"]["status"] == "200":
                            latencies.append(data["data"]["value"])
                    except orjson.JSONDecodeError:
                        logging.error(f"Failed to parse JSON data: {line}")

        latencies = np.sort(latencies)
        strategy_latencies[strategy] = [np.percentile(latencies, p) for p in percentiles]

    fig, ax = plt.subplots()
    x = np.arange(len(percentiles))
    width = 0.1

    for idx, strategy in enumerate(SCHEDULING_STRATEGIES):
        ax.bar(x + idx * width, strategy_latencies[strategy], width, label=LABELS[strategy])

    ax.set_xlabel("Percentile")
    ax.set_ylabel("Latency (ms)")
    ax.set_xticks(x + width * (len(SCHEDULING_STRATEGIES) - 1) / 2)
    ax.set_xticklabels([f'p{percentile}' for percentile in percentiles])
    ax.legend(fontsize='small')

    max_y = max([max(strategy_latencies[s]) for s in SCHEDULING_STRATEGIES])
    plt.yticks(range(0, math.ceil(max_y / 1000) * 1000 + 1, 1000))
    plt.ylim(0, math.ceil(max_y / 1000) * 1000)

    # stats
    for strategy, latencies in strategy_latencies.items():
        print(f"{strategy} {latencies}")

    plt.savefig(os.path.join(SAVE_PATH, "tail_latency.pdf"))
    plt.show()


def load_imbalance(num_workers, n=1):
    """
    Plots the load imbalance (coefficient of variation of the number of requests assigned per worker per second).

    Parameters:
        num_workers (int): Number of workers.
        n (int): Number of seconds over which to average the data.
    """
    imbalance_data = _calculate_load_imbalance(SCHEDULING_STRATEGIES, num_workers)

    imbalance_mean = {}
    for strategy, imb_data in imbalance_data.items():
        summed_data = defaultdict(float)
        timestamp_count = defaultdict(int)

        # Sum the data across all timestamps
        for timestamp, second_data in imb_data.items():
            for second, value in second_data.items():
                summed_data[second] += value
                timestamp_count[second] += 1

        # Calculate the average over timestamps for each second
        averaged_data = {second: summed_data[second] / timestamp_count[second] for second in summed_data}

        # Average the data over n seconds
        final_averaged_data = {}
        total_value = count = 0
        for second in sorted(averaged_data.keys()):
            total_value += averaged_data[second]
            count += 1
            if count == n:  # Every n seconds, store the average
                final_averaged_data[second + 1] = total_value / n  # Shift duration by 1
                total_value = count = 0

        # Handle any remaining seconds that don't fit into the last n-group
        if count > 0:
            final_averaged_data[second + 1] = total_value / count  # Shift duration by 1
        imbalance_mean[strategy] = final_averaged_data

    fig, ax = plt.subplots()
    for strategy, imb_data in imbalance_mean.items():
        durations, values = zip(*sorted(imb_data.items()))
        ax.plot(durations, values, label=LABELS[strategy])

    ax.set_xlabel("Duration (sec)")
    ax.set_ylabel("Coefficient of Variation")

    plt.xlim(0, 300)
    max_y = max(max(v for v in imb_data.values()) for imb_data in imbalance_mean.values())
    plt.yticks(np.arange(0, math.ceil(max_y) + 1, 0.2))
    plt.ylim(0, 1)

    plt.legend(fontsize='small', loc='upper left')
    plt.savefig(os.path.join(SAVE_PATH, f"load_imbalance.pdf"))
    plt.show()


def average_load_imbalance(num_workers):
    """
    Plots the average load imbalance.

    Parameters:
        num_workers (int): Number of workers used in the experiment.
    """
    imbalance_data = _calculate_load_imbalance(SCHEDULING_STRATEGIES, num_workers)

    average_imbalance_per_strategy = {}

    for strategy, imb_data in imbalance_data.items():
        total_value = 0
        total_count = 0

        # Sum up all values and keep count across timestamps and workers
        for timestamp, second_data in imb_data.items():
            for second, value in second_data.items():
                total_value += value
                total_count += 1

        # Calculate the average for this strategy
        if total_count > 0:
            average_imbalance_per_strategy[strategy] = total_value / total_count
        else:
            average_imbalance_per_strategy[strategy] = 0

    fig, ax = plt.subplots()
    strategies_list = list(average_imbalance_per_strategy.keys())
    averages_list = list(average_imbalance_per_strategy.values())
    ax.bar(strategies_list, averages_list, color='#212427', width=0.4)

    for i, v in enumerate(averages_list):
        ax.text(i, v + 0.01, f"{v:.2f}", ha='center', va='bottom', color='black')

    ax.set_ylabel("Average Coefficient of Variation")
    labels = LABELS.copy()
    labels["least-connections"] = "Least\nConnections"

    plt.xticks(range(len(strategies_list)), [labels[s] for s in strategies_list])
    max_y = max(averages_list)
    plt.yticks(np.arange(0, math.ceil(max_y) + 1, 0.2))
    plt.ylim(0, 1)

    # stats
    for strategy, value in average_imbalance_per_strategy.items():
        print(f"{strategy} {value}")

    plt.savefig(os.path.join(SAVE_PATH, f"average_load_imbalance.pdf"))
    plt.show()


def throughput(cut_off):
    """
    Plots the cumulative number of requests over time.
    """
    cumulative_requests = {strategy: {} for strategy in SCHEDULING_STRATEGIES}

    for strategy in SCHEDULING_STRATEGIES:
        timestamps = sorted(os.listdir(os.path.join(RESULTS_PATH, strategy)))

        for timestamp in timestamps:
            log_data = _load_log_file(os.path.join(RESULTS_PATH, strategy, timestamp), "balancer.log")
            start_time = None

            cumulative_count = 0
            cumulative_requests[strategy][timestamp] = {}

            for line in log_data:
                if "Response Status: 200" in line:
                    timestamp_str = line.split(" ")[0] + " " + line.split(" ")[1]
                    timestamp_line = datetime.strptime(timestamp_str, "%Y/%m/%d %H:%M:%S")

                    if start_time is None:
                        start_time = timestamp_line

                    elapsed_seconds = int((timestamp_line - start_time).total_seconds())
                    if elapsed_seconds > cut_off:
                        break

                    cumulative_count += 1
                    cumulative_requests[strategy][timestamp][elapsed_seconds] = cumulative_count

    # fill in missing seconds
    for strategy, data in cumulative_requests.items():
        for timestamp, values in data.items():
            # within
            last_value = 0
            for i in range(max(values.keys()) + 1):
                if i not in values:
                    values[i] = last_value
                else:
                    last_value = values[i]

            # last values
            for i in range(cut_off + 1):
                if i not in values:
                    values[i] = values[i - 1]

    # average cumulative requests, per strategy, per second
    average_cumulative_requests = {}
    for strategy, data in cumulative_requests.items():
        average_cumulative_requests[strategy] = {}
        for i in range(cut_off + 1):
            average_cumulative_requests[strategy][i] = sum([values[i] for values in data.values()]) / len(data)

    fig, ax = plt.subplots()
    for strategy, data in average_cumulative_requests.items():
        x_values, y_values = zip(*sorted(data.items()))
        ax.plot(x_values, y_values, label=LABELS[strategy])

    ax.set_xlabel("Elapsed Time (sec)")
    ax.set_ylabel("Processed Requests")
    plt.xlim(0, cut_off + 1)
    max_y = max(max(v for v in data.values()) for data in average_cumulative_requests.values())
    plt.yticks(range(0, math.ceil(max_y / 5000) * 5000 + 1, 5000))
    plt.ylim(0, math.ceil(max_y / 5000) * 5000)
    plt.legend(fontsize='small')

    # stats
    for strategy, data in cumulative_requests.items():
        for timestamp, values in data.items():
            print(f"{strategy} {timestamp} {values[cut_off]}")

    plt.savefig(os.path.join(SAVE_PATH, "throughput.pdf"))
    plt.show()


def average_throughput():
    """
    Plots the average throughput.
    """
    throughput_data = _calculate_throughput()
    average_data = _calculate_average_throughput(throughput_data)

    fig, ax = plt.subplots()
    for strategy, avg_value in average_data.items():
        ax.bar(LABELS[strategy], avg_value, color='#212427', width=0.4)
        ax.text(LABELS[strategy], avg_value + 0.5, f"{avg_value:.1f}", ha='center', va='bottom')

    ax.set_ylabel("Requests per Second")

    labels = LABELS.copy()
    labels["least-connections"] = "Least\nConnections"
    plt.xticks(range(len(SCHEDULING_STRATEGIES)), [labels[s] for s in SCHEDULING_STRATEGIES])

    max_y = max(average_data.values())
    plt.yticks(range(0, math.ceil(max_y / 20) * 20 + 1, 20))
    plt.ylim(0, math.ceil(max_y / 20) * 20)

    plt.savefig(os.path.join(SAVE_PATH, "average_throughput.pdf"))
    plt.show()


def concurrency(virtual_users, durations, cut_off):
    """
    Plots requests per second (RPS) for different concurrency levels.

    Parameters:
        virtual_users: List of virtual user counts (integers).
        durations: List of durations (integers).
        cut_off: Time in seconds to cut off the data.
    """
    rps_data = defaultdict(lambda: defaultdict(list))

    for strategy in SCHEDULING_STRATEGIES:
        timestamps = sorted(os.listdir(os.path.join(RESULTS_PATH, strategy)))

        for timestamp in timestamps:
            log_data = _load_log_file(os.path.join(RESULTS_PATH, strategy, timestamp), "balancer.log")
            start_time = None
            request_counts = [0] * len(virtual_users)

            for line in log_data:
                if "Response Status: 200" in line:
                    timestamp_str = line.split(" ")[0] + " " + line.split(" ")[1]
                    log_time = datetime.strptime(timestamp_str, "%Y/%m/%d %H:%M:%S")

                    if start_time is None:
                        start_time = log_time

                    elapsed_seconds = int((log_time - start_time).total_seconds())

                    if elapsed_seconds > cut_off:
                        break

                    # Accumulate requests for each virtual user count
                    for i in range(len(virtual_users)):
                        if i < len(virtual_users) - 1:
                            if sum(durations[:i]) <= elapsed_seconds < sum(durations[:i + 1]):
                                request_counts[i] += 1
                        else:
                            if elapsed_seconds >= sum(durations[:-1]):
                                request_counts[i] += 1

            for i in range(len(durations)):
                if durations[i] > 0:
                    rps_value = request_counts[i] / durations[i]
                    rps_data[strategy][virtual_users[i]].append(rps_value)

    fig, ax = plt.subplots()
    for strategy, user_data in rps_data.items():
        x_values = []
        avg_rps_values = []

        for vu in sorted(user_data.keys()):
            # Calculate average RPS for each virtual user count
            avg_rps = np.mean(user_data[vu])
            x_values.append(vu)
            avg_rps_values.append(avg_rps)

        ax.plot(x_values, avg_rps_values, marker='o', label=LABELS[strategy], clip_on=False)

    ax.set_xlabel("Virtual Users")
    ax.set_ylabel("Requests Per Second")

    plt.xlim(0, max(virtual_users))
    plt.xticks(virtual_users)
    max_y = max(max(v) for strategy in rps_data.values() for v in strategy.values())
    plt.yticks(range(0, math.ceil(max_y / 20) * 20 + 1, 20))
    plt.ylim(0, math.ceil(max_y / 20) * 20)

    plt.legend(fontsize='small', loc='upper left', bbox_to_anchor=(0, 1.05))

    # stats
    for strategy, user_data in rps_data.items():
        for vu, rps in user_data.items():
            print(f"{strategy} {vu} {round(np.mean(rps), 2)}")

    plt.savefig(os.path.join(SAVE_PATH, "concurrency.pdf"))
    plt.show()


def cold_starts():
    """ Plots cold start percentages. """
    # Dictionary to store strategy: (cold starts, warm starts, total invocations)
    data = {}
    for strategy in SCHEDULING_STRATEGIES:
        cold_starts = 0
        total_invocations = 0

        strategy_dir = os.path.join(RESULTS_PATH, strategy)
        timestamp_dirs = os.listdir(strategy_dir)
        for timestamp_dir in timestamp_dirs:
            worker_logs_path = os.path.join(RESULTS_PATH, strategy, timestamp_dir)
            for filename in os.listdir(worker_logs_path):
                if filename.startswith("worker-"):
                    with open(os.path.join(worker_logs_path, filename), "r") as f:
                        for line in f:
                            if "Creating new sandbox" in line:
                                cold_starts += 1
                            elif "LambdaFunc.Invoke" in line:
                                total_invocations += 1

        data[strategy] = (cold_starts, total_invocations)

    x = np.arange(len(SCHEDULING_STRATEGIES))
    cold_starts = [data[s][0] for s in SCHEDULING_STRATEGIES]
    total_invocations = [data[s][1] for s in SCHEDULING_STRATEGIES]

    for i in range(len(SCHEDULING_STRATEGIES)):
        plt.bar(x[i], cold_starts[i] / total_invocations[i] * 100, width=0.4, color='#212427')

    for i, strategy in enumerate(SCHEDULING_STRATEGIES):
        plt.text(i, cold_starts[i] / total_invocations[i] * 100 + 1,
                 f"{cold_starts[i] / total_invocations[i] * 100:.0f}%",
                 ha='center', va='bottom', color='black')

    labels = LABELS.copy()
    labels["least-connections"] = "Least\nConnections"
    plt.xticks(range(len(SCHEDULING_STRATEGIES)), [labels[s] for s in SCHEDULING_STRATEGIES])
    plt.ylabel('Cold Starts (\%)')
    plt.ylim(0, 100)

    plt.savefig(os.path.join(SAVE_PATH, "cold_starts.pdf"))
    plt.show()


def scheduling_overhead():
    """Plots the scheduling overhead."""
    overheads = {}
    for strategy in SCHEDULING_STRATEGIES:
        overheads[strategy] = []
        timestamp_dirs = os.listdir(os.path.join("results", strategy))
        for timestamp_dir in timestamp_dirs:
            log_file = os.path.join("results", strategy, timestamp_dir, "balancer.log")
            with open(log_file, 'r') as f:
                for line in f:
                    if "Selected worker" in line:
                        parts = line.split()
                        overhead = int(parts[-3])
                        overheads[strategy].append(overhead)
    fig, ax = plt.subplots()
    labels = LABELS.copy()
    labels["least-connections"] = "Least\nConnections"
    plt.xticks(range(len(SCHEDULING_STRATEGIES)), [labels[s] for s in SCHEDULING_STRATEGIES])
    for strategy in SCHEDULING_STRATEGIES:
        ax.bar(labels[strategy], np.mean(overheads[strategy]), color='#212427', width=0.4)
    ax.set_ylabel("Scheduling Overhead (ns)")
    plt.savefig(os.path.join(SAVE_PATH, "scheduling_overhead.pdf"), format="pdf")
    plt.show()

    # stats
    for strategy, overhead in overheads.items():
        print(f"{strategy} {np.mean(overhead)}")


def _convert_to_timestamp(date_time_str):
    """ Converts a date and time string in YYYY/MM/DD HH:MM:SS format to a timestamp. """
    date_time_format = "%Y/%m/%d %H:%M:%S"
    timestamp = datetime.strptime(date_time_str, date_time_format)
    return timestamp


def _calculate_load_imbalance(strategies, num_workers):
    """
    Calculates the imbalance data.

    Parameters:
        strategies (list): List of scheduling strategies.
        num_workers (int): Number of workers.

    Returns:
        dict: A dictionary where each strategy contains timestamped imbalance data.
    """
    imbalance_data = {strategy: {} for strategy in strategies}
    for strategy in strategies:
        timestamp_dirs = os.listdir(os.path.join(RESULTS_PATH, strategy))

        for timestamp in timestamp_dirs:
            log_file = os.path.join(RESULTS_PATH, strategy, timestamp, "balancer.log")
            timestamps, workers = _parse_log_file(log_file)

            imbalance_data[strategy][timestamp] = _calculate_coefficient_of_variation(timestamps, workers, num_workers)
    return imbalance_data


def _calculate_coefficient_of_variation(timestamps, workers, num_workers):
    """
    Calculates the coefficient of variation of the number of tasks assigned to each worker per second.

    Parameters:
        timestamps (list): List of timestamps.
        workers (list): List of workers corresponding to the timestamps.
        num_workers (int): Number of workers.

    Returns:
        dict: A dictionary where keys are seconds and values are the coefficient of variation.
    """
    load_per_worker = defaultdict(lambda: defaultdict(int))
    start_time = timestamps[0]

    # Populate worker load per second
    for timestamp, worker in zip(timestamps, workers):
        second = int((timestamp - start_time).total_seconds())
        load_per_worker[second][worker] += 1

    data_per_second = {}
    max_second = max(load_per_worker.keys(), default=0)

    # Compute CV for each second
    for second in range(max_second + 1):
        worker_counts = load_per_worker.get(second, {})
        total_tasks = sum(worker_counts.values())

        if total_tasks > 0:
            task_counts = np.array(list(worker_counts.values()))
            mean_tasks = total_tasks / num_workers
            std_dev_tasks = np.std(task_counts)

            cv = std_dev_tasks / mean_tasks if mean_tasks > 0 else 0

            data_per_second[second] = cv
        else:
            # No tasks assigned at this second
            data_per_second[second] = 0

    return data_per_second


def _load_log_file(directory, file):
    """
    Loads a log file and returns the data as a list of lines.
    """
    log_file_path = os.path.join(directory, file)
    if not os.path.isfile(log_file_path):
        raise FileNotFoundError(f"Log file not found: {log_file_path}")

    with open(log_file_path, "r") as log_file:
        log_data = log_file.readlines()

    return log_data


def _parse_log_file(filename):
    """
    Parses a balancer log file and returns a list of timestamps and assigned workers.
    """
    timestamps = []
    workers = []
    with open(filename) as f:
        for line in f:
            if "Selected worker" in line:
                parts = line.split()
                timestamp = _convert_to_timestamp(parts[0] + " " + parts[1])
                timestamps.append(timestamp)

                worker = parts[4]
                workers.append(worker.split(":")[-1])
    return timestamps, workers


def _calculate_throughput():
    """
    Calculates the throughput data.
    """
    throughput_timestamp_data = defaultdict(lambda: defaultdict(int))
    throughput_data = defaultdict(lambda: defaultdict(int))

    for strategy in SCHEDULING_STRATEGIES:
        timestamps = os.listdir(os.path.join(RESULTS_PATH, strategy))

        for timestamp in timestamps:
            log_data = _load_log_file(os.path.join(RESULTS_PATH, strategy, timestamp), "load_test.json")

            start_time = None
            for line in log_data:
                data = orjson.loads(line.strip())
                if data["metric"] == "http_req_duration" and data["type"] == "Point" and \
                        data["data"]["tags"]["status"] == "200":
                    timestamp_str = data["data"]["time"]
                    timestamp = datetime.strptime(timestamp_str[:19], "%Y-%m-%dT%H:%M:%S")

                    if start_time is None:
                        start_time = timestamp

                    elapsed_seconds = int((timestamp - start_time).total_seconds())
                    throughput_timestamp_data[strategy][elapsed_seconds] += 1

        for second, count in throughput_timestamp_data[strategy].items():
            throughput_data[strategy][second] = count / len(timestamps)

    return throughput_data


def _calculate_average_throughput(throughput_data):
    """
    Calculates the average throughput.
    """
    average_data = {}

    for strategy, data in throughput_data.items():
        total_requests = sum(data.values())
        total_seconds = len(data)
        average_data[strategy] = total_requests / total_seconds if total_seconds > 0 else 0

    return average_data


def main():
    latency_cdf()
    average_latency()
    tail_latency(percentiles=[90, 95, 99])
    cold_starts()
    load_imbalance(num_workers=5, n=8)
    average_load_imbalance(num_workers=5)
    throughput(cut_off=299)
    average_throughput()
    concurrency(virtual_users=[20, 50, 100], durations=[100, 100, 100], cut_off=299)
    scheduling_overhead()


if __name__ == "__main__":
    main()
