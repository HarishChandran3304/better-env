package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/spf13/cobra"
)

type DeleteCommand struct {
	keys []string
}

func NewDeleteCommand(keys []string) *DeleteCommand {
	return &DeleteCommand{
		keys: keys,
	}
}

func (d *DeleteCommand) Run() error {
	// 1. Get store path and load config
	storePath, err := getStorePath()
	if err != nil {
		return fmt.Errorf("failed to determine store path: %w", err)
	}

	configPath := filepath.Join(storePath, ConfigFileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("better-env is not configured. Run 'bnv setup' first")
	}

	// 2. Load the private key
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

	// 4. Load and decrypt existing secrets
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

	// 5. Parse existing secrets
	var secrets map[string]string
	if err := json.Unmarshal(decrypted.Bytes(), &secrets); err != nil {
		return fmt.Errorf("failed to parse secrets: %w", err)
	}

	// 6. Delete the keys and track results
	notFound := []string{}
	deleted := []string{}

	for _, key := range d.keys {
		if _, exists := secrets[key]; exists {
			delete(secrets, key)
			deleted = append(deleted, key)
		} else {
			notFound = append(notFound, key)
		}
	}

	// If no keys were deleted, exit with error
	if len(deleted) == 0 {
		fmt.Fprintf(os.Stderr, "❌ None of the specified keys were found in global store\n")
		fmt.Fprintf(os.Stderr, "ℹ️  Use 'bnv list --all' to see all keys\n")
		os.Exit(1)
	}

	// 7. Re-encrypt with public key
	publicKeyPath := filepath.Join(storePath, "public.key")
	publicKeyArmored, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	publicKey, err := crypto.NewKeyFromArmored(string(publicKeyArmored))
	if err != nil {
		return fmt.Errorf("failed to load public key: %w", err)
	}

	// Marshal updated secrets
	updatedData, err := json.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	// Encrypt
	encHandle, err := pgp.Encryption().Recipient(publicKey).New()
	if err != nil {
		return fmt.Errorf("failed to create encryption handle: %w", err)
	}

	pgpMessage, err := encHandle.Encrypt(updatedData)
	if err != nil {
		return fmt.Errorf("failed to encrypt secrets: %w", err)
	}

	// 8. Save encrypted secrets
	if err := os.WriteFile(secretsPath, pgpMessage.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write secrets file: %w", err)
	}

	// 9. Print results
	if len(deleted) == 1 {
		fmt.Printf("✅ Deleted '%s' from global store\n", deleted[0])
	} else {
		fmt.Printf("✅ Deleted %d keys from global store: %v\n", len(deleted), deleted)
	}

	if len(notFound) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: The following keys were not found: %v\n", notFound)
	}

	fmt.Println("⚠️  Warning: These keys may still be referenced in project .better-env files")

	return nil
}

var deleteCmd = &cobra.Command{
	Use:   "delete KEY [KEY...]",
	Short: "Delete one or more keys from the global store",
	Long:  "Permanently delete keys from the global encrypted store. You will be prompted for your passphrase.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dc := NewDeleteCommand(args)
		return dc.Run()
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
