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

const (
	_testdataPathIssue   = "./testdata/issue.json"
	_testdataPathIssueV2 = "./testdata/issue-2.json"
)

func TestGetIssue(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/TEST-1", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			resp, err := os.ReadFile(_testdataPathIssue)
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
			resp, err := os.ReadFile(_testdataPathIssueV2)
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

func TestGetIssueRaw(t *testing.T) {
	cases := []struct {
		title              string
		givePayloadFile    string
		giveClientCallFunc func(c *Client) (string, error)
		wantReqURL         string
		wantOut            string
	}{
		{
			title:           "v3",
			givePayloadFile: _testdataPathIssue,
			giveClientCallFunc: func(c *Client) (string, error) {
				return c.GetIssueRaw("KAN-1")
			},
			wantReqURL: "/rest/api/3/issue/KAN-1",
			wantOut: `{
  "key": "TEST-1",
  "fields": {
    "issuetype": {
      "name": "Bug"
    },
    "resolution": null,
    "created": "2020-12-03T14:05:20.974+0100",
    "priority": {
      "name": "Medium"
    },
    "labels": [],
    "assignee": null,
    "updated": "2020-12-03T14:05:20.974+0100",
    "status": {
      "name": "To Do"
    },
    "summary": "Bug summary",
    "description": {
      "version": 1,
      "type": "doc",
      "content": [
        {
          "type": "paragraph",
          "content": [
            {
              "type": "text",
              "text": "Test description"
            }
          ]
        }
      ]
    },
    "issuelinks": [
      {
        "id": "10001",
        "outwardIssue": {
          "key": "TEST-2"
        }
      },
      {
        "id": "10002",
        "outwardIssue": {}
      }
    ],
    "reporter": {
      "displayName": "Person A"
    },
    "watches": {
      "watchCount": 1,
      "isWatching": true
    }
  }
}
`,
		},
		{
			title:           "v2",
			givePayloadFile: _testdataPathIssueV2,
			giveClientCallFunc: func(c *Client) (string, error) {
				return c.GetIssueV2Raw("KAN-1")
			},
			wantReqURL: "/rest/api/2/issue/KAN-1",
			wantOut: `{
  "key": "TEST-1",
  "fields": {
    "issuetype": {
      "name": "Bug"
    },
    "resolution": null,
    "created": "2020-12-03T14:05:20.974+0100",
    "priority": {
      "name": "Medium"
    },
    "labels": [],
    "assignee": null,
    "updated": "2020-12-03T14:05:20.974+0100",
    "status": {
      "name": "To Do"
    },
    "summary": "Bug summary",
    "description": "Test description",
    "reporter": {
      "displayName": "Person A"
    },
    "watches": {
      "watchCount": 1,
      "isWatching": true
    }
  }
}
`,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, c.wantReqURL, r.URL.Path)

				respContent, err := os.ReadFile(c.givePayloadFile)
				if !assert.NoError(t, err) {
					return
				}

				w.Header().Set("Content-Type", "application/json")
				_, err = w.Write(respContent)
				if !assert.NoError(t, err) {
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
			out, err := c.giveClientCallFunc(client)
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, c.wantOut, out)
		})
	}
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

		resp, err := os.ReadFile(_testdataPathIssue)
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

		expectedBody := `{"body":"comment","properties":[{"key":"sd.public.comment","value":{"internal":false}}]}`

		assert.Equal(t, expectedBody, actualBody.String())

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(201)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.AddIssueComment("TEST-1", "comment", false)
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.AddIssueComment("TEST-1", "comment", false)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestAddIssueWorklog(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/rest/api/2/issue/TEST-1/worklog", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var (
			expectedBody, expectedQuery string
			actualBody                  = new(strings.Builder)
		)

		_, _ = io.Copy(actualBody, r.Body)

		if strings.Contains(actualBody.String(), "started") {
			expectedBody = `{"started":"2022-01-01T01:02:02.000+0200","timeSpent":"1h","comment":"comment"}`
		} else {
			expectedBody = `{"timeSpent":"1h","comment":"comment"}`
		}

		assert.Equal(t, expectedBody, actualBody.String())

		if r.URL.RawQuery != "" {
			expectedQuery = `adjustEstimate=new&newEstimate=1d`
		}
		assert.Equal(t, expectedQuery, r.URL.RawQuery)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(201)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	err := client.AddIssueWorklog("TEST-1", "2022-01-01T01:02:02.000+0200", "1h", "comment", "")
	assert.NoError(t, err)

	err = client.AddIssueWorklog("TEST-1", "", "1h", "comment", "")
	assert.NoError(t, err)

	err = client.AddIssueWorklog("TEST-1", "", "1h", "comment", "1d")
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.AddIssueWorklog("TEST-1", "", "1h", "comment", "")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestGetField(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/field", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			resp, err := os.ReadFile("./testdata/fields.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.GetField()
	assert.NoError(t, err)

	expected := []*Field{
		{
			ID:     "fixVersions",
			Name:   "Fix Version/s",
			Custom: false,
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
				FieldID  int    `json:"customId,omitempty"`
			}{
				DataType: "array",
				Items:    "version",
			},
		},
		{
			ID:     "customfield_10111",
			Name:   "Original story points",
			Custom: true,
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
				FieldID  int    `json:"customId,omitempty"`
			}{
				DataType: "number",
				FieldID:  10111,
			},
		},
		{
			ID:     "timespent",
			Name:   "Time Spent",
			Custom: false,
			Schema: struct {
				DataType string `json:"type"`
				Items    string `json:"items,omitempty"`
				FieldID  int    `json:"customId,omitempty"`
			}{
				DataType: "number",
			},
		},
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.GetField()
	assert.NotNil(t, err)
}

func TestRemoteLinkIssue(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/TEST-1/remotelink", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
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

	err := client.RemoteLinkIssue("TEST-1", "weblink title", "http://weblink.com")
	assert.NoError(t, err)

	unexpectedStatusCode = true

	err = client.RemoteLinkIssue("TEST-1", "weblink title", "https://weblink.com")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestWatchIssue(t *testing.T) {
	var (
		apiVersion2          bool
		unexpectedStatusCode bool
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		if apiVersion2 {
			assert.Equal(t, "/rest/api/2/issue/TEST-1/watchers", r.URL.Path)
		} else {
			assert.Equal(t, "/rest/api/3/issue/TEST-1/watchers", r.URL.Path)
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

	err := client.WatchIssue("TEST-1", "a12b3")
	assert.NoError(t, err)

	apiVersion2 = true
	unexpectedStatusCode = true

	err = client.WatchIssueV2("TEST-1", "a12b3")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
