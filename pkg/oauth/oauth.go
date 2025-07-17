package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/utils"
)

const (
	// JIRA OAuth2 endpoints
	jiraAuthURL            = "https://auth.atlassian.com/authorize"
	jiraTokenURL           = "https://auth.atlassian.com/oauth/token"
	accessibleResourcesURL = "https://api.atlassian.com/oauth/token/accessible-resources"

	// Default OAuth settings
	defaultRedirectURI = "http://localhost:9876/callback"
	defaultPort        = ":9876"
	callbackPath       = "/callback"

	// OAuth timeout
	oauthTimeout = 5 * time.Minute

	// OAuth storage file name
	oauthSecretsFile = "oauth_secrets.json"
)

var defaultScopes = []string{
	"read:jira-user",
	"read:jira-work",
	"read:board-scope:jira-software",
	"read:project:jira",
	"write:jira-work",
	"offline_access", // This is required to get the refresh token from JIRA
}

// OAuthConfig holds OAuth configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

// ConfigureTokenResponse holds the OAuth token response
type ConfigureTokenResponse struct {
	AccessToken  string
	RefreshToken string
	CloudID      string
}

// GetOAuth2Config creates an OAuth2 config for the given client credentials
func GetOAuth2Config(clientID, clientSecret, redirectURI string, scopes []string) *oauth2.Config {
	if scopes == nil {
		scopes = defaultScopes
	}

	if redirectURI == "" {
		redirectURI = defaultRedirectURI
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  jiraAuthURL,
			TokenURL: jiraTokenURL,
		},
	}
}

// Configure performs the complete OAuth flow and returns tokens
func Configure() (*ConfigureTokenResponse, error) {
	// Collect OAuth credentials from user
	jiraDir, err := getJiraConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get Jira config directory: %w", err)
	}

	secretStorage := utils.FileSystemStorage{BaseDir: jiraDir}

	config, err := collectOAuthCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to collect OAuth credentials: %w", err)
	}

	// Perform OAuth flow
	tokens, err := performOAuthFlow(config)
	if err != nil {
		return nil, fmt.Errorf("OAuth flow failed: %w", err)
	}

	// Get Cloud ID for Atlassian API
	cloudID, err := getCloudID(accessibleResourcesURL, tokens.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloud ID: %w", err)
	}

	// Store all OAuth secrets in a single JSON file
	oauthSecrets := &OAuthSecrets{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		Expiry:       tokens.Expiry,
	}

	if err := utils.SaveJSON(secretStorage, oauthSecretsFile, oauthSecrets); err != nil {
		return nil, fmt.Errorf("failed to store OAuth secrets: %w", err)
	}

	return &ConfigureTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		CloudID:      cloudID,
	}, nil
}

// LoadOAuthSecrets loads OAuth secrets from storage
func LoadOAuthSecrets() (*OAuthSecrets, error) {
	jiraDir, err := getJiraConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get Jira config directory: %w", err)
	}

	secretStorage := utils.FileSystemStorage{BaseDir: jiraDir}
	secrets, err := utils.LoadJSON[OAuthSecrets](secretStorage, oauthSecretsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load OAuth secrets: %w", err)
	}

	return &secrets, nil
}

// HasOAuthCredentials checks if OAuth credentials are present
func HasOAuthCredentials() bool {
	_, err := LoadOAuthSecrets()
	return err == nil
}

// collectOAuthCredentials collects OAuth credentials from the user
func collectOAuthCredentials() (*OAuthConfig, error) {
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

	return &OAuthConfig{
		ClientID:     answers.ClientID,
		ClientSecret: answers.ClientSecret,
		RedirectURI:  answers.RedirectURI,
		Scopes:       defaultScopes,
	}, nil
}

// performOAuthFlow executes the OAuth authorization flow
func performOAuthFlow(config *OAuthConfig) (*oauth2.Token, error) {
	s := cmdutil.Info("Starting OAuth flow...")
	defer s.Stop()

	// OAuth2 configuration for JIRA
	oauthConfig := GetOAuth2Config(config.ClientID, config.ClientSecret, config.RedirectURI, config.Scopes)

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
func getCloudID(url string, accessToken string) (string, error) {
	s := cmdutil.Info("Fetching cloud ID...")
	defer s.Stop()

	// Create HTTP client with bearer token
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
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
