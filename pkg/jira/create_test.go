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
)

type createTestServer struct{ code int }

func (c *createTestServer) serve(t *testing.T, expectedBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		actualBody := new(strings.Builder)
		_, _ = io.Copy(actualBody, r.Body)

		assert.JSONEq(t, expectedBody, actualBody.String())

		if c.code == 201 {
			resp, err := os.ReadFile("./testdata/create.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(201)
			_, _ = w.Write(resp)
		} else {
			w.WriteHeader(c.code)
		}
	}))
}

func (c *createTestServer) statusCode(code int) {
	c.code = code
}

func TestCreate(t *testing.T) {
	expectedBody := `{"update":{},"fields":{"project":{"key":"TEST"},"issuetype":{"name":"Bug"},` +
		`"summary":"Test bug","description":"Test description","priority":{"name":"Normal"},"labels":["test","dev"],` +
		`"components":[{"name":"BE"},{"name":"FE"}],"fixVersions":[{"name":"v2.0"},{"name":"v2.1-hotfix"}],"versions":[{"name":"v3.0"},{"name":"v3.1-hotfix"}],` +
		`"timetracking":{"originalEstimate":"2d"}}}`
	testServer := createTestServer{code: 201}
	server := testServer.serve(t, expectedBody)
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	requestData := CreateRequest{
		Project:          "TEST",
		IssueType:        "Bug",
		Summary:          "Test bug",
		Body:             "Test description",
		Priority:         "Normal",
		Labels:           []string{"test", "dev"},
		Components:       []string{"BE", "FE"},
		FixVersions:      []string{"v2.0", "v2.1-hotfix"},
		AffectsVersions:  []string{"v3.0", "v3.1-hotfix"},
		OriginalEstimate: "2d",
	}
	actual, err := client.CreateV2(&requestData)
	assert.NoError(t, err)

	expected := &CreateResponse{
		ID:  "10057",
		Key: "TEST-3",
	}

	assert.Equal(t, expected, actual)

	testServer.statusCode(400)

	_, err = client.CreateV2(&requestData)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestCreateSubtask(t *testing.T) {
	expectedBody := `{"update":{},"fields":{"project":{"key":"TEST"},"issuetype":{"name":"Sub-task"},` +
		`"parent":{"key":"TEST-123"},"summary":"Test sub-task","description":"Test description"}}`
	testServer := createTestServer{code: 201}
	server := testServer.serve(t, expectedBody)
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	requestData := CreateRequest{
		Project:        "TEST",
		IssueType:      "Sub-task",
		Summary:        "Test sub-task",
		Body:           "Test description",
		ParentIssueKey: "TEST-123",
	}
	actual, err := client.CreateV2(&requestData)
	assert.NoError(t, err)

	expected := &CreateResponse{
		ID:  "10057",
		Key: "TEST-3",
	}

	assert.Equal(t, expected, actual)

	testServer.statusCode(500)

	_, err = client.CreateV2(&requestData)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestCreateEpic(t *testing.T) {
	expectedBody := `{"update":{},"fields":{"customfield_10001":"CLI","description":"Test description","issuetype":{"name":` +
		`"Bug"},"priority":{"name":"Normal"},"project":{"key":"TEST"},"summary":"Test bug"}}`
	testServer := createTestServer{code: 201}
	server := testServer.serve(t, expectedBody)
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
	requestData := CreateRequest{
		Project:   "TEST",
		IssueType: "Bug",
		Name:      "CLI",
		Summary:   "Test bug",
		Body:      "Test description",
		Priority:  "Normal",
		EpicField: "customfield_10001",
	}
	actual, err := client.CreateV2(&requestData)
	assert.NoError(t, err)

	expected := &CreateResponse{
		ID:  "10057",
		Key: "TEST-3",
	}
	assert.Equal(t, expected, actual)

	testServer.statusCode(400)

	_, err = client.CreateV2(&requestData)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestCreateEpicNextGen(t *testing.T) {
	expectedBody := `{"update":{},"fields":{"description":"Test description","issuetype":{"name":"Bug"},` +
		`"parent":{"key":"TEST-123"},"project":{"key":"TEST"},"summary":"Test bug"}}`
	testServer := createTestServer{code: 201}
	server := testServer.serve(t, expectedBody)
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))
	requestData := CreateRequest{
		Project:        "TEST",
		IssueType:      "Bug",
		Name:           "CLI",
		Summary:        "Test bug",
		Body:           "Test description",
		ParentIssueKey: "TEST-123",
	}
	requestData.ForProjectType(ProjectTypeNextGen)

	actual, err := client.CreateV2(&requestData)
	assert.NoError(t, err)

	expected := &CreateResponse{
		ID:  "10057",
		Key: "TEST-3",
	}
	assert.Equal(t, expected, actual)

	testServer.statusCode(401)

	_, err = client.CreateV2(&requestData)
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
