package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/spf13/cobra"
)

type ShowCommand struct {
	keys []string
}

func NewShowCommand(keys []string) *ShowCommand {
	return &ShowCommand{
		keys: keys,
	}
}

func (s *ShowCommand) Run() error {
	// 1. Get store path
	storePath, err := getStorePath()
	if err != nil {
		return fmt.Errorf("failed to determine store path: %w", err)
	}

	// 2. Load private key
	privateKeyPath := filepath.Join(storePath, "private.key")
	privateKeyArmored, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	// 3. Prompt for passphrase
	fmt.Print("Enter passphrase: ")
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

	// 6. Display requested keys
	notFound := []string{}
	for _, key := range s.keys {
		value, exists := secrets[key]
		if !exists {
			notFound = append(notFound, key)
			continue
		}
		fmt.Printf("%s=%s\n", key, value)
	}

	// 7. Report any missing keys
	if len(notFound) > 0 {
		fmt.Println()
		for _, key := range notFound {
			fmt.Printf("⚠️  Secret '%s' not found\n", key)
		}
	}

	return nil
}

var showCmd = &cobra.Command{
	Use:   "show KEY [KEY2 KEY3 ...]",
	Short: "Display secret values",
	Long:  "Decrypt and display the values of one or more secrets from the encrypted store. You will be prompted for your passphrase.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sc := NewShowCommand(args)
		return sc.Run()
	},
}
