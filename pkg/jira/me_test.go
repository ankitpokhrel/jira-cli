package jira

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMe(t *testing.T) {
	var unexpectedStatusCode bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/myself", r.URL.Path)

		if unexpectedStatusCode {
			w.WriteHeader(400)
		} else {
			resp, err := ioutil.ReadFile("./testdata/myself.json")
			assert.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write(resp)
		}
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.Me()
	assert.NoError(t, err)

	expected := &Me{
		Name:  "Person A",
		Email: "user@test.com",
	}
	assert.Equal(t, expected, actual)

	unexpectedStatusCode = true

	_, err = client.Me()
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}
