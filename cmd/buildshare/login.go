package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/buildshare/cli/internal/api"
	"github.com/buildshare/cli/internal/auth"
	"github.com/buildshare/cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginAPIKey string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with BuildShare",
	Long: `Log in to your BuildShare account using email OTP, API key, or token.

For CI/CD or non-interactive login:
  buildshare login --token <token>
  buildshare login --api-key <key>

Or set the BUILDSHARE_TOKEN environment variable.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client := newClient()

		tokenOrKey := tokenFlag
		if tokenOrKey == "" {
			tokenOrKey = loginAPIKey
		}

		// Token or API key flow
		if tokenOrKey != "" {
			return loginWithTokenOrKey(ctx, client, tokenOrKey)
		}

		// Interactive OTP flow
		if cfg.CI {
			printer.Error("Interactive login is not available in CI mode.")
			fmt.Fprintln(os.Stderr, "\nUse:\n    buildshare login --token <token>\n    buildshare login --api-key <key>\n\nor:\n    BUILDSHARE_TOKEN=<token> buildshare <command>")
			os.Exit(2)
		}

		return loginInteractive(ctx, client)
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginAPIKey, "api-key", "", "Authenticate using an API key (alias for --token)")
	loginCmd.Flags().StringVarP(&loginAPIKey, "key", "k", "", "Authenticate using an API key (shorthand)")
	rootCmd.AddCommand(loginCmd)
}

func loginInteractive(ctx context.Context, client *api.Client) error {
	reader := bufio.NewReader(os.Stdin)

	printer.Header("BuildShare Login")
	printer.Newline()

	// Prompt for email
	fmt.Print("  Enter your email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	email = strings.TrimSpace(email)
	if email == "" {
		printer.Error("Email is required.")
		return nil
	}

	// Request OTP
	printer.Info("Sending OTP to " + email + "...")
	_, err = client.Post(ctx, "/user/login", api.LoginRequest{Email: email})
	if err != nil {
		if api.IsNotFound(err) {
			printer.Error("No account found for " + email)
			printer.Info("Register at https://buildshare.in to create an account.")
			return nil
		}
		return fmt.Errorf("failed to send OTP: %w", err)
	}
	printer.Success("OTP sent to " + email)
	printer.Newline()

	// Prompt for OTP (hide input)
	fmt.Print("  Enter OTP: ")
	var otp string
	if term.IsTerminal(int(os.Stdin.Fd())) {
		otpBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("failed to read OTP: %w", err)
		}
		otp = strings.TrimSpace(string(otpBytes))
		fmt.Println() // newline after hidden input
	} else {
		otp, _ = reader.ReadString('\n')
		otp = strings.TrimSpace(otp)
	}

	if otp == "" {
		printer.Error("OTP is required.")
		return nil
	}

	// Verify OTP
	printer.Info("Verifying OTP...")
	resp, err := client.Post(ctx, "/user/verify-otp", api.VerifyOTPRequest{
		Email: email,
		OTP:   otp,
	})
	if err != nil {
		return fmt.Errorf("OTP verification failed: %w", err)
	}

	var result api.AuthResult
	if err := api.Decode(resp.Data, &result); err != nil {
		return fmt.Errorf("unexpected response: %w", err)
	}

	// Save credentials
	if err := auth.Save(&auth.Credentials{
		Token:  result.Token,
		Email:  result.Email,
		Name:   result.Name,
		UserID: result.UserID,
	}); err != nil {
		printer.Warn("Could not save credentials: " + err.Error())
		printer.Info("Token: " + result.Token)
		return nil
	}

	printer.Newline()
	printer.Success(fmt.Sprintf("Logged in as %s (%s)", result.Name, result.Email))
	return nil
}

func loginWithTokenOrKey(ctx context.Context, client *api.Client, tokenOrKey string) error {
	tokenOrKey = strings.TrimSpace(tokenOrKey)
	if tokenOrKey == "" {
		printer.Error("Token or API key is required.")
		return nil
	}

	if strings.HasPrefix(tokenOrKey, "sk_") {
		return loginWithAPIKey(ctx, client, tokenOrKey)
	}

	if strings.HasPrefix(tokenOrKey, "ey") {
		return loginWithAccessToken(ctx, tokenOrKey)
	}

	// Try API key verification first
	if err := loginWithAPIKey(ctx, client, tokenOrKey); err == nil {
		return nil
	}

	// Fallback to access token verification
	return loginWithAccessToken(ctx, tokenOrKey)
}

func loginWithAPIKey(ctx context.Context, client *api.Client, key string) error {
	printer.Info("Verifying API key...")

	resp, err := client.Post(ctx, "/api-key/verify", api.VerifyAPIKeyRequest{APIKey: key})
	if err != nil {
		return fmt.Errorf("API key verification failed: %w", err)
	}

	var result api.APIKeyVerifyResult
	if err := api.Decode(resp.Data, &result); err != nil {
		return fmt.Errorf("unexpected response: %w", err)
	}

	if err := auth.Save(&auth.Credentials{
		Token:  result.AccessToken,
		Email:  result.User.Email,
		Name:   result.User.Name,
		UserID: result.User.UserID,
	}); err != nil {
		printer.Warn("Could not save credentials: " + err.Error())
		return nil
	}

	printer.Success(fmt.Sprintf("Authenticated as %s (%s)", result.User.Name, result.User.Email))
	return nil
}

func loginWithAccessToken(ctx context.Context, token string) error {
	printer.Info("Verifying token...")

	authClient := api.New(config.APIURL, token)
	resp, err := authClient.Get(ctx, "/user/profile")
	if err != nil {
		return fmt.Errorf("token verification failed: %w", err)
	}

	var profile api.UserProfile
	if err := api.Decode(resp.Data, &profile); err != nil {
		return fmt.Errorf("unexpected response: %w", err)
	}

	if err := auth.Save(&auth.Credentials{
		Token:  token,
		Email:  profile.Email,
		Name:   profile.Name,
		UserID: profile.UserID,
	}); err != nil {
		printer.Warn("Could not save credentials: " + err.Error())
		return nil
	}

	printer.Success(fmt.Sprintf("Authenticated as %s (%s)", profile.Name, profile.Email))
	return nil
}
