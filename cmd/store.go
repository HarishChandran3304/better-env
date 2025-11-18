package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	beinternal "github.com/HarishChandran3304/better-env/internal"
	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/spf13/cobra"
)

type StoreCommand struct {
	key   string
	value string
}

func NewStoreCommand(key, value string) *StoreCommand {
	return &StoreCommand{
		key:   key,
		value: value,
	}
}

func (s *StoreCommand) Run() error {
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

	// 3. Prompt for passphrase and unlock (3 attempts)
	privateKey, err := beinternal.PromptAndUnlockPrivateKey(privateKeyArmored, false, 3)
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

	// 6. Add/update the key-value pair
	secrets[s.key] = s.value

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

	fmt.Printf("Stored '%s'\n", s.key)
	return nil
}

// removed readPassword; centralized in internal helper

var (
	storeCmd = &cobra.Command{
		Use:   "store KEY",
		Short: "Store an encrypted secret key-value pair in the global store",
		Long:  "Add or update a secret in the encrypted global store. The value will be read interactively. You will be prompted for your passphrase.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prompt for value (hidden input)
			fmt.Print("Enter value: ")
			valueBytes, err := beinternal.ReadPasswordFromStdin()
			if err != nil {
				return fmt.Errorf("failed to read value: %w", err)
			}
			value := strings.TrimSpace(string(valueBytes))

			sc := NewStoreCommand(args[0], value)
			return sc.Run()
		},
	}
)

func init() {
	rootCmd.AddCommand(storeCmd)
}
