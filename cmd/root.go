package cmd

import (
	"fmt"
	"os"

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
	if err := rootCmd.Execute(); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("bnv version {{.Version}}\n")
}
