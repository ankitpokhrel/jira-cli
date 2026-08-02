package refresh

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"

	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// NewCmdRefresh is a refresh command.
func NewCmdRefresh() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh session cookie for cookie-based authentication",
		Long: `Refresh session cookie for cookie-based authentication.

This command is only applicable when using 'cookie' auth type.
It allows you to update your JSESSIONID without re-running the full 'jira init' setup.`,
		Run: refresh,
	}
}

func refresh(cmd *cobra.Command, _ []string) {
	authType := viper.GetString("auth_type")
	if authType != string(jira.AuthTypeCookie) {
		cmdutil.Failed("This command is only for cookie-based authentication (current auth_type: %s)", authType)
		return
	}

	server := viper.GetString("server")
	login := viper.GetString("login")

	if server == "" || login == "" {
		cmdutil.Failed("Missing server or login in config. Please run 'jira init' first.")
		return
	}

	fmt.Println("Refresh session cookie for", server)
	fmt.Println()
	fmt.Println("1. Open", server, "in a browser")
	fmt.Println("2. Sign in (authenticate via SSO/certificate as needed)")
	fmt.Println("3. Open browser DevTools (F12) → Application/Storage → Cookies")
	fmt.Println("4. Find the cookie named 'JSESSIONID' and copy its value")
	fmt.Println()

	var sessionCookie string
	prompt := &survey.Password{
		Message: "Paste JSESSIONID value:",
		Help:    "The session cookie will be validated and stored securely in your system keychain",
	}

	if err := survey.AskOne(prompt, &sessionCookie, survey.WithValidator(survey.Required)); err != nil {
		cmdutil.Failed("Failed to read input: %s", err.Error())
		return
	}

	// Validate cookie
	s := cmdutil.Info("Validating session cookie...")

	client := jira.NewClient(jira.Config{
		Server:   server,
		APIToken: sessionCookie,
		AuthType: &[]jira.AuthType{jira.AuthTypeCookie}[0],
	})

	me, err := client.Me()
	if err != nil {
		s.Stop()
		cmdutil.Failed("Failed to validate cookie: %s", err.Error())
		return
	}
	s.Stop()

	// Verify it's the same user
	if me.Login != login {
		cmdutil.Failed("Cookie belongs to user '%s' but config expects '%s'", me.Login, login)
		return
	}

	// Store in keychain
	if err := keyring.Set("jira-cli", login, sessionCookie); err != nil {
		cmdutil.Failed("Failed to store session cookie in keychain: %s", err.Error())
		return
	}

	cmdutil.Success("Session refreshed for %s (%s)", me.Name, me.Login)
}
