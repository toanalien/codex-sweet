package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const (
	checkInterval  = 10 * time.Minute
	quotaThreshold = 20 // Switch when < 20% remaining
	cacheExpiry    = 5 * time.Minute
	logFile        = "log.json"
	stateFile      = "state.json"
)

type QuotaCache struct {
	Email            string    `json:"email"`
	PrimaryPercent   int       `json:"primary_percent"`
	SecondaryPercent int       `json:"secondary_percent"`
	LastChecked      time.Time `json:"last_checked"`
	Available        bool      `json:"available"`
}

type AutoSwitchLog struct {
	Timestamp   time.Time `json:"timestamp"`
	FromProfile string    `json:"from_profile"`
	ToProfile   string    `json:"to_profile"`
	Reason      string    `json:"reason"`
	FromQuota   string    `json:"from_quota"`
	ToQuota     string    `json:"to_quota"`
}

type DaemonState struct {
	QuotaCache    map[string]*QuotaCache `json:"quota_cache"`
	SwitchHistory []AutoSwitchLog        `json:"switch_history"`
	LastCheckTime time.Time              `json:"last_check_time"`
}

func getLogPath() string {
	return filepath.Join(getConfigPath(), logFile)
}

func getStatePath() string {
	return filepath.Join(getConfigPath(), stateFile)
}

func loadDaemonState() (*DaemonState, error) {
	state := &DaemonState{
		QuotaCache:    make(map[string]*QuotaCache),
		SwitchHistory: []AutoSwitchLog{},
	}

	data, err := os.ReadFile(getStatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, state); err != nil {
		return nil, err
	}

	return state, nil
}

func (ds *DaemonState) save() error {
	data, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getStatePath(), data, 0600)
}

func (ds *DaemonState) logSwitch(from, to, reason, fromQuota, toQuota string) error {
	log := AutoSwitchLog{
		Timestamp:   time.Now(),
		FromProfile: from,
		ToProfile:   to,
		Reason:      reason,
		FromQuota:   fromQuota,
		ToQuota:     toQuota,
	}

	ds.SwitchHistory = append(ds.SwitchHistory, log)

	// Keep only last 100 logs
	if len(ds.SwitchHistory) > 100 {
		ds.SwitchHistory = ds.SwitchHistory[len(ds.SwitchHistory)-100:]
	}

	// Save to log file
	logData, err := json.MarshalIndent(ds.SwitchHistory, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(getLogPath(), logData, 0600)
}

func checkProfileQuota(email string, profile *Profile, state *DaemonState) (*QuotaCache, error) {
	// Check cache first
	if cached, exists := state.QuotaCache[email]; exists {
		if time.Since(cached.LastChecked) < cacheExpiry {
			return cached, nil
		}
	}

	// Check auth mode
	if profile.Auth.AuthMode != "chatgpt" {
		return nil, fmt.Errorf("profile %s: not chatgpt auth mode", email)
	}

	if profile.Auth.Tokens == nil || profile.Auth.Tokens.AccessToken == "" {
		return nil, fmt.Errorf("profile %s: missing tokens", email)
	}

	// Fetch usage
	usage, err := getUsage(profile.Auth.Tokens.AccessToken, profile.Auth.Tokens.AccountID)
	if err != nil {
		return nil, fmt.Errorf("profile %s: failed to fetch usage: %w", email, err)
	}

	primaryLeft := 100
	if usage.RateLimit.PrimaryWindow != nil {
		primaryLeft = 100 - usage.RateLimit.PrimaryWindow.UsedPercent
	}

	secondaryLeft := 100
	if usage.RateLimit.SecondaryWindow != nil {
		secondaryLeft = 100 - usage.RateLimit.SecondaryWindow.UsedPercent
	}

	cache := &QuotaCache{
		Email:            email,
		PrimaryPercent:   primaryLeft,
		SecondaryPercent: secondaryLeft,
		LastChecked:      time.Now(),
		Available:        primaryLeft > quotaThreshold && secondaryLeft > quotaThreshold && !usage.RateLimit.LimitReached,
	}

	state.QuotaCache[email] = cache
	return cache, nil
}

func selectBestProfile(pm *ProfileManager, state *DaemonState) (string, *QuotaCache, error) {
	var candidates []string
	for email := range pm.Profiles {
		candidates = append(candidates, email)
	}

	// Shuffle to randomize check order
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	var bestEmail string
	var bestCache *QuotaCache
	bestScore := -1

	for _, email := range candidates {
		profile := pm.Profiles[email]

		cache, err := checkProfileQuota(email, profile, state)
		if err != nil {
			continue
		}

		if !cache.Available {
			continue
		}

		// Score: prioritize higher quota
		score := cache.PrimaryPercent + cache.SecondaryPercent
		if score > bestScore {
			bestScore = score
			bestEmail = email
			bestCache = cache
		}
	}

	if bestEmail == "" {
		return "", nil, fmt.Errorf("no available profiles found")
	}

	return bestEmail, bestCache, nil
}

func runAutoDaemon() error {
	fmt.Println("🤖 Starting auto-switch daemon...")
	fmt.Printf("⏰ Check interval: %v\n", checkInterval)
	fmt.Printf("📊 Quota threshold: %d%%\n\n", quotaThreshold)

	// Load state
	state, err := loadDaemonState()
	if err != nil {
		return fmt.Errorf("failed to load daemon state: %w", err)
	}

	// Initialize notifier
	notifier := NewNotifier()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Run initial check
	if err := checkAndSwitch(state, notifier); err != nil {
		fmt.Printf("⚠️  Initial check failed: %v\n", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := checkAndSwitch(state, notifier); err != nil {
				fmt.Printf("⚠️  Check failed: %v\n", err)
			}

		case sig := <-sigChan:
			fmt.Printf("\n📴 Received signal %v, shutting down...\n", sig)
			if err := state.save(); err != nil {
				fmt.Printf("⚠️  Failed to save state: %v\n", err)
			}
			return nil
		}
	}
}

func checkAndSwitch(state *DaemonState, notifier *Notifier) error {
	fmt.Printf("[%s] 🔍 Checking profiles...\n", time.Now().Format("15:04:05"))

	pm, err := loadProfiles()
	if err != nil {
		return fmt.Errorf("failed to load profiles: %w", err)
	}

	if len(pm.Profiles) == 0 {
		return fmt.Errorf("no profiles available")
	}

	// Check current profile
	currentEmail := pm.Current
	if currentEmail == "" {
		// No current profile, select best one
		bestEmail, bestCache, err := selectBestProfile(pm, state)
		if err != nil {
			notifier.NotifyAllExhausted()
			return err
		}

		if err := switchToProfile(pm, bestEmail); err != nil {
			return fmt.Errorf("failed to switch to %s: %w", bestEmail, err)
		}

		fmt.Printf("✅ Switched to: %s (5h: %d%%, Weekly: %d%%)\n",
			bestEmail, bestCache.PrimaryPercent, bestCache.SecondaryPercent)

		notifier.NotifyAutoSwitch("(none)", bestEmail, bestCache.PrimaryPercent, bestCache.SecondaryPercent)
		return state.logSwitch("", bestEmail, "initial", "", formatQuota(bestCache))
	}

	currentProfile := pm.Profiles[currentEmail]
	if currentProfile == nil {
		return fmt.Errorf("current profile %s not found", currentEmail)
	}

	// Check current profile quota
	currentCache, err := checkProfileQuota(currentEmail, currentProfile, state)
	if err != nil {
		fmt.Printf("⚠️  Failed to check current profile: %v\n", err)
		// Try to switch anyway
		currentCache = &QuotaCache{Email: currentEmail, Available: false}
	}

	fmt.Printf("📊 Current: %s (5h: %d%%, Weekly: %d%%)\n",
		currentEmail, currentCache.PrimaryPercent, currentCache.SecondaryPercent)

	// Check if need to switch
	if currentCache.Available {
		fmt.Println("✓ Current profile has sufficient quota")
		state.LastCheckTime = time.Now()
		return state.save()
	}

	// Notify quota low
	notifier.NotifyQuotaLow(currentEmail, currentCache.PrimaryPercent, currentCache.SecondaryPercent)

	// Need to switch - find best alternative
	fmt.Println("⚠️  Current profile quota low, finding alternative...")

	bestEmail, bestCache, err := selectBestProfile(pm, state)
	if err != nil {
		notifier.NotifyAllExhausted()
		return err
	}

	if bestEmail == currentEmail {
		fmt.Println("⚠️  No better alternative found, staying on current profile")
		return nil
	}

	// Perform switch
	if err := switchToProfile(pm, bestEmail); err != nil {
		return fmt.Errorf("failed to switch to %s: %w", bestEmail, err)
	}

	reason := fmt.Sprintf("quota low: 5h=%d%%, weekly=%d%%", currentCache.PrimaryPercent, currentCache.SecondaryPercent)

	fmt.Printf("🔄 Switched: %s → %s (5h: %d%%, Weekly: %d%%)\n",
		currentEmail, bestEmail, bestCache.PrimaryPercent, bestCache.SecondaryPercent)

	notifier.NotifyAutoSwitch(currentEmail, bestEmail, bestCache.PrimaryPercent, bestCache.SecondaryPercent)

	state.LastCheckTime = time.Now()
	if err := state.logSwitch(currentEmail, bestEmail, reason, formatQuota(currentCache), formatQuota(bestCache)); err != nil {
		fmt.Printf("⚠️  Failed to log switch: %v\n", err)
	}

	return state.save()
}

func switchToProfile(pm *ProfileManager, email string) error {
	profile, exists := pm.Profiles[email]
	if !exists {
		return fmt.Errorf("profile %s not found", email)
	}

	if err := saveCodexAuth(&profile.Auth); err != nil {
		return err
	}

	// Update active status
	for _, p := range pm.Profiles {
		p.Active = false
	}
	profile.Active = true
	pm.Current = email

	return pm.save()
}

func formatQuota(cache *QuotaCache) string {
	if cache == nil {
		return "unknown"
	}
	return fmt.Sprintf("5h:%d%% weekly:%d%%", cache.PrimaryPercent, cache.SecondaryPercent)
}

func cmdAuto() *cobra.Command {
	return &cobra.Command{
		Use:   "auto",
		Short: "Auto-switch between profiles based on quota (daemon mode)",
		Long: `Run auto-switch daemon that:
- Checks quota every 10 minutes
- Switches to profile with highest quota when current is low
- Caches quota to minimize API calls
- Logs all switches to ~/.codex-sweet/log.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ensureConfigDir(); err != nil {
				return err
			}
			return runAutoDaemon()
		},
	}
}
