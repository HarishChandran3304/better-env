package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/spf13/cobra"
)

type UnloadCommand struct{}

func NewUnloadCommand() *UnloadCommand {
	return &UnloadCommand{}
}

func (u *UnloadCommand) Run() error {
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

	// 6. Output unset statements for requested keys
	if len(projectConfig.Keys) == 0 {
		for key := range secrets {
			fmt.Printf("unset %s\n", key)
		}
	} else {
		for _, key := range projectConfig.Keys {
			fmt.Printf("unset %s\n", key)
		}
	}

	return nil
}

var unloadCmd = &cobra.Command{
	Use:   "unload",
	Short: "Unload secrets from environment variables",
	Long:  "Output unset statements for secrets. Use with eval: eval \"$(bnv unload)\"",
	RunE: func(cmd *cobra.Command, args []string) error {
		uc := NewUnloadCommand()
		return uc.Run()
	},
}
