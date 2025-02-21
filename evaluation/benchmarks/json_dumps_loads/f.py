import json
from urllib.request import urlopen, Request
from time import time


def f(event):
    link = event['link']
    req = Request(
        url=link,
        headers={'User-Agent': 'Mozilla/5.0'}
    )

    start = time()
    file = urlopen(req)
    data = file.read().decode("utf-8")
    network = time() - start

    start = time()
    json_data = json.loads(data)
    str_json = json.dumps(json_data, indent=4)
    latency = time() - start

    return {"network": network, "serialization": latency}
