package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
	"github.com/spf13/cobra"
)

const notifyConfigFile = "notify.json"

type NotifyConfig struct {
	URLs   []string     `json:"urls"`
	Events NotifyEvents `json:"events"`
}

type NotifyEvents struct {
	QuotaLow     bool `json:"quota_low"`
	LimitReached bool `json:"limit_reached"`
	AutoSwitch   bool `json:"auto_switch"`
	AllExhausted bool `json:"all_exhausted"`
}

type Notifier struct {
	config *NotifyConfig
	sender *router.ServiceRouter
}

func getNotifyConfigPath() string {
	return filepath.Join(getConfigPath(), notifyConfigFile)
}

func loadNotifyConfig() (*NotifyConfig, error) {
	cfg := &NotifyConfig{
		URLs: []string{},
		Events: NotifyEvents{
			QuotaLow:     true,
			LimitReached: true,
			AutoSwitch:   true,
			AllExhausted: true,
		},
	}

	data, err := os.ReadFile(getNotifyConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func saveNotifyConfig(cfg *NotifyConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getNotifyConfigPath(), data, 0600)
}

func NewNotifier() *Notifier {
	cfg, err := loadNotifyConfig()
	if err != nil {
		return &Notifier{}
	}

	if len(cfg.URLs) == 0 {
		return &Notifier{config: cfg}
	}

	sender, err := shoutrrr.CreateSender(cfg.URLs...)
	if err != nil {
		fmt.Printf("Warning: failed to create notifier: %v\n", err)
		return &Notifier{config: cfg}
	}

	return &Notifier{config: cfg, sender: sender}
}

func (n *Notifier) Send(msg string) {
	if n.sender == nil {
		return
	}
	n.sender.Send(msg, nil)
}

func (n *Notifier) NotifyQuotaLow(email string, primaryPercent, secondaryPercent int) {
	if n.config == nil || !n.config.Events.QuotaLow {
		return
	}
	n.Send(fmt.Sprintf("[codex-sweet] Quota low: %s (5h: %d%%, weekly: %d%%)", email, primaryPercent, secondaryPercent))
}

func (n *Notifier) NotifyLimitReached(email string) {
	if n.config == nil || !n.config.Events.LimitReached {
		return
	}
	n.Send(fmt.Sprintf("[codex-sweet] Rate limit reached: %s", email))
}

func (n *Notifier) NotifyAutoSwitch(fromEmail, toEmail string, primaryPercent, secondaryPercent int) {
	if n.config == nil || !n.config.Events.AutoSwitch {
		return
	}
	n.Send(fmt.Sprintf("[codex-sweet] Switched: %s -> %s (5h: %d%%, weekly: %d%%)", fromEmail, toEmail, primaryPercent, secondaryPercent))
}

func (n *Notifier) NotifyAllExhausted() {
	if n.config == nil || !n.config.Events.AllExhausted {
		return
	}
	n.Send("[codex-sweet] All profiles exhausted! No available quota remaining.")
}

func cmdNotify() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notify",
		Short: "Manage notification settings",
	}

	cmd.AddCommand(cmdNotifyAdd())
	cmd.AddCommand(cmdNotifyRemove())
	cmd.AddCommand(cmdNotifyList())
	cmd.AddCommand(cmdNotifyTest())

	return cmd
}

func cmdNotifyAdd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [url]",
		Short: "Add a notification URL (e.g. telegram://bot:token@telegram?chats=123)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ensureConfigDir(); err != nil {
				return err
			}

			cfg, err := loadNotifyConfig()
			if err != nil {
				return err
			}

			// Validate URL by trying to create a sender
			_, err = shoutrrr.CreateSender(args[0])
			if err != nil {
				return fmt.Errorf("invalid notification URL: %w", err)
			}

			// Check duplicate
			for _, u := range cfg.URLs {
				if u == args[0] {
					fmt.Println("URL already exists")
					return nil
				}
			}

			cfg.URLs = append(cfg.URLs, args[0])
			if err := saveNotifyConfig(cfg); err != nil {
				return err
			}

			fmt.Printf("Added notification URL (%d total)\n", len(cfg.URLs))
			return nil
		},
	}
}

func cmdNotifyRemove() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [index]",
		Short: "Remove a notification URL by index (0-based)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadNotifyConfig()
			if err != nil {
				return err
			}

			var idx int
			if _, err := fmt.Sscanf(args[0], "%d", &idx); err != nil {
				return fmt.Errorf("invalid index: %s", args[0])
			}

			if idx < 0 || idx >= len(cfg.URLs) {
				return fmt.Errorf("index %d out of range (0-%d)", idx, len(cfg.URLs)-1)
			}

			cfg.URLs = append(cfg.URLs[:idx], cfg.URLs[idx+1:]...)
			if err := saveNotifyConfig(cfg); err != nil {
				return err
			}

			fmt.Printf("Removed notification URL (%d remaining)\n", len(cfg.URLs))
			return nil
		},
	}
}

func cmdNotifyList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured notification URLs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadNotifyConfig()
			if err != nil {
				return err
			}

			if len(cfg.URLs) == 0 {
				fmt.Println("No notification URLs configured.")
				fmt.Println("Add one with: codex-sweet notify add <url>")
				return nil
			}

			fmt.Println("\nNotification URLs:")
			fmt.Println("───────────────────────────────────────────────")
			for i, u := range cfg.URLs {
				fmt.Printf("  [%d] %s\n", i, u)
			}

			fmt.Printf("\nEvents:\n")
			fmt.Printf("  quota_low:     %v\n", cfg.Events.QuotaLow)
			fmt.Printf("  limit_reached: %v\n", cfg.Events.LimitReached)
			fmt.Printf("  auto_switch:   %v\n", cfg.Events.AutoSwitch)
			fmt.Printf("  all_exhausted: %v\n", cfg.Events.AllExhausted)
			fmt.Println()

			return nil
		},
	}
}

func cmdNotifyTest() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Send a test notification",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadNotifyConfig()
			if err != nil {
				return err
			}

			if len(cfg.URLs) == 0 {
				return fmt.Errorf("no notification URLs configured")
			}

			sender, err := shoutrrr.CreateSender(cfg.URLs...)
			if err != nil {
				return fmt.Errorf("failed to create sender: %w", err)
			}

			fmt.Println("Sending test notification...")
			errs := sender.Send("[codex-sweet] Test notification - notifications are working!", nil)
			for _, e := range errs {
				if e != nil {
					fmt.Printf("Error: %v\n", e)
				}
			}

			fmt.Println("Done!")
			return nil
		},
	}
}
