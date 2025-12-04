package cmd

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"

    "github.com/spf13/cobra"
)

type ShellCommand struct{}

func NewShellCommand() *ShellCommand { return &ShellCommand{} }

func (s *ShellCommand) Run() error {
  	env, err := ParseConfig()
		if err != nil {
			return fmt.Errorf("failed to parse .better-env: %w", err)
		}

    // Mark that we're in a better-env subshell (useful for prompts/scripts)
    env = append(env, "BNV_ACTIVE=1")
    // Fully overridden prompt format
    promptPrefix := "\n(bnv) ❯ "

    // 7. Resolve shell to exec (POSIX + Windows fallback)
    shell := ""
    args := []string{}

    if runtime.GOOS == "windows" {
        if p, _ := exec.LookPath("pwsh.exe"); p != "" {
            shell = p
            // Override prompt function for the session
            pwshCmd := "function global:prompt { \"" + promptPrefix + "\" }"
            args = []string{"-NoExit", "-Command", pwshCmd}
        } else if p, _ := exec.LookPath("powershell.exe"); p != "" {
            shell = p
            psCmd := "function global:prompt { \"" + promptPrefix + "\" }"
            args = []string{"-NoExit", "-Command", psCmd}
        } else {
            shell = os.Getenv("ComSpec")
            if shell == "" {
                shell = "cmd.exe"
            }
            // Set PROMPT env for cmd.exe (use > as the right prompt glyph)
            env = append(env, "PROMPT=(bnv) $G ")
            // Keep interactive session
            args = []string{}
        }
    } else {
        shell = os.Getenv("SHELL")
        if shell == "" {
            if _, err := os.Stat("/bin/zsh"); err == nil {
                shell = "/bin/zsh"
            } else if _, err := os.Stat("/bin/bash"); err == nil {
                shell = "/bin/bash"
            } else {
                shell = "/bin/sh"
            }
        }

        // Force interactive for common shells
        base := filepath.Base(shell)
        switch base {
        case "bash", "zsh", "fish", "sh", "dash", "ksh":
            args = append(args, "-i")
        }

        // Best-effort prompt indicator per shell
        switch base {
        case "zsh":
            // Use a temporary ZDOTDIR with a wrapper .zshrc that sources user's rc then sets PROMPT fully
            if tmpDir, err := os.MkdirTemp("", "bnv_zsh_"); err == nil {
                rcPath := filepath.Join(tmpDir, ".zshrc")
                content := fmt.Sprintf("if [ -f \"$HOME/.zshrc\" ]; then source \"$HOME/.zshrc\"; fi\nPROMPT=\"%s\"\n", promptPrefix)
                _ = os.WriteFile(rcPath, []byte(content), 0600)
                env = append(env, "ZDOTDIR="+tmpDir)
                defer os.RemoveAll(tmpDir)
            }
        case "bash":
            // Use a temporary rcfile that sources user's .bashrc then sets PS1 fully
            if f, err := os.CreateTemp("", "bnv_bashrc_*.sh"); err == nil {
                _ = f.Close()
                content := fmt.Sprintf("if [ -f \"$HOME/.bashrc\" ]; then . \"$HOME/.bashrc\"; fi\nPS1=\"%s\"\n", promptPrefix)
                _ = os.WriteFile(f.Name(), []byte(content), 0600)
                args = append(args, "--rcfile", f.Name())
                defer os.Remove(f.Name())
            }
        case "fish":
            // Override fish_prompt fully for this session only using -C initializer
            fishInit := "functions -q __bnv_orig_prompt; or functions -c fish_prompt __bnv_orig_prompt; function fish_prompt; echo -n '" + promptPrefix + "'; end"
            args = append(args, "-C", fishInit)
        default:
            // Set PS1 fully for other POSIX shells
            env = append(env, "PS1="+promptPrefix)
        }
    }

    fmt.Fprintln(os.Stderr, "Entering subshell with better-env secrets loaded. Type 'exit' to return.")
    cmd := exec.Command(shell, args...)
    cmd.Env = env
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            os.Exit(exitErr.ExitCode())
        }
        return fmt.Errorf("failed to start subshell: %w", err)
    }

    return nil
}

var (
    shellCmd = &cobra.Command{
        Use:   "shell",
        Short: "Start a subshell with secrets loaded into its environment",
        Long:  "Start an interactive subshell where project secrets are available as environment variables. No shell eval or aliases required; exiting returns to your original environment.",
        RunE: func(cmd *cobra.Command, args []string) error {
            sc := NewShellCommand()
            return sc.Run()
        },
    }
)

func init() {
    rootCmd.AddCommand(shellCmd)
}


