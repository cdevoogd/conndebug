package command

import (
	"errors"
	"fmt"
	"net"
)

type Reachable struct {
	Address string `arg:"" name:"ip:port" help:"the address to connect to"`
}

func (cmd *Reachable) AfterApply() error {
	err := validateAddress(cmd.Address)
	if err != nil {
		return fmt.Errorf("the provided address is invalid: %w", err)
	}
	return nil
}

func (cmd *Reachable) Run() error {
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
