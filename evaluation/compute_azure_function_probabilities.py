import csv
import json
import numpy as np

# Place the .txt file in the root directory of this project:
# https://github.com/Azure/AzurePublicDataset/raw/master/data/AzureFunctionsInvocationTraceForTwoWeeksJan2021.rar
DATASET_FILENAME = "AzureFunctionsInvocationTraceForTwoWeeksJan2021.txt"
RESULTS_FILENAME = "evaluation/azure_function_probabilities.json"


def calculate_probability_distribution(data):
    unique_functions, counts = np.unique(data, return_counts=True)
    total_invocations = np.sum(counts)
    probabilities = counts / total_invocations
    return dict(zip(unique_functions, probabilities))


if __name__ == "__main__":
    with open(DATASET_FILENAME, 'r', encoding='utf-8') as csvfile:
        reader = csv.reader(csvfile)
        azure2021 = [row for row in reader]

    azure2021.pop(0)
    azure2021 = np.array(azure2021)
    probability_distribution = calculate_probability_distribution(azure2021[:, 1])

    function_probabilities = {}
    for function in probability_distribution.keys():
        function_probabilities[function] = {
            "probability": probability_distribution[function]
        }

    with open(RESULTS_FILENAME, "w") as file:
        json.dump(function_probabilities, file, indent=4)

    print("Results written to", RESULTS_FILENAME)
