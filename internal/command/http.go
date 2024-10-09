package command

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const headerDelimiter = ":"

type HTTP struct {
	URL     string        `arg:"" name:"url" help:"The URL to send a request to"`
	Method  string        `name:"method" short:"M" default:"GET" enum:"${http_methods}"`
	Headers []string      `name:"header" short:"H" help:"The header(s) to add to the request" placeholder:"'Header: Value'" sep:"none"`
	Timeout time.Duration `name:"timeout" short:"t" default:"0" help:"The max amount of time the request can take. A value of 0 means no timeout."`

	headers http.Header
}

func (cmd *HTTP) AfterApply() error {
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

func (cmd *HTTP) Run() error {
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

func (cmd *HTTP) buildRequest() (*http.Request, error) {
	req, err := http.NewRequest(cmd.Method, cmd.URL, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header = cmd.headers
	return req, nil
}
