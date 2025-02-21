from time import time
import gzip
import os


def f(event):
    file_size = event['file_size']
    file_write_path = '/tmp/file'

    start = time()
    with open(file_write_path, 'wb') as file:
        file.write(os.urandom(file_size * 1024 * 1024))
    disk_latency = time() - start

    with open(file_write_path, 'rb') as file:
        start = time()
        with gzip.open('/tmp/result.gz', 'wb') as gz:
            gz.writelines(file)
        compress_latency = time() - start

    os.remove(file_write_path)
    os.remove('/tmp/result.gz')

    return {'disk_write': disk_latency, "compress": compress_latency}
