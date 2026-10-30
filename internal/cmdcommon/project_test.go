package cmdcommon

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

func TestResolveProjectType(t *testing.T) {
	newClient := func(handler http.HandlerFunc) (*jira.Client, func()) {
		server := httptest.NewServer(handler)
		client := jira.NewClient(jira.Config{Server: server.URL}, jira.WithTimeout(3*time.Second))
		return client, server.Close
	}

	apiType := func(style string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/rest/api/2/project/PRJ", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"key":"PRJ","style":"` + style + `"}`))
		}
	}
	apiError := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }

	t.Run("non-cloud returns cached value without calling the API", func(t *testing.T) {
		got, err := ResolveProjectType(nil, "PRJ", "classic", jira.InstallationTypeLocal, true)
		assert.NoError(t, err)
		assert.Equal(t, "classic", got)
	})

	t.Run("default project uses cached value without calling the API", func(t *testing.T) {
		got, err := ResolveProjectType(nil, "PRJ", "next-gen", jira.InstallationTypeCloud, false)
		assert.NoError(t, err)
		assert.Equal(t, "next-gen", got)
	})

	t.Run("empty project returns cached value without calling the API", func(t *testing.T) {
		got, err := ResolveProjectType(nil, "", "", jira.InstallationTypeCloud, false)
		assert.NoError(t, err)
		assert.Equal(t, "", got)
	})

	t.Run("overridden project resolves from the API", func(t *testing.T) {
		client, closeFn := newClient(apiType("next-gen"))
		defer closeFn()

		got, err := ResolveProjectType(client, "PRJ", "classic", jira.InstallationTypeCloud, true)
		assert.NoError(t, err)
		assert.Equal(t, "next-gen", got)
	})

	t.Run("missing cached value resolves from the API even without override", func(t *testing.T) {
		client, closeFn := newClient(apiType("classic"))
		defer closeFn()

		got, err := ResolveProjectType(client, "PRJ", "", jira.InstallationTypeCloud, false)
		assert.NoError(t, err)
		assert.Equal(t, "classic", got)
	})

	t.Run("overridden project errors on API failure instead of returning the wrong cached type", func(t *testing.T) {
		client, closeFn := newClient(apiError)
		defer closeFn()

		got, err := ResolveProjectType(client, "PRJ", "next-gen", jira.InstallationTypeCloud, true)
		assert.Error(t, err)
		assert.Equal(t, "", got)
	})

	t.Run("missing cached value falls back to cached on API failure without override", func(t *testing.T) {
		client, closeFn := newClient(apiError)
		defer closeFn()

		got, err := ResolveProjectType(client, "PRJ", "", jira.InstallationTypeCloud, false)
		assert.NoError(t, err)
		assert.Equal(t, "", got)
	})
}
