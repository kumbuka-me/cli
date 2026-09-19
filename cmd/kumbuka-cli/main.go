package main

import (
	"context"
	"os"

	"github.com/kumbuka-me/cli/internal/cli"
)

// Version is the build version injected by the release linker.
var Version = "dev"

// main executes the standalone Kumbuka CLI.
func main() {
	if err := cli.Run(context.Background(), os.Args[1:], Version, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
