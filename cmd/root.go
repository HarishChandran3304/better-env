package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bnv",
	Short: "better-env: encrypted secrets, zero plaintext, instant runtime loading.",
	Long:  "better-env: a global, GPG-encrypted store for your environment variables. Load them directly at runtime — never touch plaintext again.",
}

// version is set at build time with -ldflags "-X 'github.com/HarishChandran3304/better-env/cmd.version=vX.Y.Z'"
var version = "dev"

func Execute() {
	// Silence Cobra's default usage and error prints; we handle output ourselves
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	// Handle SIGINT/SIGTERM gracefully for all commands
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigc
		// Print a newline for clean prompt return and exit with 130 (terminated by Ctrl+C)
		fmt.Fprintln(os.Stderr)
		os.Exit(130)
	}()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("bnv version {{.Version}}\n")
}
