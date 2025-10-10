package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bnv",
	Short: "better-env: encrypted secrets, zero plaintext, instant runtime loading.",
	Long:  "Better-env: a global, GPG-encrypted store for your environment variables. Load them directly at runtime — never touch plaintext again.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}

func init() {
	// wire subcommands here
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(storeCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(copyCmd)
	rootCmd.AddCommand(loadCmd)
	rootCmd.AddCommand(unloadCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(updateCmd)
}
