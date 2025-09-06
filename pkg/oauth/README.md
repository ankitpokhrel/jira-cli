# OAuth Package

This package provides OAuth2 authentication functionality for the JIRA CLI.

## Features

- Complete OAuth2 flow implementation for Atlassian JIRA
- Local HTTP server for OAuth callback handling
- Automatic browser opening for authorization
- Secure client secret storage
- Cloud ID retrieval for Atlassian API access
- PKCE (Proof Key for Code Exchange) support

## Usage

```go
import "github.com/ankitpokhrel/jira-cli/internal/pkg/oauth"

// Perform complete OAuth flow
tokenResponse, err := oauth.Configure()
if err != nil {
    log.Fatal(err)
}

// Access the tokens and cloud ID
accessToken := tokenResponse.AccessToken
refreshToken := tokenResponse.RefreshToken
cloudID := tokenResponse.CloudID
```

## Configuration

The `Configure()` function will:

1. Prompt the user for:

   - Jira App Client ID
   - Jira App Client Secret
   - Redirect URI (defaults to `http://localhost:9876/callback`)

2. Start a local HTTP server on port 9876 to handle the OAuth callback

3. Open the user's browser to the Atlassian authorization URL

4. Exchange the authorization code for access and refresh tokens

5. Retrieve the Cloud ID for API access

6. Store the client secret securely in `~/.jira/.oauth_secret`

## Security

- Client secrets are stored with restricted permissions (0600) in a separate file
- Client secrets are cleared from memory after secure storage
- The local server automatically shuts down after receiving the callback

## Requirements

- The redirect URI must be configured in your Atlassian OAuth app
- Port 9876 must be available for the local callback server
- The user must have a web browser available for authorization
