package services

import (
	"strings"
)

type Client string

const (
	ClientUnknown    Client = "unknown"
	ClientGeth       Client = "geth"
	ClientNethermind Client = "nethermind"
	ClientBesu       Client = "besu"
	ClientErigon     Client = "erigon"
	ClientReth       Client = "reth"
	ClientEthereumJS Client = "ethereumjs"
)

var AllClients = []Client{
	ClientUnknown,
	ClientGeth,
	ClientNethermind,
	ClientBesu,
	ClientErigon,
	ClientReth,
	ClientEthereumJS,
}

func ClientFromString(client string) Client {
	asLower := strings.ToLower(client)

	if strings.Contains(asLower, "geth") {
		return ClientGeth
	}

	if strings.Contains(asLower, "nethermind") {
		return ClientNethermind
	}

	if strings.Contains(asLower, "besu") {
		return ClientBesu
	}

	if strings.Contains(asLower, "erigon") {
		return ClientErigon
	}

	if strings.Contains(asLower, "reth") {
		return ClientReth
	}

	if strings.Contains(asLower, "ethereumjs") {
		return ClientEthereumJS
	}

	return ClientUnknown
}
