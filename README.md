<img align="left" src="./web/src/assets//logo.png" width="88">
<h1>Tracoor</h1>


TODO

----------

<p align="center">
  <b> Live Versions </b>
</p>
<p align="center">
  <a href="https://tracoor.mainnet.ethpandaops.io" target="_blank">Mainnet</a>
</p>
<p align="center">
  <a href="https://tracoor.goerli.ethpandaops.io" target="_blank">Goerli</a>
</p>

----------
## Contents

* [Features](#features)
- [Usage](#usage) 
  * [Configuration](#configuration)
  * [Getting Started](#getting-started)
    + [Download a release](#download-a-release)
    + [Docker](#docker)
      - [Images](#images)
    + [Kubernetes via Helm](#kubernetes-via-helm)
    + [Building yourself](#building-yourself)
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

* [x] Sqlite
* [x] Postgres

## Usage

Tracoor requires a config file. An example file can be found [here](https://github.com/ethpandaops/tracoor/blob/master/example_config.yaml).

```bash
tracoor - fetches and serves Ethereum fork choice data

Usage:
  tracoor [flags]

Flags:
      --config string   config file (default is config.yaml) (default "config.yaml")
  -h, --help            help for tracoor
```

## Getting Started

### Download a release

Download the latest release from the [Releases page](https://github.com/ethpandaops/tracoor/releases). Extract and run with:

```bash
./tracoor --config your-config.yaml
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
docker run -d  --name tracoor -v $HOST_DIR_CHANGE_ME/config.yaml:/opt/tracoor/config.yaml -p 9090:9090 -p 5555:5555 -it ethpandaops/tracoor:latest --config /opt/tracoor/config.yaml;
docker logs -f tracoor;
```

### Kubernetes via Helm

[Read more](https://github.com/skylenet/ethereum-helm-charts/tree/master/charts/tracoor)

```bash
helm repo add ethereum-helm-charts https://ethpandaops.github.io/ethereum-helm-charts

helm install tracoor ethereum-helm-charts/tracoor -f your_values.yaml
```

### Building yourself

1. Clone the repo
   ```sh
   go get github.com/ethpandaops/tracoor
   ```
2. Change directories
   ```sh
   cd ./tracoor
   ```
3. Build the binary
   ```sh  
    go build -o tracoor .
   ```
4. Run the service
   ```sh  
    ./tracoor
   ```

## Contributing

Contributions are greatly appreciated! Pull requests will be reviewed and merged promptly if you're interested in improving the tracoor!

1. Fork the project
2. Create your feature branch:
    - `git checkout -b feat/new-feature`
3. Commit your changes:
    - `git commit -m 'feat(profit): new feature`
4. Push to the branch:
    -`git push origin feat/new-feature`
5. Open a pull request

### Running locally
#### Backend
```
go run main.go --config your_config.yaml
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
