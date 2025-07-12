package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
)

const (
	// JIRA OAuth2 endpoints
	jiraAuthURL  = "https://auth.atlassian.com/authorize"
	jiraTokenURL = "https://auth.atlassian.com/oauth/token"

	// Default OAuth settings
	defaultRedirectURI = "http://localhost:9876/callback"
	defaultPort        = ":9876"
	callbackPath       = "/callback"

	// OAuth timeout
	oauthTimeout = 5 * time.Minute
)

const (
	OWNER_ONLY       = 0o700
	OWNER_READ_WRITE = 0o600
)

// Storage defines the interface for secret storage operations
type Storage interface {
	Save(key string, value []byte) error
	Load(key string) ([]byte, error)
}

// Secret represents a secret value with storage capabilities
type Secret struct {
	Key   string
	Value string
}

func (s Secret) String() string {
	return s.Value
}

func (s Secret) Save(storage Storage) error {
	if s.Key == "" {
		return fmt.Errorf("secret key cannot be empty")
	}
	return storage.Save(s.Key, []byte(s.Value))
}

func (s *Secret) Load(storage Storage, key string) error {
	if key == "" {
		return fmt.Errorf("secret key cannot be empty")
	}

	data, err := storage.Load(key)
	if err != nil {
		return err
	}

	s.Key = key
	s.Value = string(data)
	return nil
}

// FileSystemStorage implements Storage interface for filesystem operations
type FileSystemStorage struct {
	BaseDir string
}

func (fs FileSystemStorage) Save(key string, value []byte) error {
	if err := os.MkdirAll(fs.BaseDir, OWNER_ONLY); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(fs.BaseDir, key)
	return os.WriteFile(filePath, value, OWNER_READ_WRITE)
}

func (fs FileSystemStorage) Load(key string) ([]byte, error) {
	filePath := filepath.Join(fs.BaseDir, key)
	return os.ReadFile(filePath)
}

// Config holds OAuth configuration
type Config struct {
	ClientID     string
	ClientSecret Secret
	RedirectURI  string
	Scopes       []string
}

// ConfigureTokenResponse holds the OAuth token response
type ConfigureTokenResponse struct {
	AccessToken  Secret
	RefreshToken Secret
	CloudID      string
}

// Configure performs the complete OAuth flow and returns tokens
func Configure() (*ConfigureTokenResponse, error) {
	// Collect OAuth credentials from user

	jiraDir, err := getJiraConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get Jira config directory: %w", err)
	}

	secretStorage := FileSystemStorage{BaseDir: jiraDir}

	config, err := collectOAuthCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to collect OAuth credentials: %w", err)
	}

	// Perform OAuth flow
	tokens, err := performOAuthFlow(config)
	if err != nil {
		return nil, fmt.Errorf("OAuth flow failed: %w", err)
	}

	// Store client secret securely
	if err := config.ClientSecret.Save(secretStorage); err != nil {
		return nil, fmt.Errorf("failed to store client secret: %w", err)
	}

	accessToken := Secret{Key: "access_token", Value: tokens.AccessToken}
	refreshToken := Secret{Key: "refresh_token", Value: tokens.RefreshToken}

	if err := accessToken.Save(secretStorage); err != nil {
		return nil, fmt.Errorf("failed to store access token: %w", err)
	}

	if err := refreshToken.Save(secretStorage); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}
	// Get Cloud ID for Atlassian API
	cloudID, err := getCloudID(tokens.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloud ID: %w", err)
	}

	return &ConfigureTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		CloudID:      cloudID,
	}, nil
}

// collectOAuthCredentials collects OAuth credentials from the user
func collectOAuthCredentials() (*Config, error) {
	var questions []*survey.Question
	answers := struct {
		ClientID     string
		ClientSecret string
		RedirectURI  string
	}{}

	questions = append(questions, &survey.Question{
		Name: "clientID",
		Prompt: &survey.Input{
			Message: "Jira App Client ID:",
			Help:    "This is the client ID of your Jira App that you created for OAuth authentication.",
		},
	})

	questions = append(questions, &survey.Question{
		Name: "clientSecret",
		Prompt: &survey.Password{
			Message: "Jira App Client Secret:",
			Help:    "This is the client secret of your Jira App that you created for OAuth authentication.",
		},
	})

	questions = append(questions, &survey.Question{
		Name: "redirectURI",
		Prompt: &survey.Input{
			Default: defaultRedirectURI,
			Message: "Redirect URI:",
			Help:    "The redirect URL for Jira App. Recommended to set as localhost.",
		},
	})

	if err := survey.Ask(questions, &answers, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	return &Config{
		ClientID:     answers.ClientID,
		ClientSecret: Secret{Key: "client_secret", Value: answers.ClientSecret},
		RedirectURI:  answers.RedirectURI,
		Scopes: []string{
			"read:jira-user",
			"read:jira-work",
			"write:jira-work",
			"offline_access",
			"read:board-scope:jira-software",
			"read:project:jira",
		},
	}, nil
}

// performOAuthFlow executes the OAuth authorization flow
func performOAuthFlow(config *Config) (*oauth2.Token, error) {
	s := cmdutil.Info("Starting OAuth flow...")
	defer s.Stop()

	// OAuth2 configuration for JIRA
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

	// Generate authorization URL with PKCE
	verifier := oauth2.GenerateVerifier()
	authURL := oauthConfig.AuthCodeURL(verifier, oauth2.AccessTypeOffline)

	// Start local server to handle callback
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

				// Send success response to browser
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

				codeChan <- code
			} else {
				http.NotFound(w, r)
			}
		}),
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Open browser for authorization
	fmt.Printf("Opening browser for authorization...\n")
	fmt.Printf("If the browser doesn't open automatically, please visit: %s\n", authURL)

	// Try to open browser
	if err := browser.OpenURL(authURL); err != nil {
		fmt.Printf("Could not open browser automatically: %v\n", err)
		fmt.Printf("Please manually visit: %s\n", authURL)
	}

	// Wait for authorization code
	select {
	case code := <-codeChan:
		// Shutdown server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)

		// Exchange code for token
		s.Stop()
		s = cmdutil.Info("Exchanging authorization code for access token...")
		defer s.Stop()

		token, err := oauthConfig.Exchange(context.Background(), code)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange code for token: %w", err)
		}

		return token, nil

	case err := <-errChan:
		// Shutdown server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return nil, fmt.Errorf("OAuth flow failed: %w", err)

	case <-time.After(oauthTimeout):
		// Shutdown server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return nil, fmt.Errorf("OAuth flow timed out after %v", oauthTimeout)
	}
}

// getCloudID retrieves the Cloud ID for the authenticated user
func getCloudID(accessToken string) (string, error) {
	s := cmdutil.Info("Fetching cloud ID...")
	defer s.Stop()

	// Create HTTP client with bearer token
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", "https://api.atlassian.com/oauth/token/accessible-resources", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get accessible resources: status %d", resp.StatusCode)
	}

	// Parse response to get cloud ID
	var resourceResponse []struct {
		ID        string   `json:"id"`
		Name      string   `json:"name"`
		URL       string   `json:"url"`
		Scopes    []string `json:"scopes"`
		AvatarURL string   `json:"avatarUrl"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&resourceResponse); err != nil {
		return "", fmt.Errorf("failed to decode accessible resources response: %w", err)
	}

	if len(resourceResponse) == 0 {
		return "", fmt.Errorf("no accessible resources found or cloud ID not found")
	}

	return resourceResponse[0].ID, nil
}

func getJiraConfigDir() (string, error) {
	home, err := cmdutil.GetConfigHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".jira"), nil
}
