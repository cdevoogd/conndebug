package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const headerDelimiter = ":"

func NewHTTPCommand() *cobra.Command {
	cmd := &httpCommand{}

	cobraCmd := &cobra.Command{
		Use:     "http url",
		Short:   "Send an HTTP request",
		Args:    cobra.ExactArgs(1),
		PreRunE: cmd.validate,
		RunE:    cmd.run,
	}

	flags := cobraCmd.Flags()
	flags.StringVarP(&cmd.Method, "method", "m", "GET", "The HTTP method to send the request with")
	flags.StringSliceVarP(&cmd.Headers, "header", "H", nil, "The header(s) to add to the request")
	flags.DurationVarP(&cmd.Timeout, "timeout", "t", 0, "The max amount of time the request can take. A value of 0 means no timeout.")

	return cobraCmd
}

type httpCommand struct {
	URL     string
	Method  string
	Headers []string
	Timeout time.Duration

	headers http.Header
}

func (cmd *httpCommand) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected 1 argument but recieved %d", len(args))
	}

	cmd.URL = args[0]
	err := cmd.validateURL()
	if err != nil {
		return err
	}

	err = cmd.validateMethod()
	if err != nil {
		return err
	}

	err = cmd.parseHeaders()
	if err != nil {
		return err
	}

	return nil
}

func (cmd *httpCommand) run(*cobra.Command, []string) error {
	req, err := cmd.buildRequest()
	if err != nil {
		return fmt.Errorf("error building request: %w", err)
	}

	client := &http.Client{Timeout: cmd.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}

func (cmd *httpCommand) buildRequest() (*http.Request, error) {
	req, err := http.NewRequest(cmd.Method, cmd.URL, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header = cmd.headers
	return req, nil
}

func (cmd *httpCommand) validateURL() error {
	parsedURL, err := url.Parse(cmd.URL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	switch parsedURL.Scheme {
	case "http", "https":
		break
	case "":
		return fmt.Errorf("the provided url does not include a scheme (http/https): %s", cmd.URL)
	default:
		return fmt.Errorf("the provided url does not include a supported scheme (http/https): %s", cmd.URL)
	}

	return nil
}

func (cmd *httpCommand) validateMethod() error {
	cmd.Method = strings.ToUpper(cmd.Method)
	switch cmd.Method {
	case http.MethodGet,
		http.MethodHead,
		http.MethodPut,
		http.MethodPost,
		http.MethodPatch,
		http.MethodDelete:
	default:
		return fmt.Errorf("unsupported HTTP method: %s", cmd.Method)
	}

	return nil
}

func (cmd *httpCommand) parseHeaders() error {
	cmd.headers = http.Header{}
	for _, header := range cmd.Headers {
		parts := strings.SplitN(header, headerDelimiter, 2)
		if len(parts) != 2 {
			return fmt.Errorf("header %q is malformed: missing %q delimiter", header, headerDelimiter)
		}
		cmd.headers.Add(parts[0], strings.TrimSpace(parts[1]))
	}
	return nil
}
