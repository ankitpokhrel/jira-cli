package jira

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"
)

const (
	// RFC3339 is jira datetime format.
	RFC3339 = "2006-01-02T15:04:05-0700"
	// RFC3339MilliLayout is jira datetime format with milliseconds.
	RFC3339MilliLayout = "2006-01-02T15:04:05.000-0700"

	// InstallationTypeCloud represents Jira cloud server.
	InstallationTypeCloud = "Cloud"
	// InstallationTypeLocal represents on-premise Jira servers.
	InstallationTypeLocal = "Local"

	baseURLv3 = "/rest/api/3"
	baseURLv2 = "/rest/api/2"
	baseURLv1 = "/rest/agile/1.0"

	apiVersion2 = "v2"
	apiVersion3 = "v3"
)

var (
	// ErrNoResult denotes no results.
	ErrNoResult = fmt.Errorf("jira: no result")
	// ErrEmptyResponse denotes empty response from the server.
	ErrEmptyResponse = fmt.Errorf("jira: empty response from server")
)

// ErrUnexpectedResponse denotes response code other than the expected one.
type ErrUnexpectedResponse struct {
	Body       Errors
	Status     string
	StatusCode int
}

func (e *ErrUnexpectedResponse) Error() string {
	return e.Body.String()
}

// ErrMultipleFailed represents a grouped error, usually when
// multiple request fails when running them in a loop.
type ErrMultipleFailed struct {
	Msg string
}

func (e *ErrMultipleFailed) Error() string {
	return e.Msg
}

// Errors is a jira error type.
type Errors struct {
	Errors          map[string]string
	ErrorMessages   []string
	WarningMessages []string
}

func (e Errors) String() string {
	var out strings.Builder

	if len(e.ErrorMessages) > 0 || len(e.Errors) > 0 {
		out.WriteString("\nError:\n")
		for _, v := range e.ErrorMessages {
			out.WriteString(fmt.Sprintf("  - %s\n", v))
		}
		for k, v := range e.Errors {
			out.WriteString(fmt.Sprintf("  - %s: %s\n", k, v))
		}
	}

	if len(e.WarningMessages) > 0 {
		out.WriteString("\nWarning:\n")
		for _, v := range e.WarningMessages {
			out.WriteString(fmt.Sprintf("  - %s\n", v))
		}
	}

	return out.String()
}

// Header is a key, value pair for request headers.
type Header map[string]string

// MTLSConfig is MTLS authtype specific config.
type MTLSConfig struct {
	CaCert     string
	ClientCert string
	ClientKey  string
}

// Config is a jira config.
type Config struct {
	Server     string
	Login      string
	APIToken   string
	AuthType   *AuthType
	Insecure   *bool
	Debug      bool
	MTLSConfig MTLSConfig
}

// Client is a jira client.
type Client struct {
	transport http.RoundTripper
	insecure  bool
	server    string
	login     string
	authType  *AuthType
	token     string
	timeout   time.Duration
	debug     bool
}

// ClientFunc decorates option for client.
type ClientFunc func(*Client)

// NewClient instantiates new jira client.
func NewClient(c Config, opts ...ClientFunc) *Client {
	client := Client{
		server:   strings.TrimSuffix(c.Server, "/"),
		login:    c.Login,
		token:    c.APIToken,
		authType: c.AuthType,
		debug:    c.Debug,
	}

	for _, opt := range opts {
		opt(&client)
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: client.insecure,
		},
		DialContext: (&net.Dialer{
			Timeout: client.timeout,
		}).DialContext,
	}

	if c.AuthType != nil && *c.AuthType == AuthTypeMTLS {
		// Create a CA certificate pool and add cert.pem to it.
		caCert, err := os.ReadFile(c.MTLSConfig.CaCert)
		if err != nil {
			log.Fatalf("%s, %s", err, c.MTLSConfig.CaCert)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		// Read the key pair to create the certificate.
		cert, err := tls.LoadX509KeyPair(c.MTLSConfig.ClientCert, c.MTLSConfig.ClientKey)
		if err != nil {
			log.Fatal(err)
		}

		// Add the MTLS specific configuration.
		transport.TLSClientConfig.RootCAs = caCertPool
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
		transport.TLSClientConfig.Renegotiation = tls.RenegotiateFreelyAsClient
	}

	client.transport = transport

	return &client
}

// WithTimeout is a functional opt to attach timeout to the client.
func WithTimeout(to time.Duration) ClientFunc {
	return func(c *Client) {
		c.timeout = to
	}
}

// WithInsecureTLS is a functional opt that allow you to skip TLS certificate verification.
func WithInsecureTLS(ins bool) ClientFunc {
	return func(c *Client) {
		c.insecure = ins
	}
}

// Get sends GET request to v3 version of the jira api.
func (c *Client) Get(ctx context.Context, path string, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodGet, c.server+baseURLv3+path, nil, headers)
}

// GetV2 sends GET request to v2 version of the jira api.
func (c *Client) GetV2(ctx context.Context, path string, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodGet, c.server+baseURLv2+path, nil, headers)
}

// GetV1 sends get request to v1 version of the jira api.
func (c *Client) GetV1(ctx context.Context, path string, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodGet, c.server+baseURLv1+path, nil, headers)
}

// Post sends POST request to v3 version of the jira api.
func (c *Client) Post(ctx context.Context, path string, body []byte, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodPost, c.server+baseURLv3+path, body, headers)
}

// PostV2 sends POST request to v2 version of the jira api.
func (c *Client) PostV2(ctx context.Context, path string, body []byte, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodPost, c.server+baseURLv2+path, body, headers)
}

// PostV1 sends POST request to v1 version of the jira api.
func (c *Client) PostV1(ctx context.Context, path string, body []byte, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodPost, c.server+baseURLv1+path, body, headers)
}

// Put sends PUT request to v3 version of the jira api.
func (c *Client) Put(ctx context.Context, path string, body []byte, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodPut, c.server+baseURLv3+path, body, headers)
}

// PutV2 sends PUT request to v2 version of the jira api.
func (c *Client) PutV2(ctx context.Context, path string, body []byte, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodPut, c.server+baseURLv2+path, body, headers)
}

// PutV1 sends PUT request to v1 version of the jira api.
func (c *Client) PutV1(ctx context.Context, path string, body []byte, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodPut, c.server+baseURLv1+path, body, headers)
}

// DeleteV2 sends DELETE request to v2 version of the jira api.
func (c *Client) DeleteV2(ctx context.Context, path string, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodDelete, c.server+baseURLv2+path, nil, headers)
}

func (c *Client) request(ctx context.Context, method, endpoint string, body []byte, headers Header) (*http.Response, error) {
	var (
		req *http.Request
		res *http.Response
		err error
	)

	req, err = http.NewRequest(method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	defer func() {
		if c.debug {
			dump(req, res)
		}
	}()

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Set default auth type to `basic`.
	if c.authType == nil {
		basic := AuthTypeBasic
		c.authType = &basic
	}

	// When need to compare using `String()` here, it is used to handle cases where the
	// authentication type might be empty, ensuring it defaults to the appropriate value.
	switch c.authType.String() {
	case string(AuthTypeMTLS):
		if c.token != "" {
			req.Header.Add("Authorization", "Bearer "+c.token)
		}
	case string(AuthTypeBearer):
		req.Header.Add("Authorization", "Bearer "+c.token)
	case string(AuthTypeBasic):
		req.SetBasicAuth(c.login, c.token)
	}

	httpClient := &http.Client{Transport: c.transport}

	return httpClient.Do(req.WithContext(ctx))
}

func dump(req *http.Request, res *http.Response) {
	reqDump, _ := httputil.DumpRequest(req, true)
	prettyPrintDump("Request Details", reqDump)

	if res != nil {
		respDump, _ := httputil.DumpResponse(res, false)
		prettyPrintDump("Response Details", respDump)
	}
}

func prettyPrintDump(heading string, data []byte) {
	const separatorWidth = 60

	fmt.Printf("\n\n%s", strings.ToUpper(heading))
	fmt.Printf("\n%s\n\n", strings.Repeat("-", separatorWidth))
	fmt.Print(string(data))
}

func formatUnexpectedResponse(res *http.Response) *ErrUnexpectedResponse {
	var b Errors

	// We don't care about decoding error here.
	_ = json.NewDecoder(res.Body).Decode(&b)

	return &ErrUnexpectedResponse{
		Body:       b,
		Status:     res.Status,
		StatusCode: res.StatusCode,
	}
}
