package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/spf13/cobra"
)

type LoadCommand struct{}

func NewLoadCommand() *LoadCommand {
	return &LoadCommand{}
}

func (l *LoadCommand) Run() error {
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

	// 3. Prompt for passphrase (to stderr so it doesn't interfere with eval output)
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

	// 6. Output export statements for requested keys
	// If no keys specified in .better-env, export all
	if len(projectConfig.Keys) == 0 {
		for key, value := range secrets {
			fmt.Printf("export %s=%s\n", key, escapeShellValue(value))
		}
	} else {
		// Export only specified keys
		for _, key := range projectConfig.Keys {
			if value, exists := secrets[key]; exists {
				fmt.Printf("export %s=%s\n", key, escapeShellValue(value))
			} else {
				fmt.Fprintf(os.Stderr, "⚠️  Warning: '%s' not found in secrets store\n", key)
			}
		}
	}

	return nil
}

// escapeShellValue properly escapes a value for shell export
// Wraps in single quotes and escapes any single quotes in the value
func escapeShellValue(value string) string {
	// Replace single quotes with '\'' (end quote, escaped quote, start quote)
	escaped := strings.ReplaceAll(value, "'", "'\\''")
	return fmt.Sprintf("'%s'", escaped)
}

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Load secrets into environment variables",
	Long:  "Decrypt and output export statements for secrets. Use with eval: eval \"$(bnv launch)\"",
	RunE: func(cmd *cobra.Command, args []string) error {
		lc := NewLoadCommand()
		return lc.Run()
	},
}

// TODO: Remember to prompt the user to run the following command for this to be able to work:
// TODO: "echo 'bnv() { if [ \"$1\" = \"load\" ]; then eval \"$(command bnv load)\"; else command bnv \"$@\"; fi }' >> ~/.zshrc && source ~/.zshrc"
