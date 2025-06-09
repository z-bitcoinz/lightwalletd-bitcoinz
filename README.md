[![Build Status](https://github.com/bitcoinz-xyz/lightwalletd/actions/workflows/ci.yml/badge.svg)](https://github.com/bitcoinz-xyz/lightwalletd/actions)
[![codecov](https://codecov.io/gh/bitcoinz-xyz/lightwalletd/branch/main/graph/badge.svg)](https://codecov.io/gh/bitcoinz-xyz/lightwalletd)

# Security Disclaimer

lightwalletd is under active development, some features are more stable than
others. The code has not been subjected to a thorough review by an external
auditor, and recent code changes have not yet received security review from
the BitcoinZ security team.

Developers should familiarize themselves with the wallet app threat
model, since it contains important information about the security and privacy
limitations of light wallets that use lightwalletd.

---

# Overview

[lightwalletd](https://github.com/bitcoinz-xyz/lightwalletd) is a backend service that provides a bandwidth-efficient interface to the BitcoinZ blockchain. Currently, lightwalletd supports the Sapling protocol version and beyond as its primary concern. The intended purpose of lightwalletd is to support the development and operation of mobile-friendly shielded light wallets.

lightwalletd is a backend service that provides a bandwidth-efficient interface to the BitcoinZ blockchain for mobile and other wallets, enabling efficient access to the BitcoinZ network.

To view status of [CI pipeline](https://github.com/bitcoinz-xyz/lightwalletd/actions)

To view detailed [Codecov](https://codecov.io/gh/bitcoinz-xyz/lightwalletd) report

Documentation for lightwalletd clients (the gRPC interface) is in `docs/rtd/index.html`. The current version of this file corresponds to the two `.proto` files; if you change these files, please regenerate the documentation by running `make doc`, which requires docker to be installed. 

# Local/Developer docker-compose Usage

[docs/docker-compose-setup.md](./docs/docker-compose-setup.md)

# Local/Developer Usage

## BitcoinZd

You must start a local instance of `bitcoinzd`, and its `.bitcoinz/bitcoinz.conf` file must include the following entries
(set the user and password strings accordingly):
```
txindex=1
lightwalletd=1
experimentalfeatures=1
rpcuser=xxxxx
rpcpassword=xxxxx
```

The `bitcoinzd` can be configured to run `mainnet` or `testnet` (or `regtest`). If you stop `bitcoinzd` and restart it on a different network (switch from `testnet` to `mainnet`, for example), you must also stop and restart lightwalletd.

It's necessary to run `bitcoinzd --reindex` one time for these options to take effect. This typically takes several hours, and requires more space in the `.bitcoinz` data directory.

## Lightwalletd

This assumes you used `git` to download this source code, and that you have a [suitable version of Go installed](https://golang.org/doc/install).

```
$ go run main.go --help
Lightwalletd is a backend service that provides a
         bandwidth-efficient interface to the BitcoinZ blockchain

Usage:
  lightwalletd [flags]

Flags:
      --data-dir string                data directory (such as db) (default "/var/lib/lightwalletd")
      --darkside-timeout int           override 30 minute default darkside timeout (default 30)
      --darkside-very-insecure         run with GRPC-controllable mock bitcoinzd for integration testing (shuts down after 30 minutes)
      --bitcoinz-conf-path string         conf file to pull RPC creds from (default "./bitcoinz.conf")
      --gen-cert-very-insecure         run with self-signed TLS certificate, only for debugging, DO NOT use in production
      --grpc-bind-addr string          the address to listen on (default "127.0.0.1:9067")
  -h, --help                           help for lightwalletd
      --http-bind-addr string          the address to listen on for http status (default "127.0.0.1:8080")
      --log-file string                log file to write to (default "./server.log")
      --log-level uint                 log level (logrus 1-7) (default 6)
      --no-tls-very-insecure           run without the required TLS certificate, only for debugging, DO NOT use in production
      --ping-very-insecure             allow Ping GRPC for testing
      --redownload                     re-fetch all blocks from bitcoinzd; reinitialize local cache files
      --rpchost string                 RPC host
      --rpcpassword string             RPC password
      --rpcport string                 RPC host port
      --rpcuser string                 RPC user name
      --sync-from-height int           re-fetch blocks from bitcoinzd, starting at this height (default -1)
      --tls-cert string                the path to a TLS certificate (default "./cert.pem")
      --tls-key string                 the path to a TLS key file (default "./cert.key")

```

To use a local bitcoinzd instance running on testnet:
```
$ go run main.go --bitcoinz-conf-path ~/.bitcoinz/bitcoinz.conf --log-level 7
```

To use a bitcoinzd instance running on another machine, you can omit the `--bitcoinz-conf-path` option and instead specify the RPC connection options directly:
```
$ go run main.go --log-level 7 --rpcuser=user --rpcpassword=password --rpchost=otherhost --rpcport=1979
```

# Production Usage

Build the server:
```
$ make
```

**x86_64 and ARM64 binaries** are [available for download](https://github.com/bitcoinz-xyz/lightwalletd/releases) for each tagged release.

You'll need to supply a TLS certificate:
```
$ ./lightwalletd --tls-cert cert.pem --tls-key cert.key --bitcoinz-conf-path ~/.bitcoinz/bitcoinz.conf --log-file /var/log/server.log
```

If you want to run on testnet, you need to run bitcoinzd on testnet (add `testnet=1` to `bitcoinz.conf`) and run lightwalletd with the `--bitcoinz-conf-path` flag pointing to the `bitcoinz.conf`.

If you have Docker, you can run a production-stage lightwalletd with
```
$ make docker
$ docker run --init --rm --volume "${PWD}/lightwalletd.yml:/home/lightwalletd/lightwalletd.yml" lightwalletd:latest
```
, which reads the config file `./lightwalletd.yml`. You can also change the configuration file used by changing the path after `--volume`.

# Tests

The tests need a running bitcoinzd. If you're running `bitcoinzd` locally, the tests will use the same bitcoinzd.

For the full test, you need to have the bitcoinzd binary in your $PATH. This is because one test (`TestServerPing`) creates its own bitcoinzd instance to ensure lightwalletd will shut down if bitcoinzd goes away.

Start bitcoinzd in regtest mode:
```
$ bitcoinzd -regtest -cachesize=400 -debug=mempool -debug=net -debug=rpc
```

Run the tests:
```
$ make test
```

or equivalently:
```
$ go test ./...
```

# Troubleshooting

If you're having a problem, run lightwalletd with `--log-level 7` and see if the logs give more information.

## Getting started

First, install [Go >= 1.23](https://golang.org/dl/#stable).

Then, clone the repo:

```
git clone https://github.com/bitcoinz-xyz/lightwalletd.git
cd lightwalletd
```

Now, build:

```
make
```

This will build the server binary, placing it at `./lightwalletd`.

Use `make help` to see the other available `make` targets.

### File and Directory structure

The main functionality is in the `frontend` package, specifically [`frontend/service.go`](./frontend/service.go). The `parser` package is useful for parsing BitcoinZ block and transaction data. The `grpcclient` package is useful for connecting to lightwalletd from another program in Go.

## BitcoinZ Community Support

If you need help or have questions about BitcoinZ lightwalletd, please join our community:

- **Discord**: [BitcoinZ Community Server](https://discord.gg/L9wRFTFVCp)
- **Telegram**: [BitcoinZ Official](https://t.me/BitcoinZOfficial)
- **GitHub Issues**: [Report bugs or request features](https://github.com/bitcoinz-xyz/lightwalletd/issues)
- **Reddit**: [r/BitcoinZ](https://www.reddit.com/r/BitcoinZ/)

For development questions and technical discussions, please use GitHub Issues or join our development channels in Discord.

## License

This code is distributed under the terms of the MIT license. See the accompanying file LICENSE for details.

## Contributing

We welcome contributions to BitcoinZ lightwalletd! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute.

## Building and Running with Docker

Refer to [docs/docker-run.md](docs/docker-run.md) for guidance on building and running with Docker.