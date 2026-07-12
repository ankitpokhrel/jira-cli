package jira

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
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

// ErrUnexpectedContentType denotes a successful HTTP response whose body
// isn't JSON. This usually means the request never reached the Jira API,
// for instance because it was intercepted by an SSO/login redirect, a VPN
// captive portal, or a reverse proxy, or because the configured host
// doesn't support the requested API version.
type ErrUnexpectedContentType struct {
	StatusCode  int
	ContentType string
	BodySnippet string
}

func (e *ErrUnexpectedContentType) Error() string {
	msg := fmt.Sprintf(
		"jira: unexpected non-JSON response (status %d, content-type %q); this usually means the request was "+
			"intercepted (e.g. by a login/SSO page, VPN captive portal, or proxy) or that the configured host "+
			"doesn't support the requested API version; verify the host and installation type with `jira init`",
		e.StatusCode, e.ContentType,
	)
	if e.BodySnippet != "" {
		msg += fmt.Sprintf("\nresponse body starts with: %s", e.BodySnippet)
	}
	return msg
}

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
			fmt.Fprintf(&out, "  - %s\n", v)
		}
		for k, v := range e.Errors {
			fmt.Fprintf(&out, "  - %s: %s\n", k, v)
		}
	}

	if len(e.WarningMessages) > 0 {
		out.WriteString("\nWarning:\n")
		for _, v := range e.WarningMessages {
			fmt.Fprintf(&out, "  - %s\n", v)
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
			MinVersion: tls.VersionTLS12,
			// Opt-in only, via --insecure, which `jira init` warns loudly about.
			InsecureSkipVerify: client.insecure, //nolint:gosec
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

	req, err = http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
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

	res, err = httpClient.Do(req)
	if err != nil {
		return res, err
	}

	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices &&
		res.ContentLength != 0 && !isJSONContentType(res) {
		ctErr := &ErrUnexpectedContentType{
			StatusCode:  res.StatusCode,
			ContentType: res.Header.Get("Content-Type"),
			BodySnippet: bodySnippet(res.Body),
		}
		_ = res.Body.Close()
		return nil, ctErr
	}

	return res, nil
}

// isJSONContentType reports whether the response's `Content-Type` header
// indicates a JSON payload. A missing header is treated as JSON since some
// endpoints (e.g. `204 No Content`) don't set one.
func isJSONContentType(res *http.Response) bool {
	ct := res.Header.Get("Content-Type")
	if ct == "" {
		return true
	}
	return strings.Contains(ct, "json")
}

// bodySnippet reads the leading bytes of a non-JSON body so the error can show
// what actually came back. `--debug` dumps headers only, so without this the
// user has no way to see the intercepting page that caused the failure.
func bodySnippet(body io.Reader) string {
	const maxLen = 200

	buf, err := io.ReadAll(io.LimitReader(body, maxLen))
	if err != nil {
		return ""
	}

	snippet := strings.Join(strings.Fields(string(buf)), " ")
	if len(snippet) == maxLen {
		snippet += "..."
	}
	return snippet
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
