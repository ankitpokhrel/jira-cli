package jira

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	// RFC3339 is jira datetime format.
	RFC3339 = "2006-01-02T15:04:05-0700"

	baseURLv3 = "/rest/api/3"
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

// Config is a jira config.
type Config struct {
	Server   string
	Login    string
	APIToken string
}

// Client is a jira client.
type Client struct {
	transport http.RoundTripper
	server    string
	login     string
	token     string
	timeout   time.Duration
	v1        bool
}

// ClientFunc decorates option for client.
type ClientFunc func(*Client)

// NewClient instantiates new jira client.
func NewClient(c Config, opts ...ClientFunc) *Client {
	client := Client{
		server: strings.TrimSuffix(c.Server, "/"),
		login:  c.Login,
		token:  c.APIToken,
		v1:     false,
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

// Get sends get request to the jira server.
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.server+baseURLv3+path, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.login, c.token)

	res, err := c.transport.RoundTrip(req.WithContext(ctx))

	return res, err
}

// GetV1 sends get request to v1 version of the jira api.
func (c *Client) GetV1(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.server+baseURLv1+path, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.login, c.token)

	res, err := c.transport.RoundTrip(req.WithContext(ctx))

	return res, err
}
