package main

import (
	"context"
	"os"

	"github.com/kumbuka-me/cli/internal/cli"
)

var Version = "dev"

func main() {
	if err := cli.Run(context.Background(), os.Args[1:], Version, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
