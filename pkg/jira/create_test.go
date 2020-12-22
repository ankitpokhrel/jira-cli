package jira

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type createTestServer struct{ code int }

func (c createTestServer) serve(t *testing.T, expectedBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		actualBody := new(strings.Builder)
		_, _ = io.Copy(actualBody, r.Body)

		assert.Equal(t, expectedBody, actualBody.String())

		if c.code == 201 {
			resp, err := ioutil.ReadFile("./testdata/create.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(201)
			_, _ = w.Write(resp)
		} else {
			w.WriteHeader(c.code)
		}
	}))
}

func (c createTestServer) statusCode(code int) {
	c.code = code
}

func TestCreate(t *testing.T) {
	expectedBody := `{"update":{},"fields":{"project":{"key":"TEST"},"issuetype":{"name":"Bug"},` +
		`"summary":"Test bug","description":{"version":1,"type":"doc","content":[{"type":"paragraph","content":` +
		`[{"type":"text","text":"Test description"}]}]},"priority":{"name":"Normal"},"labels":["test","dev"]}}`
	testServer := createTestServer{code: 201}
	server := testServer.serve(t, expectedBody)
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3))

	requestData := CreateRequest{
		Project:   "TEST",
		IssueType: "Bug",
		Summary:   "Test bug",
		Body:      "Test description",
		Priority:  "Normal",
		Labels:    []string{"test", "dev"},
	}
	actual, err := client.Create(&requestData)
	assert.NoError(t, err)

	expected := &CreateResponse{
		ID:  "10057",
		Key: "TEST-3",
	}

	assert.Equal(t, expected, actual)

	testServer.statusCode(400)

	_, err = client.Create(&requestData)
	assert.Error(t, ErrUnexpectedStatusCode, err)
}

func TestCreateEpic(t *testing.T) {
	expectedBody := `{"update":{},"fields":{"customfield_10001":"CLI","description":{"content":[{"content":[{"text":` +
		`"Test description","type":"text"}],"type":"paragraph"}],"type":"doc","version":1},"issuetype":{"name":` +
		`"Bug"},"priority":{"name":"Normal"},"project":{"key":"TEST"},"summary":"Test bug"}}`
	testServer := createTestServer{code: 201}
	server := testServer.serve(t, expectedBody)
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3))
	requestData := CreateRequest{
		Project:       "TEST",
		IssueType:     "Bug",
		Name:          "CLI",
		Summary:       "Test bug",
		Body:          "Test description",
		Priority:      "Normal",
		EpicFieldName: "customfield_10001",
	}
	actual, err := client.Create(&requestData)
	assert.NoError(t, err)

	expected := &CreateResponse{
		ID:  "10057",
		Key: "TEST-3",
	}

	assert.Equal(t, expected, actual)

	testServer.statusCode(400)

	_, err = client.Create(&requestData)
	assert.Error(t, ErrUnexpectedStatusCode, err)
}
