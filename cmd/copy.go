package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

type CopyCommand struct {
	keys []string
}

func NewCopyCommand(keys []string) *CopyCommand {
	return &CopyCommand{
		keys: keys,
	}
}

func (c *CopyCommand) Run() error {
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

	// 6. Collect values for requested keys
	var values []string
	notFound := []string{}

	for _, key := range c.keys {
		value, exists := secrets[key]
		if !exists {
			notFound = append(notFound, key)
			continue
		}
		values = append(values, fmt.Sprintf("%s=%s", key, value))
	}

	// 7. Copy to clipboard
	if len(values) > 0 {
		clipboardContent := strings.Join(values, "\n")
		if err := clipboard.WriteAll(clipboardContent); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}

		if len(c.keys) == 1 {
			fmt.Printf("✅ Secret '%s' copied to clipboard\n", c.keys[0])
		} else {
			fmt.Printf("✅ %d secrets copied to clipboard\n", len(values))
		}
	}

	// 8. Report any missing keys
	if len(notFound) > 0 {
		fmt.Println()
		for _, key := range notFound {
			fmt.Printf("⚠️  Secret '%s' not found\n", key)
		}
	}

	return nil
}

var copyCmd = &cobra.Command{
	Use:   "copy KEY [KEY2 KEY3 ...]",
	Short: "Copy secret values to clipboard",
	Long:  "Decrypt and copy the values of one or more secrets to your clipboard. You will be prompted for your passphrase. The values will not be displayed on screen.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cc := NewCopyCommand(args)
		return cc.Run()
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
}
