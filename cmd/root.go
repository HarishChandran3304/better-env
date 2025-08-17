package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bnv",
	Short: "better-env: secuure global env var manager",
	Long:  "Better-env securely stores env vars globally and loads them without plaintext on disk.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}
