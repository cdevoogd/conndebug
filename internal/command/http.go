package command

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/urfave/cli/v3"
)

var HTTP = &cli.Command{
	Name:      "http",
	Usage:     "send an HTTP request",
	Action:    runHTTP,
	ArgsUsage: "url",
}

func runHTTP(ctx context.Context, cmd *cli.Command) error {
	numArgs := cmd.Args().Len()
	if numArgs > 1 {
		return fmt.Errorf("expected a single argument, but received %d", numArgs)
	}

	if numArgs == 0 {
		return fmt.Errorf("a URL was not provided")
	}
	url := cmd.Args().Get(0)

	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("error building request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}
