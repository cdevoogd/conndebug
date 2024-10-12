package cmd

import (
	"errors"
	"fmt"
	"net"

	"github.com/spf13/cobra"
)

func NewReachableCommand() *cobra.Command {
	cmd := &reachableCommand{}

	return &cobra.Command{
		Use:     "reachable ip:port",
		Short:   "Test if an address is reachable over TCP",
		Args:    cobra.ExactArgs(1),
		PreRunE: cmd.validate,
		RunE:    cmd.run,
	}
}

type reachableCommand struct {
	Address string
}

func (cmd *reachableCommand) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected 1 argument but received %d", len(args))
	}

	cmd.Address = args[0]
	err := cmd.validateAddress()
	if err != nil {
		return fmt.Errorf("the provided address is invalid: %w", err)
	}
	return nil
}

func (cmd *reachableCommand) run(*cobra.Command, []string) error {
	fmt.Printf("Connecting to: %s\n", cmd.Address)

	dialer := net.Dialer{}
	conn, err := dialer.Dial("tcp", cmd.Address)
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

func (cmd *reachableCommand) validateAddress() error {
	host, port, err := net.SplitHostPort(cmd.Address)
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
