package command

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/urfave/cli/v3"
)

var HTTPTrace = &cli.Command{
	Name:      "httptrace",
	Usage:     "trace an HTTP(S) connection using GET",
	Action:    runHTTPTrace,
	ArgsUsage: "url",
}

func runHTTPTrace(ctx context.Context, cmd *cli.Command) error {
	numArgs := cmd.Args().Len()
	if numArgs > 1 {
		return fmt.Errorf("expected a single argument, but received %d", numArgs)
	}

	if numArgs == 0 {
		return fmt.Errorf("a URL was not provided")
	}
	url := cmd.Args().Get(0)

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

	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
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
