package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"redterm/internal/config"
	redcontext "redterm/internal/context"
	"redterm/internal/llm"
	"redterm/internal/terminal"
)

var (
	cfgFile string
	engFile string
)

var rootCmd = &cobra.Command{
	Use:   "redterm",
	Short: "Red team intelligent terminal",
	Long: `redterm — transparent terminal with embedded AI for red team operations.

Press Ctrl+G (configurable) to open the command bar, then type:
  /suggest          recommend the next command
  /attack           brainstorm attack paths
  /sitrep           situational awareness summary
  /engage           set or update engagement context
  /context <file>   inject a file into context
  /prompt <text>    manually add context text
  /clear            clear the context buffer
  /help             show command reference

Provider is selected via config or REDTERM_PROVIDER env var (openai|anthropic|ollama).`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}

		provider, err := llm.New(cfg)
		if err != nil {
			return fmt.Errorf("provider: %w", err)
		}

		buf := redcontext.New(cfg.ContextLines)

		sc, err := redcontext.NewSharedContext(buf)
		if err != nil {
			return fmt.Errorf("shared context: %w", err)
		}
		defer sc.Close()

		var eng *config.Engagement
		if engFile != "" {
			eng, err = config.LoadEngagement(engFile)
			if err != nil {
				return fmt.Errorf("engagement file: %w", err)
			}
		}

		return terminal.Run(cfg.Shell, cfg.TriggerByte(), provider, buf, sc, eng)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/redterm/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&engFile, "engagement", "", "engagement context YAML file")
}
