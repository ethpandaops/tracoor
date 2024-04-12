<img align="left" src="./web/src/assets//logo.png" width="88">
<h1>Tracoor</h1>

Tracoor captures, stores and makes available beacon states, beacon blocks, execution debug traces, execution bad blocks and invalid gossiped verified blocks.

----------

<p align="center">
  <b> Live Versions </b>
</p>
<p align="center">
  <a href="https://tracoor.mainnet.ethpandaops.io" target="_blank">Mainnet</a>
</p>
<p align="center">
  <a href="https://tracoor.holesky.ethpandaops.io" target="_blank">Holesky</a>
</p>
<p align="center">
  <a href="https://tracoor.sepolia.ethpandaops.io" target="_blank">Sepolia</a>
</p>

----------
## Contents

* [Features](#features)
- [Usage](#usage) 
  * [Server](#server)
  * [Agent](#agent)
- [Getting Started](#getting-started)
  * [Download a release](#download-a-release)
  * [Docker](#docker)
    + [Images](#images)
  * [Kubernetes via Helm](#kubernetes-via-helm)
  * [Building yourself](#building-yourself)
* [Contributing](#contributing)
  + [Running locally](#running-locally)
    - [Backend](#backend)
    - [Frontend](#frontend)
* [Contact](#contact)

----------


## Features

* [x] Web interface for viewing beacon states, execution block traces and invalid execution blocks
* [x] Configurable retention period
* [x] Prometheus metrics

### Capturing

* [x] Ethereum Beacon Node
* [x] Ethereum Execution Node

### Storing

* [x] S3

### Indexing

* [x] Postgres

## Usage

Tracoor has two components, a server and an agent. The server has a web interface and serves the data captured by the agent. The agent captures data from the Ethereum Beacon Node and Ethereum Execution Node.

### Server

Tracoor server requires a config file. An example file can be found [here](https://github.com/ethpandaops/tracoor/blob/master/example_server_config.yaml).

```bash
Runs tracoor in Server mode, which means it will listen to gRPC requests from
	tracoor agent nodes and forward the data on to the configured sinks.

Usage:
  tracoor server [flags]

Flags:
      --config string   config file (default is server.yaml) (default "server.yaml")
  -h, --help            help for server
```

### Agent

Tracoor agent requires a config file. An example file can be found [here](https://github.com/ethpandaops/tracoor/blob/master/example_agent_config.yaml).

```bash
Runs tracoor in agent mode, which means it will listen for events from
	an Ethereum beacon node and forward the data on to 	the configured sinks.

Usage:
  tracoor agent [flags]

Flags:
      --config string   config file (default is agent.yaml) (default "agent.yaml")
  -h, --help            help for agent
```

## Getting Started

### Download a release

Download the latest release from the [Releases page](https://github.com/ethpandaops/tracoor/releases). Extract and run with:

```bash
./tracoor --help
```

### Docker

Available as a docker image at [ethpandaops/tracoor](https://hub.docker.com/r/ethpandaops/tracoor/tags)
#### Images

- `latest` - distroless, multiarch
- `latest-debian` - debian, multiarch
- `$version` - distroless, multiarch, pinned to a release (i.e. `0.1.0`)
- `$version-debian` - debian, multiarch, pinned to a release (i.e. `0.1.0-debian`)

**Quick start**

```bash
docker run -d  --name tracoor -v $HOST_DIR_CHANGE_ME/config.yaml:/opt/tracoor/config.yaml -p 9090:9090 -p 5555:5555 -it ethpandaops/tracoor:latest server --config /opt/tracoor/config.yaml;
docker logs -f tracoor;
```

### Kubernetes via Helm

- [tracoor-server](https://github.com/skylenet/ethereum-helm-charts/tree/master/charts/tracoor-server)
- [tracoor-agent](https://github.com/skylenet/ethereum-helm-charts/tree/master/charts/tracoor-agent)

```bash
helm repo add ethereum-helm-charts https://ethpandaops.github.io/ethereum-helm-charts

# server
helm install tracoor ethereum-helm-charts/tracoor-server -f your_values.yaml

# agent
helm install tracoor ethereum-helm-charts/tracoor-agent -f your_values.yaml
```

### Building yourself

1. Clone the repo
   ```sh
   go get github.com/ethpandaops/tracoor
   ```
1. Change directories
   ```sh
   cd ./tracoor
   ```
1. Build the binary
   ```sh  
    go build -o tracoor .
   ```
1. Run the server
   ```sh  
    ./tracoor server --config example_server_config.yaml
   ```
1. Run the agent
   ```sh  
    ./tracoor agent --config example_agent_config.yaml
   ```

## Contributing

Contributions are greatly appreciated! Pull requests will be reviewed and merged promptly if you're interested in improving the tracoor!

1. Fork the project
1. Create your feature branch:
    - `git checkout -b feat/new-feature`
1. Commit your changes:
    - `git commit -m 'feat(profit): new feature`
1. Push to the branch:
    -`git push origin feat/new-feature`
1. Open a pull request

### Running locally
#### Server
```
go run main.go server --config example_server_config.yaml
```

#### Agent
```
go run main.go agent --config example_agent_config.yaml
```

#### Frontend

A frontend is provided in this project in [`./web`](https://github.com/ethpandaops/tracoor/blob/master/example_config.yaml) directory which needs to be built before it can be served by the server, eg. `http://localhost:5555`.

The frontend can be built with the following command;
```bash
# install node modules and build
make build-web
```

Building frontend requires `npm` and `NodeJS` to be installed.


## Contact

Sam - [@samcmau](https://twitter.com/samcmau)

Andrew - [@savid](https://twitter.com/Savid)
