package cmd

import (
	"crypto/tls"
	"crypto/x509"
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
	flags.BoolVar(&cmd.PrintHeaders, "print-headers", false, "Print out the response headers")
	flags.BoolVar(&cmd.Insecure, "insecure", false, "Skip TLS server verification")
	flags.StringVar(&cmd.ServerName, "server-name", "", "Override the server name used to verify the server's certificate")
	flags.StringVar(&cmd.RootCertificate, "root-cert", "", "Path to a PEM-encoded CA root certificate to trust")
	flags.StringVar(&cmd.ClientCertificate, "cert", "", "Path to a PEM-encoded client certificate to use")
	flags.StringVar(&cmd.ClientKey, "key", "", "Path to a PEM-encoded private key to use")

	fileFlags := []string{"root-cert", "cert", "key"}
	for _, ff := range fileFlags {
		err := cobraCmd.MarkFlagFilename(ff)
		if err != nil {
			// This fails if a flag name is passed that doesn't exist
			panic(fmt.Sprintf("Failed to mark flag %q as a filename", ff))
		}
	}
	cobraCmd.MarkFlagsRequiredTogether("cert", "key")

	return cobraCmd
}

type httpCommand struct {
	URL               string
	Method            string
	Headers           []string
	Timeout           time.Duration
	PrintHeaders      bool
	Insecure          bool
	ServerName        string
	RootCertificate   string
	ClientCertificate string
	ClientKey         string

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
	client, err := cmd.buildClient()
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}

	req, err := cmd.buildRequest()
	if err != nil {
		return fmt.Errorf("error building request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if cmd.PrintHeaders {
		err = resp.Header.Write(os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to write response headers: %w", err)
		}
	}

	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write response body: %w", err)
	}

	return nil
}

func (cmd *httpCommand) buildRequest() (*http.Request, error) {
	req, err := http.NewRequest(cmd.Method, cmd.URL, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header = cmd.headers
	return req, nil
}

func (cmd *httpCommand) buildClient() (*http.Client, error) {
	rootCertPool, err := cmd.getRootCertPool()
	if err != nil {
		return nil, fmt.Errorf("error getting root cert pool: %w", err)
	}

	certificates, err := cmd.getTLSCertificates()
	if err != nil {
		return nil, fmt.Errorf("error loading tls certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		ServerName:         cmd.ServerName,
		InsecureSkipVerify: cmd.Insecure,
		RootCAs:            rootCertPool,
		Certificates:       certificates,
	}

	client := &http.Client{
		Timeout: cmd.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return client, nil
}

func (cmd *httpCommand) getRootCertPool() (*x509.CertPool, error) {
	if cmd.RootCertificate == "" {
		return x509.SystemCertPool()
	}

	rootPEM, err := os.ReadFile(cmd.RootCertificate)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(rootPEM)
	if !ok {
		return nil, fmt.Errorf("no root certs were successfully parsed from %q", cmd.RootCertificate)
	}

	return pool, nil
}

func (cmd *httpCommand) getTLSCertificates() ([]tls.Certificate, error) {
	if cmd.ClientCertificate == "" || cmd.ClientKey == "" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(cmd.ClientCertificate, cmd.ClientKey)
	if err != nil {
		return nil, err
	}

	return []tls.Certificate{cert}, nil
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
