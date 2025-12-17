package main

import (
	"os"

	"github.com/buker/revi/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
