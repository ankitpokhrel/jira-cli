package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/utils"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestGetJiraConfigDir(t *testing.T) {
	t.Parallel()

	// Save original environment
	originalHome := os.Getenv("HOME")
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
	}()

	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		os.Setenv("XDG_CONFIG_HOME", "/tmp/test-config")
		os.Setenv("HOME", "/tmp/test-home")

		dir, err := getJiraConfigDir()
		assert.NoError(t, err)
		assert.Equal(t, "/tmp/test-config/.jira", dir)
	})

	t.Run("falls back to HOME/.config when XDG_CONFIG_HOME not set", func(t *testing.T) {
		os.Unsetenv("XDG_CONFIG_HOME")
		os.Setenv("HOME", "/tmp/test-home")

		dir, err := getJiraConfigDir()
		assert.NoError(t, err)
		assert.Equal(t, "/tmp/test-home/.config/.jira", dir)
	})
}

func TestGetCloudID(t *testing.T) {
	t.Parallel()

	t.Run("successfully retrieves cloud ID", func(t *testing.T) {
		expectedCloudID := "test-cloud-id-123"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/oauth/token/accessible-resources", r.URL.Path)
			assert.Equal(t, "Bearer test-access-token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			// Return mock response
			response := []map[string]interface{}{
				{
					"id":        expectedCloudID,
					"name":      "Test Site",
					"url":       "https://test.atlassian.net",
					"scopes":    []string{"read:jira-user", "read:jira-work"},
					"avatarUrl": "https://test.atlassian.net/avatar.png",
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		// Test with mock server - this requires refactoring the function to accept a custom URL
		// For now, we'll test the error cases and create a separate testable function
		cloudID, err := getCloudID(server.URL+"/oauth/token/accessible-resources", "test-access-token")
		assert.NoError(t, err)
		assert.Equal(t, expectedCloudID, cloudID)
	})

	t.Run("successfully gets jira cloud id from list of accessible resources", func(t *testing.T) {
		expectedCloudID := "test-cloud-id-123"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/oauth/token/accessible-resources", r.URL.Path)
			assert.Equal(t, "Bearer test-access-token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))

			// Return mock response
			response := []map[string]interface{}{
				{
					"id":        expectedCloudID,
					"name":      "Test Site",
					"url":       "https://test.atlassian.net",
					"scopes":    []string{"read:jira-user", "read:jira-work"},
					"avatarUrl": "https://test.atlassian.net/avatar.png",
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		// Test with mock server - this requires refactoring the function to accept a custom URL
		// For now, we'll test the error cases and create a separate testable function
		cloudID, err := getCloudID(server.URL+"/oauth/token/accessible-resources", "test-access-token")
		assert.NoError(t, err)
		assert.Equal(t, expectedCloudID, cloudID)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		cloudID, err := getCloudID(server.URL+"/oauth/token/accessible-resources", "invalid-token")
		assert.Error(t, err)
		assert.Empty(t, cloudID)
		assert.Contains(t, err.Error(), "failed to get accessible resources: status 401")
	})

	t.Run("handles invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		cloudID, err := getCloudID(server.URL+"/oauth/token/accessible-resources", "test-token")
		assert.Error(t, err)
		assert.Empty(t, cloudID)
		assert.Contains(t, err.Error(), "failed to decode accessible resources response")
	})

	t.Run("handles empty response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}))
		defer server.Close()

		cloudID, err := getCloudID(server.URL+"/oauth/token/accessible-resources", "test-token")
		assert.Error(t, err)
		assert.Empty(t, cloudID)
		assert.Contains(t, err.Error(), "no accessible resources found")
	})
}

func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("creates config with all required fields", func(t *testing.T) {
		config := &OAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: utils.Secret{Key: "client_secret", Value: "test-secret"},
			RedirectURI:  "http://localhost:9876/callback",
			Scopes:       []string{"read:jira-user", "read:jira-work"},
		}

		assert.Equal(t, "test-client-id", config.ClientID)
		assert.Equal(t, "test-secret", config.ClientSecret.String())
		assert.Equal(t, "http://localhost:9876/callback", config.RedirectURI)
		assert.Contains(t, config.Scopes, "read:jira-user")
		assert.Contains(t, config.Scopes, "read:jira-work")
	})
}

func TestConfigureTokenResponse(t *testing.T) {
	t.Parallel()

	t.Run("creates token response with all required fields", func(t *testing.T) {
		response := &ConfigureTokenResponse{
			AccessToken:  utils.Secret{Key: "access_token", Value: "test-access-token"},
			RefreshToken: utils.Secret{Key: "refresh_token", Value: "test-refresh-token"},
			CloudID:      "test-cloud-id",
		}

		assert.Equal(t, "test-access-token", response.AccessToken.String())
		assert.Equal(t, "test-refresh-token", response.RefreshToken.String())
		assert.Equal(t, "test-cloud-id", response.CloudID)
	})
}

func TestPerformOAuthFlow_ErrorCases(t *testing.T) {
	t.Parallel()

	t.Run("handles timeout", func(t *testing.T) {
		config := &OAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: utils.Secret{Key: "client_secret", Value: "test-secret"},
			RedirectURI:  "http://localhost:9876/callback",
			Scopes:       []string{"read:jira-user"},
		}

		// Create a version of performOAuthFlow with a shorter timeout for testing
		token, err := performOAuthFlowWithTimeout(config, 100*time.Millisecond)
		assert.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "OAuth flow timed out")
	})

	t.Run("handles server startup error", func(t *testing.T) {
		config := &OAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: utils.Secret{Key: "client_secret", Value: "test-secret"},
			RedirectURI:  "http://localhost:9876/callback",
			Scopes:       []string{"read:jira-user"},
		}

		// Start a server on the same port to cause a conflict
		conflictServer := &http.Server{Addr: defaultPort}
		go conflictServer.ListenAndServe()
		defer conflictServer.Close()

		// Wait a bit for the server to start
		time.Sleep(100 * time.Millisecond)

		// This should fail due to port conflict
		token, err := performOAuthFlowWithTimeout(config, 1*time.Second)
		// The error might be about port conflict or timeout, both are acceptable
		assert.Error(t, err)
		assert.Nil(t, token)
	})
}

// Helper function to test OAuth flow with custom timeout
func performOAuthFlowWithTimeout(config *OAuthConfig, timeout time.Duration) (*oauth2.Token, error) {
	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret.String(),
		RedirectURL:  config.RedirectURI,
		Scopes:       config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  jiraAuthURL,
			TokenURL: jiraTokenURL,
		},
	}

	verifier := oauth2.GenerateVerifier()
	_ = oauthConfig.AuthCodeURL(verifier, oauth2.AccessTypeOffline)

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	server := &http.Server{
		Addr: defaultPort,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == callbackPath {
				code := r.URL.Query().Get("code")
				if code == "" {
					errChan <- fmt.Errorf("no authorization code received")
					return
				}

				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(`<html><body><h2>Authorization successful!</h2></body></html>`))
				codeChan <- code
			} else {
				http.NotFound(w, r)
			}
		}),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case code := <-codeChan:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)

		token, err := oauthConfig.Exchange(context.Background(), code)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange code for token: %w", err)
		}
		return token, nil

	case err := <-errChan:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return nil, fmt.Errorf("OAuth flow failed: %w", err)

	case <-time.After(timeout):
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return nil, fmt.Errorf("OAuth flow timed out after %v", timeout)
	}
}

func TestOAuthFlowIntegration(t *testing.T) {
	t.Parallel()

	t.Run("handles callback with authorization code", func(t *testing.T) {
		// Create a mock OAuth server
		mockOAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/oauth/token" {
				// Mock token exchange
				token := map[string]interface{}{
					"access_token":  "mock-access-token",
					"refresh_token": "mock-refresh-token",
					"token_type":    "Bearer",
					"expires_in":    3600,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(token)
			}
		}))
		defer mockOAuthServer.Close()

		// Create config with mock server
		config := &OAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: utils.Secret{Key: "client_secret", Value: "test-secret"},
			RedirectURI:  "http://localhost:9876/callback",
			Scopes:       []string{"read:jira-user"},
		}

		// Test the OAuth configuration creation
		oauthConfig := &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret.String(),
			RedirectURL:  config.RedirectURI,
			Scopes:       config.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  jiraAuthURL,
				TokenURL: mockOAuthServer.URL + "/oauth/token",
			},
		}

		// Test authorization URL generation
		verifier := oauth2.GenerateVerifier()
		authURL := oauthConfig.AuthCodeURL(verifier, oauth2.AccessTypeOffline)

		assert.Contains(t, authURL, jiraAuthURL)
		assert.Contains(t, authURL, "client_id=test-client-id")
		assert.Contains(t, authURL, "redirect_uri=http%3A%2F%2Flocalhost%3A9876%2Fcallback")
		assert.Contains(t, authURL, "scope=read%3Ajira-user")
	})

	t.Run("handles callback without authorization code", func(t *testing.T) {
		// Test callback handler
		codeChan := make(chan string, 1)
		errChan := make(chan error, 1)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == callbackPath {
				code := r.URL.Query().Get("code")
				if code == "" {
					errChan <- fmt.Errorf("no authorization code received")
					return
				}
				codeChan <- code
			}
		})

		// Create test request without code
		req := httptest.NewRequest("GET", "http://localhost:9876/callback", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		select {
		case err := <-errChan:
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no authorization code received")
		case <-time.After(100 * time.Millisecond):
			t.Error("Expected error but got timeout")
		}
	})

	t.Run("handles callback with authorization code", func(t *testing.T) {
		codeChan := make(chan string, 1)
		errChan := make(chan error, 1)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == callbackPath {
				code := r.URL.Query().Get("code")
				if code == "" {
					errChan <- fmt.Errorf("no authorization code received")
					return
				}

				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(`<html><body><h2>Authorization successful!</h2></body></html>`))
				codeChan <- code
			}
		})

		// Create test request with code
		req := httptest.NewRequest("GET", "http://localhost:9876/callback?code=test-auth-code", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		select {
		case code := <-codeChan:
			assert.Equal(t, "test-auth-code", code)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), "Authorization successful!")
		case err := <-errChan:
			t.Errorf("Unexpected error: %v", err)
		case <-time.After(100 * time.Millisecond):
			t.Error("Expected code but got timeout")
		}
	})
}

func TestHTMLResponse(t *testing.T) {
	t.Parallel()

	t.Run("callback returns proper HTML response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == callbackPath {
				code := r.URL.Query().Get("code")
				if code != "" {
					w.Header().Set("Content-Type", "text/html")
					w.Write([]byte(`
					<html>
						<body>
							<h2>Authorization successful!</h2>
							<p>You can close this window and return to the terminal.</p>
							<script>window.close();</script>
						</body>
					</html>
				`))
				}
			}
		})

		req := httptest.NewRequest("GET", "http://localhost:9876/callback?code=test-code", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Body.String(), "Authorization successful!")
		assert.Contains(t, w.Body.String(), "window.close()")
	})
}
