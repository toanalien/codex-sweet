package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testEmail = "user@example.com"

func TestCmdSaveDuplicateProfileExistingUsableNoOverwrite(t *testing.T) {
	setupTempHome(t)
	writeCodexAuth(t, authWithToken(testEmail, "fresh-token", "fresh-account"))
	writeProfiles(t, &ProfileManager{
		Profiles: map[string]*Profile{
			testEmail: profileWithAuth(testEmail, authWithToken(testEmail, "old-usable-token", "old-account")),
		},
		Current: testEmail,
	})
	stubUsageChecker(t, map[string]usageCheckResult{
		"fresh-token":      {usage: usageForEmail(testEmail)},
		"old-usable-token": {usage: usageForEmail(testEmail)},
	})

	if err := runSaveCommand(t); err != nil {
		t.Fatalf("save returned error: %v", err)
	}

	got := readProfile(t, testEmail)
	if got.Auth.Tokens.AccessToken != "old-usable-token" {
		t.Fatalf("expected existing usable profile not to be overwritten, got access token %q", got.Auth.Tokens.AccessToken)
	}
	if !got.Active {
		t.Fatalf("expected existing profile to remain active")
	}
}

func TestCmdSaveDuplicateProfileExistingUnusableOverwritten(t *testing.T) {
	tests := []struct {
		name         string
		existingAuth CodexAuth
		usageResults map[string]usageCheckResult
	}{
		{
			name:         "existing 401",
			existingAuth: authWithToken(testEmail, "old-401-token", "old-account"),
			usageResults: map[string]usageCheckResult{
				"old-401-token": {err: &UsageAPIError{StatusCode: http.StatusUnauthorized, Body: "unauthorized"}},
				"fresh-token":   {usage: usageForEmail(testEmail)},
			},
		},
		{
			name:         "existing 403",
			existingAuth: authWithToken(testEmail, "old-403-token", "old-account"),
			usageResults: map[string]usageCheckResult{
				"old-403-token": {err: &UsageAPIError{StatusCode: http.StatusForbidden, Body: "forbidden"}},
				"fresh-token":   {usage: usageForEmail(testEmail)},
			},
		},
		{
			name:         "existing missing token",
			existingAuth: authMissingAccessToken(testEmail),
			usageResults: map[string]usageCheckResult{
				"fresh-token": {usage: usageForEmail(testEmail)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTempHome(t)
			writeCodexAuth(t, authWithToken(testEmail, "fresh-token", "fresh-account"))
			writeProfiles(t, &ProfileManager{
				Profiles: map[string]*Profile{
					testEmail: profileWithAuth(testEmail, tt.existingAuth),
				},
				Current: testEmail,
			})
			stubUsageChecker(t, tt.usageResults)

			if err := runSaveCommand(t); err != nil {
				t.Fatalf("save returned error: %v", err)
			}

			got := readProfile(t, testEmail)
			if got.Auth.Tokens == nil || got.Auth.Tokens.AccessToken != "fresh-token" {
				t.Fatalf("expected unusable existing profile to be overwritten with fresh token, got %#v", got.Auth.Tokens)
			}
			if !got.Active {
				t.Fatalf("expected overwritten profile to be active")
			}
		})
	}
}

func TestCmdSaveDuplicateProfileCurrentAuthInvalidOrTemporaryErrorNoOverwrite(t *testing.T) {
	tests := []struct {
		name        string
		currentAuth CodexAuth
		currentErr  error
		wantErrText string
	}{
		{
			name:        "current 401",
			currentAuth: authWithToken(testEmail, "fresh-401-token", "fresh-account"),
			currentErr:  &UsageAPIError{StatusCode: http.StatusUnauthorized, Body: "unauthorized"},
			wantErrText: "current Codex credentials are also unusable",
		},
		{
			name:        "current 403",
			currentAuth: authWithToken(testEmail, "fresh-403-token", "fresh-account"),
			currentErr:  &UsageAPIError{StatusCode: http.StatusForbidden, Body: "forbidden"},
			wantErrText: "current Codex credentials are also unusable",
		},
		{
			name:        "current server error",
			currentAuth: authWithToken(testEmail, "fresh-500-token", "fresh-account"),
			currentErr:  &UsageAPIError{StatusCode: http.StatusInternalServerError, Body: "server error"},
			wantErrText: "current Codex credentials could not be verified",
		},
		{
			name:        "current network error",
			currentAuth: authWithToken(testEmail, "fresh-network-token", "fresh-account"),
			currentErr:  errors.New("connection reset by peer"),
			wantErrText: "current Codex credentials could not be verified",
		},
		{
			name:        "current missing access token",
			currentAuth: authMissingAccessToken(testEmail),
			wantErrText: "current Codex credentials are also unusable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTempHome(t)
			writeCodexAuth(t, tt.currentAuth)
			writeProfiles(t, &ProfileManager{
				Profiles: map[string]*Profile{
					testEmail: profileWithAuth(testEmail, authMissingAccessToken(testEmail)),
				},
				Current: testEmail,
			})

			usageResults := map[string]usageCheckResult{}
			if tt.currentAuth.Tokens != nil && tt.currentAuth.Tokens.AccessToken != "" {
				usageResults[tt.currentAuth.Tokens.AccessToken] = usageCheckResult{err: tt.currentErr}
			}
			stubUsageChecker(t, usageResults)

			err := runSaveCommand(t)
			if err == nil {
				t.Fatalf("expected save to fail")
			}
			if !strings.Contains(err.Error(), tt.wantErrText) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErrText, err.Error())
			}

			got := readProfile(t, testEmail)
			if got.Auth.Tokens != nil && got.Auth.Tokens.AccessToken != "" {
				t.Fatalf("expected invalid/unverified current auth not to overwrite existing profile, got token %q", got.Auth.Tokens.AccessToken)
			}
		})
	}
}

func TestCmdSaveDuplicateProfileExistingTemporaryErrorNoOverwrite(t *testing.T) {
	tests := []struct {
		name        string
		existingErr error
	}{
		{
			name:        "existing server error",
			existingErr: &UsageAPIError{StatusCode: http.StatusInternalServerError, Body: "server error"},
		},
		{
			name:        "existing network error",
			existingErr: errors.New("temporary dial failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTempHome(t)
			writeCodexAuth(t, authWithToken(testEmail, "fresh-token", "fresh-account"))
			writeProfiles(t, &ProfileManager{
				Profiles: map[string]*Profile{
					testEmail: profileWithAuth(testEmail, authWithToken(testEmail, "old-temporary-token", "old-account")),
				},
				Current: testEmail,
			})
			stubUsageChecker(t, map[string]usageCheckResult{
				"fresh-token":         {usage: usageForEmail(testEmail)},
				"old-temporary-token": {err: tt.existingErr},
			})

			if err := runSaveCommand(t); err != nil {
				t.Fatalf("save returned error: %v", err)
			}

			got := readProfile(t, testEmail)
			if got.Auth.Tokens.AccessToken != "old-temporary-token" {
				t.Fatalf("expected temporary existing auth error not to allow overwrite, got token %q", got.Auth.Tokens.AccessToken)
			}
		})
	}
}

type usageCheckResult struct {
	usage *UsageResponse
	err   error
}

func stubUsageChecker(t *testing.T, results map[string]usageCheckResult) {
	t.Helper()
	previous := usageChecker
	usageChecker = func(accessToken, accountID string) (*UsageResponse, error) {
		if result, ok := results[accessToken]; ok {
			return result.usage, result.err
		}
		t.Fatalf("unexpected usage check for token %q and account %q", accessToken, accountID)
		return nil, nil
	}
	t.Cleanup(func() { usageChecker = previous })
}

func setupTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func runSaveCommand(t *testing.T) error {
	t.Helper()

	oldStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = writePipe
	defer func() { os.Stdout = oldStdout }()

	cmd := cmdSave()
	runErr := cmd.RunE(cmd, nil)

	if err := writePipe.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}
	_, _ = io.ReadAll(readPipe)
	if err := readPipe.Close(); err != nil {
		t.Fatalf("close stdout reader: %v", err)
	}

	return runErr
}

func writeCodexAuth(t *testing.T, auth CodexAuth) {
	t.Helper()
	path := getCodexAuthPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("create codex auth dir: %v", err)
	}
	writeJSONFile(t, path, &auth)
}

func writeProfiles(t *testing.T, pm *ProfileManager) {
	t.Helper()
	if err := ensureConfigDir(); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	writeJSONFile(t, getProfilesPath(), pm)
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readProfile(t *testing.T, name string) *Profile {
	t.Helper()
	pm, err := loadProfiles()
	if err != nil {
		t.Fatalf("load profiles: %v", err)
	}
	profile := pm.Profiles[name]
	if profile == nil {
		t.Fatalf("profile %q not found in %#v", name, pm.Profiles)
	}
	return profile
}

func profileWithAuth(name string, auth CodexAuth) *Profile {
	return &Profile{
		Name:      name,
		Auth:      auth,
		CreatedAt: time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC),
		Active:    true,
	}
}

func authWithToken(email, accessToken, accountID string) CodexAuth {
	return CodexAuth{
		AuthMode: "chatgpt",
		Tokens: &TokenInfo{
			IDToken:     jwtWithEmail(email),
			AccessToken: accessToken,
			AccountID:   accountID,
		},
	}
}

func authMissingAccessToken(email string) CodexAuth {
	return CodexAuth{
		AuthMode: "chatgpt",
		Tokens: &TokenInfo{
			IDToken:   jwtWithEmail(email),
			AccountID: "stale-account",
		},
	}
}

func usageForEmail(email string) *UsageResponse {
	return &UsageResponse{
		Email:    email,
		PlanType: "plus",
		RateLimit: RateLimit{
			Allowed: true,
		},
	}
}

func jwtWithEmail(email string) string {
	encode := func(value any) string {
		data, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		return base64.RawURLEncoding.EncodeToString(data)
	}
	return encode(map[string]string{"alg": "none", "typ": "JWT"}) + "." + encode(map[string]string{"email": email}) + ".signature"
}
