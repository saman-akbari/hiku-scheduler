import logging
import os

import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
    handlers=[
        logging.StreamHandler()
    ]
)

plt.style.use("evaluation/scientific.mplstyle")
SAVE_PATH = "plots"
if not os.path.exists(SAVE_PATH):
    os.makedirs(SAVE_PATH)


def _read_csv(filename):
    return pd.read_csv(filename, encoding='utf-8')


def stats(azure2021):
    logging.info(azure2021.describe(include='all'))
    logging.info(f"Number of unique apps: {azure2021['app'].nunique()}")
    logging.info(f"Number of unique functions: {azure2021['func'].nunique()}")
    logging.info(f"Minimum duration: {azure2021['duration'].min()}")
    logging.info(f"Maximum duration: {azure2021['duration'].max()}")
    logging.info(f"Average duration: {azure2021['duration'].mean()}")


def skewed_function_popularity(azure2021):
    # Extract the function names
    funcs = azure2021['func']

    # Count the number of invocations for each function
    func_counts = funcs.value_counts().sort_values(ascending=False)
    invocation_counts = func_counts.values
    func_names = np.arange(1, len(func_counts) + 1)

    # Plot
    plt.figure(figsize=(6.4, 3.9))
    plt.bar(func_names, invocation_counts, width=10)

    plt.xlabel("Function Rank")
    plt.xticks([1, 100, 200, 300, 400], ['$1$', '$100$', '$200$', '$300$', '$400$'])
    plt.ylabel("Invocations")
    plt.xlim(1, len(func_names))
    plt.ylim(0, max(invocation_counts))
    plt.yticks([0, 200000, 400000, 600000],
               ['$0$', '$200000$', '$400000$', '$600000$'])

    plt.text(0.95, 0.95, f'Total Functions: {len(func_counts)}',
             transform=plt.gca().transAxes, ha='right', va='top')

    plt.savefig(os.path.join(SAVE_PATH, "skewed_function_popularity.pdf"))
    plt.show()


def heterogeneous_function_performance(azure2021):
    # Extract the function names and execution times
    df = azure2021[['func', 'duration']].copy()
    df['exec_time'] = df['duration'].astype(float)

    # Calculate statistics for each function
    grouped = df.groupby('func')['exec_time']
    func_stats = grouped.agg(['mean', 'std'])

    # Scatter plot with error bars
    x = np.arange(1, len(func_stats) + 1)
    fig, ax = plt.subplots(figsize=(6.4, 3.9))
    ax.errorbar(x, func_stats['mean'], yerr=func_stats['std'], fmt='o', ecolor='r', capsize=5, markersize=4,
                linestyle='None')

    ax.set_xlabel("Function")
    ax.set_ylabel("Avg. Execution Time (ms)")
    ax.set_xlim(0, len(func_stats) + 1)
    ax.set_ylim(0, 300)

    plt.savefig(os.path.join(SAVE_PATH, "heterogeneous_function_performance.pdf"))
    plt.show()


def bursty_invocations(azure2021):
    # Convert end_timestamp to minutes elapsed
    df = azure2021[['end_timestamp']].copy()
    df['minutes_elapsed'] = (df['end_timestamp'] - df['end_timestamp'].min()) / 60
    df = df.sort_values('minutes_elapsed')

    # Calculate interarrival times (in seconds)
    df['interarrival_time'] = df['minutes_elapsed'].diff() * 60

    # Group interarrival times by minute
    df['minute'] = df['minutes_elapsed'].astype(int)
    grouped = df.groupby('minute')['interarrival_time']
    avg_interarrival_times = grouped.mean()

    # Plot
    plt.figure(figsize=(6.4, 3.9))
    plt.plot(avg_interarrival_times.index, avg_interarrival_times.values, linewidth=1)

    plt.xlabel('Minutes Elapsed')
    plt.xlim(0, avg_interarrival_times.index.max())
    plt.ylabel('Avg. Interarrival Time (sec)')
    plt.ylim(0.01, 100)
    plt.yscale('log')
    plt.yticks([0.01, 0.1, 1, 10, 100], ['$0.01$', '$0.1$', '$1$', '$10$', '$100$'])

    plt.savefig(os.path.join(SAVE_PATH, "bursty_invocations.pdf"))
    plt.show()


def main():
    azure2021 = _read_csv("AzureFunctionsInvocationTraceForTwoWeeksJan2021.txt")
    stats(azure2021)
    skewed_function_popularity(azure2021)
    heterogeneous_function_performance(azure2021)
    bursty_invocations(azure2021)


if __name__ == "__main__":
    main()
