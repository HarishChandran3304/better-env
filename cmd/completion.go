package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

func init() {
    completionCmd := &cobra.Command{
        Use:   "completion [bash|zsh|fish|powershell]",
        Short: "Generate shell completion scripts",
        Long:  "Generate shell completion scripts for bash, zsh, fish, or PowerShell.",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            shell := args[0]
            switch shell {
            case "bash":
                return rootCmd.GenBashCompletion(os.Stdout)
            case "zsh":
                return rootCmd.GenZshCompletion(os.Stdout)
            case "fish":
                return rootCmd.GenFishCompletion(os.Stdout, true)
            case "powershell":
                return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
            default:
                return fmt.Errorf("unsupported shell: %s (use bash|zsh|fish|powershell)", shell)
            }
        },
    }

    rootCmd.AddCommand(completionCmd)
}


