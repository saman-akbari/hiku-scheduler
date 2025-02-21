# OpenLambda

We did the following changes to [OpenLambda](https://github.com/open-lambda/open-lambda): (i) added endpoint
configuration for the scheduler, (ii) introduced a notification system for sandbox destruction, and
(iii) fixes related to cloud deployment and package pulling. For convenience, we already provide an executable binary
for `GOOS=linux GOARCH=amd64` with these changes.

If your target platform differs, or you wish to reproduce the executable that we provide, follow the usage instructions
below.

## Usage

1. Clone and check out the `open-lambda` repository:
   ```bash
   git clone https://github.com/open-lambda/open-lambda.git open-lambda
   cd open-lambda
   git checkout 0a834cee321fda36767775653394e0d6b5f00a2c
   ```

2. Replace the corresponding files in the cloned `open-lambda` repository with the ones in `open-lambda-mod`.

3. Build the OpenLambda executable:
   ```bash
   make ol
   ```

## Acknowledgments

We modify files from [OpenLambda](https://github.com/open-lambda/open-lambda), licensed under
the [Apache License 2.0](LICENSE).
