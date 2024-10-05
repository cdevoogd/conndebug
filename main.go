package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/cdevoogd/conndebug/internal/command"
	"github.com/urfave/cli/v3"
)

func main() {
	root := &cli.Command{
		Usage: "small utilities for testing and debugging network connections",
		Commands: []*cli.Command{
			command.Reachable,
		},
	}

	if err := root.Run(context.Background(), os.Args); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
