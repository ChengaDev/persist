package cmd

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/ChengaDev/persist/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage psst configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print current configuration",
	Args:  cobra.NoArgs,
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Update a configuration value",
	Long: `Available keys:

  session_timeout       int   Minutes to cache the master password after first
                              unlock (0 = disabled, prompt every time)
  clipboard_clear_after int   Seconds before the clipboard is wiped after
                              psst get (0 = never)
  default_secure        bool  Make --secure the default on every psst save`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfigShow(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tVALUE\tDESCRIPTION")
	fmt.Fprintln(w, "---\t-----\t-----------")
	fmt.Fprintf(w, "session_timeout\t%d\tMinutes to cache password (0 = disabled)\n", cfg.SessionTimeout)
	fmt.Fprintf(w, "clipboard_clear_after\t%d\tSeconds before clipboard is wiped (0 = never)\n", cfg.ClipboardClearAfter)
	fmt.Fprintf(w, "default_secure\t%v\tMake --secure the default on psst save\n", cfg.DefaultSecure)
	return w.Flush()
}

func runConfigSet(_ *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch key {
	case "session_timeout":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("session_timeout must be a non-negative integer (minutes)")
		}
		wasDisabled := cfg.SessionTimeout == 0
		cfg.SessionTimeout = n
		if n > 0 && wasDisabled {
			printSessionDisclaimer()
		}

	case "clipboard_clear_after":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("clipboard_clear_after must be a non-negative integer (seconds)")
		}
		cfg.ClipboardClearAfter = n

	case "default_secure":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("default_secure must be true or false")
		}
		cfg.DefaultSecure = b

	default:
		return fmt.Errorf("unknown config key %q — run `psst config show` to see available keys", key)
	}

	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Set %s = %s\n", key, value)
	return nil
}

func printSessionDisclaimer() {
	fmt.Fprintln(os.Stderr, "Security notice — password session enabled:")
	fmt.Fprintln(os.Stderr, "  The master password will be cached on disk at ~/.persist/.session")
	fmt.Fprintln(os.Stderr, "  (chmod 600, automatically deleted when the session expires).")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  Risk: the file is readable by root. If your disk is not encrypted,")
	fmt.Fprintln(os.Stderr, "  anyone with physical access could read it.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  Recommendation: enable full-disk encryption (FileVault on macOS)")
	fmt.Fprintln(os.Stderr, "  before using this feature.")
}
