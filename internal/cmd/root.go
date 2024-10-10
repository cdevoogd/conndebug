package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func Execute() {
	cmd := &cobra.Command{
		Use:   "conndebug",
		Short: "A small CLI for basic network connection debugging",
	}
	cmd.AddCommand(NewHTTPCommand())
	cmd.AddCommand(NewHTTPTraceCommand())
	cmd.AddCommand(NewReachableCommand())

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
