package jira

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	errEmptyResponse        = fmt.Errorf("jira: empty response from server")
	errUnexpectedStatusCode = fmt.Errorf("jira: unexpected status code")
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
	baseURL   string
	login     string
	token     string
	timeout   time.Duration
}

// ClientFunc decorates option for client.
type ClientFunc func(*Client)

// NewClient instantiates new jira client.
func NewClient(c Config, opts ...ClientFunc) *Client {
	client := Client{
		server:  strings.TrimSuffix(c.Server, "/"),
		baseURL: "/rest/api/2",
		login:   c.Login,
		token:   c.APIToken,
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

func (c *Client) endpoint(path string) string {
	return c.server + c.baseURL + path
}

// Get sends get request to the jira server.
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.endpoint(path), nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.login, c.token)

	res, err := c.transport.RoundTrip(req.WithContext(ctx))

	return res, err
}
