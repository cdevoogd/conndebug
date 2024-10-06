package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cdevoogd/conndebug/internal/command"
	"github.com/urfave/cli/v3"
)

func main() {
	root := &cli.Command{
		Usage: "small utilities for testing and debugging network connections",
		Commands: []*cli.Command{
			command.HTTPTrace,
			command.Reachable,
		},
	}

	if err := root.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
