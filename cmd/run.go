package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
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

	// 1. Read project config from .better-env
	configPath := filepath.Join(".", ".better-env")
	if !exists(configPath) {
		return fmt.Errorf(".better-env file not found. Run 'bnv init' first")
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read .better-env: %w", err)
	}

	var projectConfig ProjectConfig
	if err := json.Unmarshal(configData, &projectConfig); err != nil {
		return fmt.Errorf("failed to parse .better-env: %w", err)
	}

	// 2. Load private key
	storePath := projectConfig.StorePath
	privateKeyPath := filepath.Join(storePath, "private.key")
	privateKeyArmored, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	// 3. Prompt for passphrase
	fmt.Fprint(os.Stderr, "Enter passphrase: ")
	passphrase, err := readPassword()
	if err != nil {
		return fmt.Errorf("failed to read passphrase: %w", err)
	}

	privateKey, err := crypto.NewPrivateKeyFromArmored(string(privateKeyArmored), passphrase)
	if err != nil {
		return fmt.Errorf("failed to unlock private key: %w", err)
	}
	defer privateKey.ClearPrivateParams()

	// 4. Decrypt secrets
	secretsPath := filepath.Join(storePath, SecretsFileName)
	encryptedData, err := os.ReadFile(secretsPath)
	if err != nil {
		return fmt.Errorf("failed to read secrets file: %w", err)
	}

	pgp := crypto.PGP()
	decHandle, err := pgp.Decryption().DecryptionKey(privateKey).New()
	if err != nil {
		return fmt.Errorf("failed to create decryption handle: %w", err)
	}
	defer decHandle.ClearPrivateParams()

	decrypted, err := decHandle.Decrypt(encryptedData, crypto.Bytes)
	if err != nil {
		return fmt.Errorf("failed to decrypt secrets: %w", err)
	}

	// 5. Parse secrets
	var secrets map[string]string
	if err := json.Unmarshal(decrypted.Bytes(), &secrets); err != nil {
		return fmt.Errorf("failed to parse secrets: %w", err)
	}

	// 6. Build environment with secrets
	env := os.Environ()

	if len(projectConfig.Keys) == 0 {
		// Add all secrets
		for key, value := range secrets {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	} else {
		// Add only specified keys
		for _, key := range projectConfig.Keys {
			if value, exists := secrets[key]; exists {
				env = append(env, fmt.Sprintf("%s=%s", key, value))
			} else {
				fmt.Fprintf(os.Stderr, "Warning: '%s' not found in secrets store\n", key)
			}
		}
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
