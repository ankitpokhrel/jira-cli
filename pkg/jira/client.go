package jira

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

const (
	// RFC3339 is jira datetime format.
	RFC3339 = "2006-01-02T15:04:05-0700"

	baseURLv3 = "/rest/api/3"
	baseURLv2 = "/rest/api/2"
	baseURLv1 = "/rest/agile/1.0"
)

var (
	// ErrNoResult denotes no results.
	ErrNoResult = fmt.Errorf("jira: no result")
	// ErrEmptyResponse denotes empty response from the server.
	ErrEmptyResponse = fmt.Errorf("jira: empty response from server")
	// ErrUnexpectedStatusCode denotes response code other than 200.
	ErrUnexpectedStatusCode = fmt.Errorf("jira: unexpected status code")
)

// Header is a key, value pair for request headers.
type Header map[string]string

// Config is a jira config.
type Config struct {
	Server   string
	Login    string
	APIToken string
	Debug    bool
}

// Client is a jira client.
type Client struct {
	transport http.RoundTripper
	server    string
	login     string
	token     string
	timeout   time.Duration
	debug     bool
}

// ClientFunc decorates option for client.
type ClientFunc func(*Client)

// NewClient instantiates new jira client.
func NewClient(c Config, opts ...ClientFunc) *Client {
	client := Client{
		server: strings.TrimSuffix(c.Server, "/"),
		login:  c.Login,
		token:  c.APIToken,
		debug:  c.Debug,
	}
	client.transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: client.timeout,
		}).DialContext,
	}

	for _, opt := range opts {
		opt(&client)
	}

	return &client
}

// WithTimeout is a functional opt to attach timeout to the client.
func WithTimeout(to time.Duration) ClientFunc {
	return func(c *Client) {
		c.timeout = to
	}
}

// Get sends GET request to v3 version of the jira api.
func (c *Client) Get(ctx context.Context, path string, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodGet, c.server+baseURLv3+path, nil, headers)
}

// GetV1 sends get request to v1 version of the jira api.
func (c *Client) GetV1(ctx context.Context, path string, headers Header) (*http.Response, error) {
	return c.request(ctx, http.MethodGet, c.server+baseURLv1+path, nil, headers)
}

// Post sends POST request to v3 version of the jira api.
func (c *Client) Post(ctx context.Context, path string, body []byte, headers Header) (*http.Response, error) {
	res, err := c.request(ctx, http.MethodPost, c.server+baseURLv3+path, body, headers)
	if err != nil {
		return res, err
	}
	return res, err
}

// PostV2 sends POST request to v2 version of the jira api.
func (c *Client) PostV2(ctx context.Context, path string, body []byte, headers Header) (*http.Response, error) {
	res, err := c.request(ctx, http.MethodPost, c.server+baseURLv2+path, body, headers)
	if err != nil {
		return res, err
	}
	return res, err
}

// PostV1 sends POST request to v1 version of the jira api.
func (c *Client) PostV1(ctx context.Context, path string, body []byte, headers Header) (*http.Response, error) {
	res, err := c.request(ctx, http.MethodPost, c.server+baseURLv1+path, body, headers)
	if err != nil {
		return res, err
	}
	return res, err
}

// Put sends PUT request to v3 version of the jira api.
func (c *Client) Put(ctx context.Context, path string, body []byte, headers Header) (*http.Response, error) {
	res, err := c.request(ctx, http.MethodPut, c.server+baseURLv3+path, body, headers)
	if err != nil {
		return res, err
	}
	return res, err
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

	req.SetBasicAuth(c.login, c.token)

	res, err = c.transport.RoundTrip(req.WithContext(ctx))

	return res, err
}

func dump(req *http.Request, res *http.Response) {
	reqDump, _ := httputil.DumpRequest(req, true)
	respDump, _ := httputil.DumpResponse(res, false)

	prettyPrintDump("Request Details", reqDump)
	prettyPrintDump("Response Details", respDump)
}

func prettyPrintDump(heading string, data []byte) {
	fmt.Printf("\n\n%s", strings.ToUpper(heading))
	fmt.Printf(fmt.Sprintf("\n%s\n\n", strings.Repeat("-", 60)))
	fmt.Print(string(data))
}
