import json


def validate_latency_data(data, start_type, benchmark, expected_count=20):
    if len(data[start_type][benchmark]) != expected_count:
        raise ValueError(
            f"Sanity check failed for {start_type} -> {benchmark}: expected {expected_count} elements, found {len(data[start_type][benchmark])}."
        )


def calculate_benchmark_averages(data):
    averages = {
        "cold_start": {},
        "warm_start": {}
    }

    for start_type in ["cold_start", "warm_start"]:
        for benchmark, latencies in data[start_type].items():
            validate_latency_data(data, start_type, benchmark)
            averages[start_type][benchmark] = round(sum(latencies) / len(latencies))

    return averages


def calculate_cold_warm_factor(averages):
    total_cold_start_avg = sum(averages["cold_start"].values()) / len(averages["cold_start"])
    total_warm_start_avg = sum(averages["warm_start"].values()) / len(averages["warm_start"])
    return total_cold_start_avg / total_warm_start_avg


with open('results/cold_and_warm_start_latency.json', 'r') as f:
    data = json.load(f)

averages = calculate_benchmark_averages(data)
cold_warm_factor = calculate_cold_warm_factor(averages)

print("Averages for each benchmark (cold and warm starts):")
print(json.dumps(averages, indent=4))
print(f"\nOverall, cold starts are on average {cold_warm_factor:.2f} times slower than warm starts.")
