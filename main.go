package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	configDir     = ".codex-sweet"
	profilesFile  = "profiles.json"
	codexAuthPath = ".codex/auth.json"
)

type TokenInfo struct {
	IDToken      string `json:"id_token,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	AccountID    string `json:"account_id,omitempty"`
}

type CodexAuth struct {
	AuthMode     string     `json:"auth_mode,omitempty"`
	OpenAIAPIKey *string    `json:"OPENAI_API_KEY,omitempty"`
	Tokens       *TokenInfo `json:"tokens,omitempty"`
	LastRefresh  string     `json:"last_refresh,omitempty"`
}

type Profile struct {
	Name      string    `json:"name"`
	Auth      CodexAuth `json:"auth"`
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"`
}

type ProfileManager struct {
	Profiles map[string]*Profile `json:"profiles"`
	Current  string              `json:"current"`
}

type RateWindow struct {
	UsedPercent        int   `json:"used_percent"`
	LimitWindowSeconds int   `json:"limit_window_seconds"`
	ResetAfterSeconds  int   `json:"reset_after_seconds"`
	ResetAt            int64 `json:"reset_at"`
}

type RateLimit struct {
	Allowed         bool        `json:"allowed"`
	LimitReached    bool        `json:"limit_reached"`
	PrimaryWindow   *RateWindow `json:"primary_window"`
	SecondaryWindow *RateWindow `json:"secondary_window"`
}

type Credits struct {
	HasCredits          bool   `json:"has_credits"`
	Unlimited           bool   `json:"unlimited"`
	Balance             string `json:"balance"`
	ApproxLocalMessages []int  `json:"approx_local_messages"`
	ApproxCloudMessages []int  `json:"approx_cloud_messages"`
}

type UsageResponse struct {
	UserID              string    `json:"user_id"`
	AccountID           string    `json:"account_id"`
	Email               string    `json:"email"`
	PlanType            string    `json:"plan_type"`
	RateLimit           RateLimit `json:"rate_limit"`
	CodeReviewRateLimit RateLimit `json:"code_review_rate_limit"`
	Credits             Credits   `json:"credits"`
}

func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	}
	if seconds < 86400 {
		hours := seconds / 3600
		mins := (seconds % 3600) / 60
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	return fmt.Sprintf("%dd %dh", days, hours)
}

func drawProgressBar(usedPercent int, width int) string {
	leftPercent := 100 - usedPercent
	filled := (leftPercent * width) / 100
	empty := width - filled

	bar := "["
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += " "
	}
	bar += "]"
	return bar
}

func formatResetTime(resetAt int64) string {
	t := time.Unix(resetAt, 0)
	now := time.Now()

	// If same day, just show time
	if t.Format("2006-01-02") == now.Format("2006-01-02") {
		return t.Format("15:04")
	}

	// If different day, show time and date
	return t.Format("15:04 on 02 Jan")
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return home
}

func getConfigPath() string {
	return filepath.Join(getHomeDir(), configDir)
}

func getProfilesPath() string {
	return filepath.Join(getConfigPath(), profilesFile)
}

func getCodexAuthPath() string {
	return filepath.Join(getHomeDir(), codexAuthPath)
}

func ensureConfigDir() error {
	return os.MkdirAll(getConfigPath(), 0755)
}

func loadProfiles() (*ProfileManager, error) {
	pm := &ProfileManager{
		Profiles: make(map[string]*Profile),
	}

	data, err := os.ReadFile(getProfilesPath())
	if err != nil {
		if os.IsNotExist(err) {
			return pm, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, pm); err != nil {
		return nil, err
	}

	return pm, nil
}

func (pm *ProfileManager) save() error {
	data, err := json.MarshalIndent(pm, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getProfilesPath(), data, 0600)
}

func loadCodexAuth() (*CodexAuth, error) {
	data, err := os.ReadFile(getCodexAuthPath())
	if err != nil {
		return nil, err
	}

	var auth CodexAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, err
	}

	return &auth, nil
}

func saveCodexAuth(auth *CodexAuth) error {
	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}

	codexDir := filepath.Dir(getCodexAuthPath())
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(getCodexAuthPath(), data, 0600)
}

func getUsage(accessToken, accountID string) (*UsageResponse, error) {
	url := "https://chatgpt.com/backend-api/wham/usage"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Mimic codex-tui user agent
	osVersion := runtime.GOOS
	arch := runtime.GOARCH
	userAgent := fmt.Sprintf("codex-sweet/0.1.0 (%s; %s)", osVersion, arch)

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("chatgpt-account-id", accountID)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Host", "chatgpt.com")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var usage UsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return nil, err
	}

	return &usage, nil
}

func getEmailFromAuth(auth *CodexAuth) (string, error) {
	// Try to get email from API if we have ChatGPT auth
	if auth.AuthMode == "chatgpt" && auth.Tokens != nil && auth.Tokens.AccessToken != "" && auth.Tokens.AccountID != "" {
		usage, err := getUsage(auth.Tokens.AccessToken, auth.Tokens.AccountID)
		if err == nil && usage.Email != "" {
			return usage.Email, nil
		}
	}

	// Fallback: try to decode from ID token (JWT)
	if auth.Tokens != nil && auth.Tokens.IDToken != "" {
		// Simple JWT decode (not validating signature, just extracting claims)
		parts := strings.Split(auth.Tokens.IDToken, ".")
		if len(parts) >= 2 {
			payload, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err == nil {
				var claims map[string]interface{}
				if err := json.Unmarshal(payload, &claims); err == nil {
					if email, ok := claims["email"].(string); ok && email != "" {
						return email, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("unable to extract email from credentials")
}

func cmdSave() *cobra.Command {
	return &cobra.Command{
		Use:   "save",
		Short: "Save current Codex credentials as a profile (auto-named by email)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ensureConfigDir(); err != nil {
				return err
			}

			auth, err := loadCodexAuth()
			if err != nil {
				return fmt.Errorf("failed to read ~/.codex/auth.json: %w", err)
			}

			// Extract email from credentials
			email, err := getEmailFromAuth(auth)
			if err != nil {
				return fmt.Errorf("failed to get email from credentials: %w\nMake sure you're logged in with 'codex auth login --device-auth'", err)
			}

			pm, err := loadProfiles()
			if err != nil {
				return err
			}

			// Check if email already exists
			for name, profile := range pm.Profiles {
				existingEmail, _ := getEmailFromAuth(&profile.Auth)
				if existingEmail == email {
					fmt.Printf("⚠️  Profile already exists: '%s' (%s)\n", name, email)
					fmt.Printf("💡 Tip: Use 'codex-sweet switch %s' to activate it\n", name)
					return nil
				}
			}

			// Use email as profile name
			profileName := email

			// Set all profiles to inactive
			for _, p := range pm.Profiles {
				p.Active = false
			}

			pm.Profiles[profileName] = &Profile{
				Name:      profileName,
				Auth:      *auth,
				CreatedAt: time.Now(),
				Active:    true,
			}
			pm.Current = profileName

			if err := pm.save(); err != nil {
				return err
			}

			fmt.Printf("✓ Saved profile '%s'\n", profileName)
			return nil
		},
	}
}

func cmdSwitch() *cobra.Command {
	return &cobra.Command{
		Use:   "switch [profile-name]",
		Short: "Switch to a different profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := args[0]

			pm, err := loadProfiles()
			if err != nil {
				return err
			}

			profile, exists := pm.Profiles[profileName]
			if !exists {
				return fmt.Errorf("profile '%s' not found", profileName)
			}

			if err := saveCodexAuth(&profile.Auth); err != nil {
				return fmt.Errorf("failed to update ~/.codex/auth.json: %w", err)
			}

			// Update active status
			for _, p := range pm.Profiles {
				p.Active = false
			}
			profile.Active = true
			pm.Current = profileName

			if err := pm.save(); err != nil {
				return err
			}

			fmt.Printf("✓ Switched to profile '%s'\n", profileName)
			return nil
		},
	}
}

func cmdList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all saved profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			pm, err := loadProfiles()
			if err != nil {
				return err
			}

			if len(pm.Profiles) == 0 {
				fmt.Println("No profiles saved yet.")
				return nil
			}

			fmt.Println("\nSaved profiles:")
			fmt.Println("───────────────────────────────────────────────")
			for name, profile := range pm.Profiles {
				marker := " "
				if profile.Active {
					marker = "●"
				}
				fmt.Printf("%s %s (created: %s)\n",
					marker,
					name,
					profile.CreatedAt.Format("2006-01-02 15:04"))
			}
			fmt.Println()
			return nil
		},
	}
}

func cmdInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [profile-name]",
		Short: "Show profile information",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pm, err := loadProfiles()
			if err != nil {
				return err
			}

			var profileName string
			if len(args) > 0 {
				profileName = args[0]
			} else {
				profileName = pm.Current
			}

			if profileName == "" {
				return fmt.Errorf("no active profile, please specify a profile name")
			}

			profile, exists := pm.Profiles[profileName]
			if !exists {
				return fmt.Errorf("profile '%s' not found", profileName)
			}

			fmt.Printf("\n📋 Profile: %s\n", profileName)
			fmt.Println("───────────────────────────────────────────────")
			fmt.Printf("Created:     %s\n", profile.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Active:      %v\n", profile.Active)
			fmt.Printf("Auth Mode:   %s\n", profile.Auth.AuthMode)

			if profile.Auth.OpenAIAPIKey != nil && *profile.Auth.OpenAIAPIKey != "" {
				fmt.Printf("API Key:     %s...%s\n", (*profile.Auth.OpenAIAPIKey)[:7], (*profile.Auth.OpenAIAPIKey)[len(*profile.Auth.OpenAIAPIKey)-4:])
			}

			if profile.Auth.Tokens != nil {
				if profile.Auth.Tokens.AccessToken != "" {
					fmt.Printf("Access Token: %s...%s\n", profile.Auth.Tokens.AccessToken[:20], profile.Auth.Tokens.AccessToken[len(profile.Auth.Tokens.AccessToken)-20:])
				}
				if profile.Auth.Tokens.AccountID != "" {
					fmt.Printf("Account ID:  %s\n", profile.Auth.Tokens.AccountID)
				}
			}

			if profile.Auth.LastRefresh != "" {
				fmt.Printf("Last Refresh: %s\n", profile.Auth.LastRefresh)
			}
			fmt.Println()

			return nil
		},
	}

	return cmd
}

func printProfileUsage(profileName string, profile *Profile) error {
	// Check auth mode and get tokens
	if profile.Auth.AuthMode != "chatgpt" {
		fmt.Printf("⚠️  %s: skipped (auth mode: %s)\n", profileName, profile.Auth.AuthMode)
		return nil
	}

	if profile.Auth.Tokens == nil || profile.Auth.Tokens.AccessToken == "" || profile.Auth.Tokens.AccountID == "" {
		fmt.Printf("⚠️  %s: skipped (missing tokens)\n", profileName)
		return nil
	}

	usage, err := getUsage(profile.Auth.Tokens.AccessToken, profile.Auth.Tokens.AccountID)
	if err != nil {
		fmt.Printf("❌ %s: failed to fetch usage (%v)\n", profileName, err)
		return nil
	}

	fmt.Printf("\n📊 %s - %s (%s)\n", profileName, usage.Email, usage.PlanType)
	fmt.Println("───────────────────────────────────────────────────────────")

	// Primary window (5h)
	if usage.RateLimit.PrimaryWindow != nil {
		leftPercent := 100 - usage.RateLimit.PrimaryWindow.UsedPercent
		bar := drawProgressBar(usage.RateLimit.PrimaryWindow.UsedPercent, 20)
		resetTime := formatResetTime(usage.RateLimit.PrimaryWindow.ResetAt)
		fmt.Printf("5h limit:        %s %3d%% left (resets %s)\n", bar, leftPercent, resetTime)
	}

	// Secondary window (weekly)
	if usage.RateLimit.SecondaryWindow != nil {
		leftPercent := 100 - usage.RateLimit.SecondaryWindow.UsedPercent
		bar := drawProgressBar(usage.RateLimit.SecondaryWindow.UsedPercent, 20)
		resetTime := formatResetTime(usage.RateLimit.SecondaryWindow.ResetAt)
		fmt.Printf("Weekly limit:    %s %3d%% left (resets %s)\n", bar, leftPercent, resetTime)
	}

	fmt.Println()
	return nil
}

func cmdUsage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usage [profile-name]",
		Short: "Check Codex usage for profiles (all profiles if no name specified)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pm, err := loadProfiles()
			if err != nil {
				return err
			}

			if len(pm.Profiles) == 0 {
				fmt.Println("No profiles saved yet.")
				return nil
			}

			// If profile name specified, show only that one
			if len(args) > 0 {
				profileName := args[0]
				profile, exists := pm.Profiles[profileName]
				if !exists {
					return fmt.Errorf("profile '%s' not found", profileName)
				}
				return printProfileUsage(profileName, profile)
			}

			// Otherwise, show all profiles
			for name, profile := range pm.Profiles {
				printProfileUsage(name, profile)
			}

			return nil
		},
	}

	return cmd
}

func cmdDelete() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [profile-name]",
		Short: "Delete a saved profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := args[0]

			pm, err := loadProfiles()
			if err != nil {
				return err
			}

			if _, exists := pm.Profiles[profileName]; !exists {
				return fmt.Errorf("profile '%s' not found", profileName)
			}

			delete(pm.Profiles, profileName)

			if pm.Current == profileName {
				pm.Current = ""
			}

			if err := pm.save(); err != nil {
				return err
			}

			fmt.Printf("✓ Deleted profile '%s'\n", profileName)
			return nil
		},
	}
}

func cmdAvailable() *cobra.Command {
	return &cobra.Command{
		Use:   "available",
		Short: "Show profiles with available usage limits",
		RunE: func(cmd *cobra.Command, args []string) error {
			pm, err := loadProfiles()
			if err != nil {
				return err
			}

			if len(pm.Profiles) == 0 {
				fmt.Println("No profiles saved yet.")
				return nil
			}

			fmt.Println("\n🔍 Checking available profiles...")
			fmt.Println()

			available := []string{}
			for name, profile := range pm.Profiles {
				if profile.Auth.AuthMode != "chatgpt" {
					continue
				}
				if profile.Auth.Tokens == nil || profile.Auth.Tokens.AccessToken == "" {
					continue
				}

				usage, err := getUsage(profile.Auth.Tokens.AccessToken, profile.Auth.Tokens.AccountID)
				if err != nil {
					continue
				}

				// Check if has available limits (less than 80% used)
				hasAvailable := false
				if usage.RateLimit.PrimaryWindow != nil && usage.RateLimit.PrimaryWindow.UsedPercent < 80 {
					hasAvailable = true
				}
				if usage.RateLimit.SecondaryWindow != nil && usage.RateLimit.SecondaryWindow.UsedPercent < 80 {
					hasAvailable = true
				}

				if hasAvailable && !usage.RateLimit.LimitReached {
					available = append(available, name)

					primary := 100 - usage.RateLimit.PrimaryWindow.UsedPercent
					weekly := 100
					if usage.RateLimit.SecondaryWindow != nil {
						weekly = 100 - usage.RateLimit.SecondaryWindow.UsedPercent
					}

					marker := " "
					if profile.Active {
						marker = "●"
					}

					fmt.Printf("%s %s - 5h: %d%% left, Weekly: %d%% left\n",
						marker, name, primary, weekly)
				}
			}

			if len(available) == 0 {
				fmt.Println("⚠️  No profiles with available limits found.")
				fmt.Println("💡 Tip: Wait for limits to reset or add more profiles")
			} else {
				fmt.Printf("\n✓ Found %d profile(s) with available limits\n", len(available))
			}

			fmt.Println()
			return nil
		},
	}
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "codex-sweet",
		Short: "Manage multiple Codex authentication profiles",
		Long:  "A CLI tool to save, switch, and manage multiple Codex authentication profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default action: show available profiles
			return cmdAvailable().RunE(cmd, args)
		},
	}

	rootCmd.AddCommand(
		cmdSave(),
		cmdSwitch(),
		cmdList(),
		cmdInfo(),
		cmdUsage(),
		cmdAvailable(),
		cmdDelete(),
		cmdAuto(),
		cmdNotify(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
