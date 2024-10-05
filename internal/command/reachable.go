package command

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/urfave/cli/v3"
)

var Reachable = &cli.Command{
	Name:      "reachable",
	Usage:     "attempt to connect and immediately close a connection to test if an address is reachable",
	Action:    checkReachable,
	ArgsUsage: "ip:port",
}

func checkReachable(ctx context.Context, cmd *cli.Command) error {
	numArgs := cmd.Args().Len()
	if numArgs > 1 {
		return fmt.Errorf("expected a single argument, but received %d", numArgs)
	}

	if numArgs == 0 {
		return fmt.Errorf("an address was not provided")
	}

	rawAddress := cmd.Args().Get(0)
	err := validateAddress(rawAddress)
	if err != nil {
		return fmt.Errorf("the provided address is invalid: %w", err)
	}

	fmt.Printf("Connecting to: %s\n", rawAddress)

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", rawAddress)
	if err != nil {
		return fmt.Errorf("error dialing address: %w", err)
	}

	fmt.Println("Connection succeeded")

	err = conn.Close()
	if err != nil {
		return fmt.Errorf("error closing the connection: %w", err)
	}

	fmt.Println("Connection closed")

	return nil
}

func validateAddress(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if port == "" {
		return errors.New("missing port")
	}
	if host == "" {
		return errors.New("missing host")
	}
	return nil
}
