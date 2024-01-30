package main

import (
	"github.com/ethpandaops/tracoor/cmd"

	_ "github.com/lib/pq"
)

func main() {
	cmd.Execute()
}
