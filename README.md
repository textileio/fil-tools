# Powergate

[![Made by Textile](https://img.shields.io/badge/made%20by-Textile-informational.svg?style=popout-square)](https://textile.io)
[![Chat on Slack](https://img.shields.io/badge/slack-slack.textile.io-informational.svg?style=popout-square)](https://slack.textile.io)
[![GitHub license](https://img.shields.io/github/license/textileio/filecoin.svg?style=popout-square)](./LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/textileio/powergate?style=flat-square)](https://goreportcard.com/report/github.com/textileio/powergate?style=flat-square)
[![GitHub action](https://github.com/textileio/powergate/workflows/Tests/badge.svg?style=popout-square)](https://github.com/textileio/powergate/actions)

Powergate is a multitiered file storage API built on Filecoin and IPFS, and an index builder for Filecoin data. It's designed to be modular and extensible.

Join us on our [public Slack channel](https://slack.textile.io/) for news, discussions, and status updates. [Check out our blog](https://medium.com/textileio) for the latest posts and announcements.

*Warning* This project is still **pre-release** and is not ready for production usage.

## Table of Contents

-   [Prerequisites](#prerequisites)
-   [Design](#design)
-   [Installation](#installation)
-   [Localnet mode](#localnet-mode)
-   [Production setup](#production-setup)
-   [Tests](#tests)
-   [Benchmark](#benchmark)
-   [Contributing](#contributing)
-   [Changelog](#changelog)
-   [License](#license)

## Prerequisites

To build from source, you need to have Go 1.14 or newer installed.

## Design

Powergate is composed of different modules which can be used independently, and compose together to provide other higher-level modules.

Here's a high-level overview of the main components, and how Powergate interacts with IPFS and a Filecoin client:
![Powergate Design](https://user-images.githubusercontent.com/6136245/86490337-6dbd5200-bd3d-11ea-9895-3689dd1c8a8f.png)

Note in the diagram that the Lotus and Filecoin client node _doesn't need_ to be in the same host where Powergate is running. They can, but isn't necessary.

### 📢 Deals module
The Deals module provides a lower layer of abstraction to a Filecoin client node. It provides simple APIs to store, watch, and retrieve data in the Filecoin network. Currently, it interacts with the Lotus client but we have plans to support other Filecoin clients.

### 👷 Indices and Reputation scoring
Powergate builds three indexes related to on-chain and off-chain data.

The _Miners index_ provides processed data regarding registered miners (on-chain and off-chain), such as: total miner power, relative power, online status, geolocation, and more!

The _Ask index_ provides a fast-retrieval up to date snapshot of miner's asking prices for data storage.

The _Faults index_ provides history data about miners faults while proving their storage on-chain. 

Built on top of the previous indexes, a _Reputation_ module constructs a weighted-scoring system that allows to sort miners considering multiple on-chain and off-chain data, such as: compared price to the median of the market, low storage-fault history, power on network, and external sources (soon!).

###  ⚡ FFS
This module provides a multitiered file storage API built on Filecoin and IPFS. Storing data on IPFS and Filecoin is as easy as expressing your desired configuration for storing a Cid.

Want to know more about this Powergate module? Check out the [FFS design document](https://github.com/textileio/powergate/blob/master/ffs/Design.md) and our presentation and demo at the _IPFS Pinning Summit_:

[![Video](https://img.youtube.com/vi/aiOTSkz_6aY/0.jpg)](https://youtu.be/aiOTSkz_6aY)

### 💫 API + CLI

Powergate exposes an API built from the various modules through gRPC endpoints. 
You can explore our [`.proto` files](https://github.com/textileio/powergate/tree/master/proto/powergate) to generate your clients, or take advange of a ready-to-use Powergate Go and [JS client](https://github.com/textileio/js-powergate-client). 🙌

We have a CLI that supports most of Powergate features.

To build and install the CLI, run:
```bash
$ make install-pow
```
The binary will be placed automatically in `$GOPATH/bin`. You may have to set the Path variables using the below commands
```bash
$ export PATH=$PATH:$(go env GOPATH)/bin
$ export GOPATH=$(go env GOPATH)
```
You can then run `pow` in your terminal.

You can read the [generated CLI docs](https://github.com/textileio/powergate/blob/master/cli-docs/pow/pow.md) in this repo, or run `pow` with the `--help` flag to see the available commands:

```
$ pow --help
A client for storage and retreival of powergate data

Usage:
  pow [flags]
  pow [command]

Available Commands:
  admin        Provides admin commands
  config       Provides commands to interact with cid storage configs
  data         Provides commands to interact with general data APIs
  deals        Provides commands to view Filecoin deal information
  help         Help about any command
  id           Returns the user id
  storage-jobs Provides commands to query for storage jobs in various states
  version      Display version information for pow and the connected server
  wallet       Provides commands about filecoin wallets

Flags:
  -h, --help                   help for pow
      --serverAddress string   address of the powergate service api (default "127.0.0.1:5002")
  -t, --token string           user auth token
  -v, --version                display version information for pow and the connected server

Use "pow [command] --help" for more information about a command.
```

## Installation

Powergate installation involves running external dependencies, and wiring them correctly with Powergate.

### External dependencies
Powergate needs external dependencies in order to provide full functionality, in particular a synced Filecoin client and a IPFS node.

#### Filecoin client
Currently, we support the Lotus Filecoin client but we plan to support other clients.

All described modules of Powergate need to comunicate with Lotus to build indices data, and provide storing and retrieving features in FFS. To install Lotus refer to its [official](https://lotu.sh/) documentation, taking special attention to [its dependencies](https://docs.lotu.sh/en+install-lotus-ubuntu). 

Fully syncing a Lotus node can take time, so be sure to check you're fully synced doing `./lotus sync status`.

We also automatically generate a public Docker image targeting the `master` branch of Lotus. This image is a pristine version of Lotus, with a sidecar reverse proxy to provide external access to the containerized API. For more information, refer to [textileio/lotus-build](https://github.com/textileio/lotus-build) and its [Dockerhub repository](https://hub.docker.com/repository/docker/textile/lotus).

In short, a fully-synced Lotus node should be available with its API (`127.0.0.1:1234`, by default) port accessible to Powergate.

### IPFS node
A running IPFS node is needed if you plan to use the FFS module.

If that's the case, you can refer [here](https://docs.ipfs.io/guides/guides/install/) for installation instructions, or its [Dockerhub repository](https://hub.docker.com/r/ipfs/go-ipfs) if you want to run a contanerized version. Currently we're supporting v0.5.1. The API endpoint should be accessible to Powergate (port 5001, by default). 

Since FFS _HotStorage_ is pinning Cids in the IPFS node, Powergate should be the only party controlling the pinset of the node. Other systems can share the same IPFS node if can  **guarantee** not unpinning Cids pinned by Powergate FFS instances. 

### Geolite database
Powergate needs an offline geo-location database to resolve miners country using their IP address. The same folder in which `powd` is executing, should have the Geolite2 database file `GeoLite2-City.mmdb` or you can pass the `--maxminddbfolder` flag to `powd` to specify the path of the folder containing `GeoLite2-City.mmdb`.
You can copy this file from the GitHub repo at `iplocation/maxmind/GeoLite2-City.mmdb`. If you run Powergate using Docker, this database is bundeled in the image so isn't necessary to have extra considerations.

### Server
To build and install the Powergate server, run:
```bash
make install-powd
```
You can run the `-h` flag to see the configurable flags:
```bash
$ powd -h 
Usage of powd:
      --askindexmaxparallel string       Max parallel query ask to execute while updating index (default "3")
      --askindexqueryasktimeout string   Timeout in seconds for a query ask (default "15")
      --askindexrefreshinterval string   Refresh interval measured in minutes (default "60")
      --askindexrefreshonstart           If true it will refresh the index on start
      --autocreatemasteraddr             Automatically creates & funds a master address if none is provided.
      --dealwatchpollduration string     Poll interval in seconds used by Deals Module watch to detect state changes (default "900")
      --debug                            Enable debug log level in all loggers.
      --devnet                           Indicate that will be running on an ephemeral devnet. --repopath will be autocleaned on exit.
      --disableindices                   Disable all indices updates, useful to help Lotus syncing process
      --disablenoncompliantapis          Disable APIs that may not easily comply with US law
      --ffsadmintoken string             FFS admin token for authorized APIs. If empty, the APIs will be open to the public.
      --ffsdealfinalitytimeout string    Deadline in minutes in which a deal must prove liveness changing status before considered abandoned (default "4320")
      --ffsminerselector string          Miner selector to be used by FFS: 'sr2', 'reputation' (default "sr2")
      --ffsminerselectorparams string    Miner selector configuration parameter, depends on --ffsminerselector (default "https://raw.githubusercontent.com/filecoin-project/slingshot/master/miners.json")
      --ffsminimumpiecesize string       Minimum piece size in bytes allowed to be stored in Filecoin (default "67108864")
      --ffsschedmaxparallel string       Maximum amount of Jobs executed in parallel (default "1000")
      --ffsusemasteraddr                 Use the master address as the initial address for all new FFS instances instead of creating a new unique addess for each new FFS instance.
      --gatewaybasepath string           Gateway base path. (default "/")
      --gatewayhostaddr string           Gateway host listening address. (default "0.0.0.0:7000")
      --grpchostaddr string              gRPC host listening address. (default "/ip4/0.0.0.0/tcp/5002")
      --grpcwebproxyaddr string          gRPC webproxy listening address. (default "0.0.0.0:6002")
      --ipfsapiaddr string               IPFS API endpoint multiaddress. (Optional, only needed if FFS is used) (default "/ip4/127.0.0.1/tcp/5001")
      --lotushost string                 Lotus client API endpoint multiaddress. (default "/ip4/127.0.0.1/tcp/1234")
      --lotusmasteraddr string           Existing wallet address in Lotus to be used as source of funding for new FFS instances. (Optional)
      --lotustoken string                Lotus API authorization token. This flag or --lotustoken file are mandatory.
      --lotustokenfile string            Path of a file that contains the Lotus API authorization token.
      --maxminddbfolder string           Path of the folder containing GeoLite2-City.mmdb (default ".")
      --mongodb string                   Mongo database name. (if --mongouri is used, is mandatory
      --mongouri string                  Mongo URI to connect to MongoDB database. (Optional: if empty, will use Badger)
      --repopath string                  Path of the repository where Powergate state will be saved. (default "~/.powergate")
      --walletinitialfund int            FFS initial funding transaction amount in attoFIL received by --lotusmasteraddr. (if set) (default 250000000000000000)
```

## Localnet mode

Having a fully synced Lotus node can take a considerable amount of time and effort to mantain. We have built [lotus-devnet](https://github.com/textileio/lotus-devnet) which runs a local network with a _sectorbuilder_ mock. This provides a fast way to spinup a local network where the sealing process if mocked, but the rest of the node logic is the same as production The _localnet_ supports both 2Kib and 512Kib sectors, and the speed of block production is configurable. Refer to [lotus-devnet](https://github.com/textileio/lotus-devnet) readme for more information.

If you're interested in running Powergate and experiment with the CLI, the fastest way is to replace the Lotus client dependency with a running localnet, which runs a local Lotus client connected to a network with local miners. 

A simple docker-compose setup is available that will run Powergate connected to a Lotus local network with 512Mib sectors and allows to use the gRPC API or CLI without any extra config flags! 🎊  Note: you will first need to [install Docker compose](https://docs.docker.com/compose/install/) in order to get started.
```bash
cd docker
make localnet
```
This will build Powergate `powd`, a Lotus local network with `BIGSECTORS=true` by default, an IPFS node and wire them correctly to be ready to use.

**Note**: Running `BIGSECTORS=false make localnet` will create the Lotus devent using 2Kib sectors. This may be more appropriate for certain development or testing scenarios. 

Here is a full example of using the local network:
Terminal 1:
```bash
cd docker
make localnet
```


Wait for seeing logs about the height of the chain increase in a regular cadence.

Terminal 2 (in the top-level repo directory):
```bash
make build
❯ head -c 700 </dev/urandom > myfile
❯ pow admin user create
{
  "user":  {
    "id":  "c06382e0-2021-4234-be53-6e07a8d40065",
    "token":  "883f57b1-4e66-47f8-b291-7cf8b10f6370"
  }
}
❯ pow data stage -t 883f57b1-4e66-47f8-b291-7cf8b10f6370 myfile
{
  "cid":  "QmQJxVtp61Y7UrdjUKuWvse3TxGHaPDyA7RobrBhFwqcBM"
}
❯ pow config apply -w -t 883f57b1-4e66-47f8-b291-7cf8b10f6370 QmYaAK8SSsKJsJdtahCbUe7MZzQdkPBybFCcQJJ3dKZpfm
{
  "jobId":  "b4110048-5367-4ae5-8508-709bf7969748"
}
                 JOB ID                |       STATUS       | MINER  |  PRICE   |    DEAL STATUS     
---------------------------------------+--------------------+--------+----------+--------------------
  b4110048-5367-4ae5-8508-709bf7969748 | JOB_STATUS_SUCCESS |        |          |                    
                                       |                    | f01000 | 62500000 | StorageDealActive
❯ pow data get -t 883f57b1-4e66-47f8-b291-7cf8b10f6370 QmYaAK8SSsKJsJdtahCbUe7MZzQdkPBybFCcQJJ3dKZpfm myfile2
> Success! Data written to myfile2
```

In this example we created a random 700 bytes file for the test, but since the localnet supports 512Mib sectors you can store store bigger files.

## Production setup

A production setup is also provided in the `docker` folder. It launches `powd` connected to `lotus` and `ipfs`, plus a set of monitoring components:
- _Prometheus_, which is the backend for metrics processing.
- _Grafana_, for metrics dashboard.
- _cAdvisor_, for container metrics.
- _Lotus_, node running on the current mainnet.
- _IPFS_, node running to back Powergate FFS.
- _Powergate_, wired with all of above components.

Depending on which network you want to connect to, you have to run different commands:
- `make up`, to connect to `mainnet`.

Remember that you should wait for _Lotus_ to be fully-synced which might take a long time; you can check your current node sync status running `lotus sync status` inside the Lotus container. We also provide automatically generated Dockerhub images of Powergate server, see [textile/powergate](https://hub.docker.com/r/textile/powergate).

If you're interested in a more detailed explanation about Powergate installation, please refer to the [installation docs](docs/manual_installation.md).

## Tests
We have a big set of tests for covering most important Powergate features.

For integration tests, we leverage our `textileio/lotus-devnet` configured with 2Kib sectors to provide fast iteration and CI runs.

If you want to run tests locally:
```bash
make test
```
It will auto-download any necessary dependencies and run all tests.

## Benchmark
There's a dedicated binary to run benchmarks against a Powergate server. For more information see the [specific README](cmd/powbench/README.md). 

Soon we'll add benchmark results against real miners in mainnet, so stay tuned. ⌛ 

## Contributing

This project is a work in progress. As such, there's a few things you can do right now to help out:

-   **Ask questions**! We'll try to help. Be sure to drop a note (on the above issue) if there is anything you'd like to work on and we'll update the issue to let others know. Also [get in touch](https://slack.textile.io) on Slack.
-   **Open issues**, [file issues](https://github.com/textileio/powergate/issues), submit pull requests!
-   **Perform code reviews**. More eyes will help a) speed the project along b) ensure quality and c) reduce possible future bugs.
-   **Take a look at the code**. Contributions here that would be most helpful are **top-level comments** about how it should look based on your understanding. Again, the more eyes the better.
-   **Add tests**. There can never be enough tests.

Before you get started, be sure to read our [contributors guide](./CONTRIBUTING.md) and our [contributor covenant code of conduct](./CODE_OF_CONDUCT.md).

## Changelog

[Changelog is published to Releases.](https://github.com/textileio/powergate/releases)

## License

[MIT](LICENSE)
