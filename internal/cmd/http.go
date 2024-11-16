package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/smallstep/certinfo"
	"github.com/spf13/cobra"
)

const (
	headerDelimiter   = ":"
	cookieDelimiter   = "="
	contentTypeHeader = "Content-Type"
	// defaultContentType is the content type that will be set when the user included data for a
	// body, but has not manually set a content type header.
	defaultContentType = "text/plain"
)

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
	flags.StringSliceVarP(&cmd.Headers, "header", "H", nil, "The header(s) to add to the request (format: 'Header: value')")
	flags.StringSliceVarP(&cmd.Cookies, "cookie", "c", nil, "The cookie(s) to add to the request (format: 'key=value')")
	flags.DurationVarP(&cmd.Timeout, "timeout", "t", 0, "The max amount of time the request can take. A value of 0 means no timeout.")
	flags.StringVarP(&cmd.DataRaw, "data", "d", "", "Raw data that should be sent in the body of the request")
	flags.StringVar(&cmd.DataFile, "data-file", "", "The path to a file (or '-' for stdin) to use as the request body")
	flags.BoolVar(&cmd.Insecure, "insecure", false, "Skip TLS server verification")
	flags.StringVar(&cmd.ServerName, "server-name", "", "Override the server name used to verify the server's certificate")
	flags.StringVar(&cmd.RootCertificate, "root-cert", "", "Path to a PEM-encoded CA root certificate to trust")
	flags.StringVar(&cmd.ClientCertificate, "cert", "", "Path to a PEM-encoded client certificate to use")
	flags.StringVar(&cmd.ClientKey, "key", "", "Path to a PEM-encoded private key to use")
	flags.BoolVar(&cmd.PrintStatus, "print-status", false, "Print out the response status")
	flags.BoolVar(&cmd.PrintTLSState, "print-tls", false, "Print out the response TLS information")
	flags.BoolVar(&cmd.PrintShortCertificates, "short-certs", false, "When printing TLS info, print the short representation of the certificates")
	flags.BoolVar(&cmd.PrintHeaders, "print-headers", false, "Print out the response headers")
	flags.StringVarP(&cmd.OutputFile, "output", "o", "", "A file path to output the response body to")

	fileFlags := []string{"data-file", "root-cert", "cert", "key", "output"}
	for _, ff := range fileFlags {
		err := cobraCmd.MarkFlagFilename(ff)
		if err != nil {
			// This fails if a flag name is passed that doesn't exist
			panic(fmt.Sprintf("Failed to mark flag %q as a filename", ff))
		}
	}

	cobraCmd.MarkFlagsRequiredTogether("cert", "key")
	cobraCmd.MarkFlagsMutuallyExclusive("data", "data-file")

	return cobraCmd
}

type httpCommand struct {
	// Request Options
	URL      string
	Method   string
	Headers  []string
	Cookies  []string
	Timeout  time.Duration
	DataRaw  string
	DataFile string
	// TLS Options
	Insecure          bool
	ServerName        string
	RootCertificate   string
	ClientCertificate string
	ClientKey         string
	// Output Options
	PrintStatus            bool
	PrintTLSState          bool
	PrintShortCertificates bool
	PrintHeaders           bool
	OutputFile             string

	url       *url.URL
	headers   http.Header
	cookieJar *cookiejar.Jar
}

func (cmd *httpCommand) log(a ...any) {
	fmt.Fprintln(os.Stderr, a...)
}

func (cmd *httpCommand) logf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func (cmd *httpCommand) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected 1 argument but received %d", len(args))
	}
	cmd.URL = args[0]

	helpers := []func() error{
		cmd.parseURL,
		cmd.validateMethod,
		cmd.parseHeaders,
		cmd.parseCookies,
	}

	for _, helper := range helpers {
		err := helper()
		if err != nil {
			return err
		}
	}

	return nil
}

func (cmd *httpCommand) run(*cobra.Command, []string) error {
	client, err := cmd.buildClient()
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}

	body, err := cmd.openBody()
	if err != nil {
		return fmt.Errorf("error opening body: %w", err)
	}
	defer body.Close()

	req, err := cmd.buildRequest(body)
	if err != nil {
		return fmt.Errorf("error building request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if cmd.PrintTLSState {
		cmd.logTLSState(resp.TLS)
	}

	if cmd.PrintStatus {
		cmd.log(resp.Status)
	}

	if cmd.PrintHeaders {
		err = resp.Header.Write(os.Stderr)
		if err != nil {
			return fmt.Errorf("failed to write response headers: %w", err)
		}
	}

	err = cmd.writeOutput(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write response body: %w", err)
	}

	return nil
}

func (cmd *httpCommand) buildRequest(body io.ReadCloser) (*http.Request, error) {
	req, err := http.NewRequest(cmd.Method, cmd.url.String(), body)
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
		Jar:     cmd.cookieJar,
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

func (cmd *httpCommand) parseURL() error {
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

	cmd.url = parsedURL
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

	// If the request is going to have a body, but the user did not explicitly set a content type,
	// then include a default content type to prevent issues with servers that expect it. A more
	// accurate type can be included by the user using the --header/-H flag.
	if cmd.hasBody() && cmd.headers.Get(contentTypeHeader) == "" {
		cmd.headers.Set(contentTypeHeader, defaultContentType)
	}

	return nil
}

func (cmd *httpCommand) parseCookies() error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("error creating cookie jar: %w", err)
	}

	allCookies := []*http.Cookie{}
	for _, cookie := range cmd.Cookies {
		cookies, err := http.ParseCookie(cookie)
		if err != nil {
			return fmt.Errorf("failed to parse cookie input %q: %w", cookie, err)
		}
		allCookies = append(allCookies, cookies...)
	}

	jar.SetCookies(cmd.url, allCookies)
	cmd.cookieJar = jar
	return nil
}

func (cmd *httpCommand) hasBody() bool {
	return cmd.DataRaw != "" || cmd.DataFile != ""
}

func (cmd *httpCommand) openBody() (io.ReadCloser, error) {
	switch {
	case cmd.DataRaw != "":
		return io.NopCloser(strings.NewReader(cmd.DataRaw)), nil
	case cmd.DataFile == "-":
		return os.Stdin, nil
	case cmd.DataFile != "":
		return os.Open(cmd.DataFile)
	default:
		return http.NoBody, nil
	}
}

func (cmd *httpCommand) writeOutput(response io.Reader) error {
	if cmd.OutputFile == "" {
		_, err := io.Copy(os.Stdout, response)
		return err
	}

	f, err := os.Create(cmd.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, response)
	return err
}

func (cmd *httpCommand) logTLSState(state *tls.ConnectionState) {
	if state == nil {
		cmd.log("No TLS connection state is available. The request was likely unencrypted (HTTP).")
		return
	}

	getCertificateText := certinfo.CertificateText
	if cmd.PrintShortCertificates {
		getCertificateText = certinfo.CertificateShortText
	}

	cmd.log("TLS Version:", tls.VersionName(state.Version))
	cmd.log("Cipher Suite:", tls.CipherSuiteName(state.CipherSuite))
	cmd.log("Negotiated Protocol (ALPN):", state.NegotiatedProtocol)
	cmd.log("Server Name:", state.ServerName)
	for i, cert := range state.PeerCertificates {
		cmd.logf("Peer Certificate #%d:\n", i)
		text, err := getCertificateText(cert)
		if err != nil {
			cmd.logf("Failed to parse certificate: %s", err)
			continue
		}
		cmd.log(text)
	}
}
