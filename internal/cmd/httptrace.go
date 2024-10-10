package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	"github.com/spf13/cobra"
)

func NewHTTPTraceCommand() *cobra.Command {
	cmd := &httpTraceCommand{}

	return &cobra.Command{
		Use:     "httptrace url",
		Short:   "Trace an HTTP GET request",
		Args:    cobra.ExactArgs(1),
		PreRunE: cmd.validate,
		RunE:    cmd.run,
	}
}

type httpTraceCommand struct {
	URL string
}

func (cmd *httpTraceCommand) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected 1 argument but recieved %d", len(args))
	}

	cmd.URL = args[0]
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

func (cmd *httpTraceCommand) run(*cobra.Command, []string) error {
	start := time.Now()
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			logWithDelta(start, "DNS start - host=%q", info.Host)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			logWithDelta(start, "DNS done - addrs=%v", info.Addrs)
		},
		ConnectStart: func(network, addr string) {
			logWithDelta(start, "Connection starting - network=%q, addr=%q", network, addr)
		},
		ConnectDone: func(network string, addr string, err error) {
			if err != nil {
				logWithDelta(start, "Connection failed - network=%q, addr=%q, err=%q", network, addr, err)
				return
			}
			logWithDelta(start, "Connection done - network=%q, addr=%q", network, addr)
		},
		TLSHandshakeStart: func() {
			logWithDelta(start, "TLS handshake starting")
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			if err != nil {
				logWithDelta(start, "TLS handshake failed - err=%q", err)
				return
			}
			logWithDelta(start, "TLS handshake complete")
		},
		GotConn: func(httptrace.GotConnInfo) {
			logWithDelta(start, "Got connection")
		},
		GotFirstResponseByte: func() {
			logWithDelta(start, "Got first response byte")
		},
	}

	req, err := http.NewRequest(http.MethodGet, cmd.URL, http.NoBody)
	if err != nil {
		return fmt.Errorf("error building request: %w", err)
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func logWithDelta(start time.Time, format string, args ...any) {
	delta := time.Since(start)
	fmt.Printf("%6dms: ", delta.Milliseconds())
	fmt.Printf(format, args...)
	fmt.Println()
}
