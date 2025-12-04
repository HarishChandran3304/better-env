package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

type RunCommand struct {
	command []string
}

func NewRunCommand(command []string) *RunCommand {
	return &RunCommand{
		command: command,
	}
}

func (r *RunCommand) Run() error {
	if len(r.command) == 0 {
		return fmt.Errorf("no command specified")
	}

	env, err := ParseConfig(); 
	if err != nil {
		return fmt.Errorf("failed to parse .better-env: %w", err)
	}

	// 7. Execute command with secrets in environment
	cmd := exec.Command(r.command[0], r.command[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}

var runCmd = &cobra.Command{
	Use:   "run COMMAND [ARGS...]",
	Short: "Run a command with secrets in its environment",
	Long:  "Execute a command with decrypted secrets available as environment variables. The secrets are only available to the child process and never exported to the parent shell.",
	Example: `  bnv run node server.js  
  bnv run python3 main.py`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rc := NewRunCommand(args)
		return rc.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
