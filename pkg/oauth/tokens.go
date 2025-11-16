package oauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/utils"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// OAuthSecrets holds all OAuth secrets in a single structure.
type OAuthSecrets struct {
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// PersistentTokenSource implements oauth2.TokenSource with automatic token persistence.
type PersistentTokenSource struct {
	clientID        string
	clientSecret    string
	storage         utils.Storage
	fallbackStorage utils.Storage
	usingFallback   bool
}

// IsExpired checks if the access token is expired.
func (o *OAuthSecrets) IsExpired() bool {
	return time.Now().After(o.Expiry)
}

// IsValid checks if the OAuth secrets are valid and not expired.
func (o *OAuthSecrets) IsValid() bool {
	return o.AccessToken != "" && !o.IsExpired()
}

// ToOAuth2Token converts OAuthSecrets to oauth2.Token.
func (o *OAuthSecrets) ToOAuth2Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  o.AccessToken,
		RefreshToken: o.RefreshToken,
		TokenType:    o.TokenType,
		Expiry:       o.Expiry,
	}
}

// FromOAuth2Token updates OAuthSecrets from oauth2.Token.
func (o *OAuthSecrets) FromOAuth2Token(token *oauth2.Token) {
	o.AccessToken = token.AccessToken
	o.RefreshToken = token.RefreshToken
	o.TokenType = token.TokenType
	o.Expiry = token.Expiry
}

// NewPersistentTokenSource creates a new TokenSource that persists tokens.
// It attempts to use keyring storage first, falling back to filesystem storage if keyring fails.
func NewPersistentTokenSource(login, clientID, clientSecret string) (*PersistentTokenSource, error) {
	jiraDir, err := getJiraConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get Jira config directory: %w", err)
	}

	keyringStorage := utils.NewKeyRingStorage(login)
	fallbackFileSystemStorage := utils.FileSystemStorage{BaseDir: jiraDir}

	return &PersistentTokenSource{
		clientID:        clientID,
		clientSecret:    clientSecret,
		storage:         keyringStorage,
		fallbackStorage: fallbackFileSystemStorage,
		usingFallback:   false,
	}, nil
}

// Token implements oauth2.TokenSource interface.
func (pts *PersistentTokenSource) Token() (*oauth2.Token, error) {
	// Load current token from storage
	secrets, err := utils.LoadJSON[OAuthSecrets](pts.storage, oauthSecretsFile)
	if err != nil {
		// If primary storage fails and we're not already using fallback, try fallback
		if !pts.usingFallback && pts.fallbackStorage != nil {
			fmt.Println("Warning: Primary storage failed, falling back to FileSystemStorage for OAuth tokens")
			secrets, err = utils.LoadJSON[OAuthSecrets](pts.fallbackStorage, oauthSecretsFile)
			if err == nil {
				// Successfully loaded from fallback, switch to using it
				pts.storage = pts.fallbackStorage
				pts.usingFallback = true
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load OAuth secrets: %w", err)
		}
	}

	token := secrets.ToOAuth2Token()

	// If token is still valid, return it
	if token.Valid() {
		return token, nil
	}

	// Token needs refresh - create OAuth2 config for refresh
	oauthConfig := &oauth2.Config{
		ClientID:     pts.clientID,
		ClientSecret: pts.clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  jiraAuthURL,
			TokenURL: jiraTokenURL,
		},
	}

	// Refresh the token
	refreshedToken, err := oauthConfig.TokenSource(context.Background(), token).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh OAuth token: %w", err)
	}

	// Save the refreshed token
	secrets.FromOAuth2Token(refreshedToken)
	if err := pts.saveSecrets(&secrets); err != nil {
		// Log error but don't fail the request - we still have a valid token
		fmt.Printf("Warning: failed to save refreshed OAuth token: %v\n", err)
	}

	return refreshedToken, nil
}

// saveSecrets attempts to save secrets to primary storage, falling back if necessary.
func (pts *PersistentTokenSource) saveSecrets(secrets *OAuthSecrets) error {
	err := utils.SaveJSON(pts.storage, oauthSecretsFile, secrets)
	if err != nil && !pts.usingFallback && pts.fallbackStorage != nil {
		if errors.Is(err, keyring.ErrSetDataTooBig) {
			cmdutil.Warn("Data was too big to save to the keyring, falling back to filesystem storage")
		}
		err = utils.SaveJSON(pts.fallbackStorage, oauthSecretsFile, secrets)
		if err == nil {
			// Successfully saved to fallback, switch to using it
			pts.storage = pts.fallbackStorage
			pts.usingFallback = true
		}
	}
	return err
}

// LoadOAuth2TokenSource creates a TokenSource from stored OAuth secrets.
func LoadOAuth2TokenSource(login string) (oauth2.TokenSource, error) {
	// Load OAuth secrets to get client credentials
	secrets, err := LoadOAuthSecrets(login)
	if err != nil {
		return nil, fmt.Errorf("failed to load OAuth secrets: %w", err)
	}

	// Create persistent token source
	tokenSource, err := NewPersistentTokenSource(login, secrets.ClientID, secrets.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create token source: %w", err)
	}

	return tokenSource, nil
}
