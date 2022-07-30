package jira

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/adf"
)

func TestGetIssue(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			resp, err := os.ReadFile("./testdata/issue.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.GetIssue("TEST-1")
	assert.NoError(t, err)

	expected := &Issue{
		Key: "TEST-1",
		Fields: IssueFields{
			Summary: "Bug summary",
			Description: &adf.ADF{
				Version: 1,
				DocType: "doc",
				Content: []*adf.Node{
					{
						NodeType: "paragraph",
						Content: []*adf.Node{
							{NodeType: "text", NodeValue: adf.NodeValue{Text: "Test description"}},
						},
					},
				},
			},
			Labels:    []string{},
			IssueType: IssueType{Name: "Bug"},
			Priority: struct {
				Name string `json:"name"`
			}{Name: "Medium"},
			Reporter: struct {
				Name string `json:"displayName"`
			}{Name: "Person A"},
			Watches: struct {
				IsWatching bool `json:"isWatching"`
				WatchCount int  `json:"watchCount"`
			}{IsWatching: true, WatchCount: 1},
			Status: struct {
				Name string `json:"name"`
			}{Name: "To Do"},
			Created: "2020-12-03T14:05:20.974+0100",
			Updated: "2020-12-03T14:05:20.974+0100",
			IssueLinks: []struct {
				ID       string `json:"id"`
				LinkType struct {
					Name    string `json:"name"`
					Inward  string `json:"inward"`
					Outward string `json:"outward"`
				} `json:"type"`
				InwardIssue  *Issue `json:"inwardIssue,omitempty"`
				OutwardIssue *Issue `json:"outwardIssue,omitempty"`
			}{
				{
					ID:           "10001",
					OutwardIssue: &Issue{Key: "TEST-2"},
				},
				{
					ID:           "10002",
					OutwardIssue: &Issue{},
				},
			},
		},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.GetIssue("TEST-1")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestGetIssueWithoutDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1", r.URL.Path)

		resp, err := os.ReadFile("./testdata/issue-1.json")
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.GetIssue("TEST-1")
	assert.NoError(t, err)

	var nilADF *adf.ADF
	expected := &Issue{
		Key: "TEST-1",
		Fields: IssueFields{
			Summary:     "Bug summary",
			Description: nilADF,
			Labels:      []string{},
			IssueType:   IssueType{Name: "Bug"},
			Priority: struct {
				Name string `json:"name"`
			}{Name: "Medium"},
			Reporter: struct {
				Name string `json:"displayName"`
			}{Name: "Person A"},
			Watches: struct {
				IsWatching bool `json:"isWatching"`
				WatchCount int  `json:"watchCount"`
			}{IsWatching: true, WatchCount: 1},
			Status: struct {
				Name string `json:"name"`
			}{Name: "To Do"},
			Created: "2020-12-03T14:05:20.974+0100",
			Updated: "2020-12-03T14:05:20.974+0100",
		},
	}
	assert.Equal(t, expected, actual)
}

func TestGetIssueV2(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			resp, err := os.ReadFile("./testdata/issue-2.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.GetIssueV2("TEST-1")
	assert.NoError(t, err)

	expected := &Issue{
		Key: "TEST-1",
		Fields: IssueFields{
			Summary:     "Bug summary",
			Description: "Test description",
			Labels:      []string{},
			IssueType:   IssueType{Name: "Bug"},
			Priority: struct {
				Name string `json:"name"`
			}{Name: "Medium"},
			Reporter: struct {
				Name string `json:"displayName"`
			}{Name: "Person A"},
			Watches: struct {
				IsWatching bool `json:"isWatching"`
				WatchCount int  `json:"watchCount"`
			}{IsWatching: true, WatchCount: 1},
			Status: struct {
				Name string `json:"name"`
			}{Name: "To Do"},
			Created: "2020-12-03T14:05:20.974+0100",
			Updated: "2020-12-03T14:05:20.974+0100",
		},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.GetIssueV2("TEST-1")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestAssignIssue(t *testing.T) {
	var (
		apiVersion2          bool
		unexpectedStatusCode bool
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		if apiVersion2 {
			assert.Equal(t, "/rest/api/2/issue/TEST-1/assignee", r.URL.Path)
		} else {
			assert.Equal(t, "/rest/api/3/issue/TEST-1/assignee", r.URL.Path)
		}

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(204)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.AssignIssue("TEST-1", "a12b3")
	assert.NoError(t, err)

	err = client.AssignIssue("TEST-1", "none")
	assert.NoError(t, err)

	apiVersion2 = true
	unexpectedStatusCode = true

	err = client.AssignIssueV2("TEST-1", "default")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestGetIssueLinkTypes(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issueLinkType", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			resp, err := os.ReadFile("./testdata/issue-link-types.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.GetIssueLinkTypes()
	assert.NoError(t, err)

	expected := []*IssueLinkType{
		{
			ID:      "10000",
			Name:    "Blocks",
			Inward:  "is blocked by",
			Outward: "blocks",
		}, {
			ID:      "10001",
			Name:    "Cloners",
			Inward:  "is cloned by",
			Outward: "clones",
		}, {
			ID:      "10002",
			Name:    "Relates",
			Inward:  "relates to",
			Outward: "relates to",
		},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.GetIssueLinkTypes()
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestLinkIssue(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issueLink", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(201)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.LinkIssue("TEST-1", "TEST-2", "Blocks")
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.LinkIssue("TEST-1", "TEST-2", "invalid")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestUnlinkIssue(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issueLink/123", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(204)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.UnlinkIssue("123")
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.UnlinkIssue("123")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestGetLinkID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1", r.URL.Path)

		resp, err := os.ReadFile("./testdata/issue.json")
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(resp)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.GetLinkID("TEST-1", "TEST-2")
	assert.NoError(t, err)

	expected := "10001"
	assert.Equal(t, expected, actual)

	_, err = client.GetLinkID("TEST-1", "TEST-1234")
	assert.NotNil(t, err)
	assert.Equal(t, "no link found between provided issues", err.Error())
}

func TestAddIssueComment(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/rest/api/2/issue/TEST-1/comment", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		actualBody := new(strings.Builder)
		_, _ = io.Copy(actualBody, r.Body)

		expectedBody := `{"body":"comment"}`

		assert.Equal(t, expectedBody, actualBody.String())

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(201)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.AddIssueComment("TEST-1", "comment")
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.AddIssueComment("TEST-1", "comment")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
