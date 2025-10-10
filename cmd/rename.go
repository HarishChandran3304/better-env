package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	beinternal "github.com/HarishChandran3304/better-env/internal"
	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/spf13/cobra"
)

type RenameCommand struct {
	oldKey string
	newKey string
}

func NewRenameCommand(oldKey, newKey string) *RenameCommand {
	return &RenameCommand{
		oldKey: oldKey,
		newKey: newKey,
	}
}

func (r *RenameCommand) Run() error {
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

	// 6. Check if old key exists
	value, exists := secrets[r.oldKey]
	if !exists {
		return fmt.Errorf("key '%s' not found in global store", r.oldKey)
	}

	// 7. Check if new key already exists
	if _, exists := secrets[r.newKey]; exists {
		return fmt.Errorf("key '%s' already exists in global store. Delete it first or choose a different name", r.newKey)
	}

	// 8. Rename the key
	secrets[r.newKey] = value
	delete(secrets, r.oldKey)

	// 9. Re-encrypt with public key
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

	// 10. Save encrypted secrets
	if err := os.WriteFile(secretsPath, pgpMessage.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write secrets file: %w", err)
	}

	fmt.Printf("Renamed '%s' to '%s' in global store\n", r.oldKey, r.newKey)
	fmt.Println("Note: Project .better-env files may still reference the old key name. Update them manually.")

	// 11. Offer to update current project's .better-env if it exists
	if err := r.updateCurrentProject(); err != nil {
		fmt.Fprintf(os.Stderr, "Note: %v\n", err)
	}

	return nil
}

func (r *RenameCommand) updateCurrentProject() error {
	configPath := filepath.Join(".", ".better-env")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Silently skip if there's no .better-env in the current directory
		return nil
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var projectConfig ProjectConfig
	if err := json.Unmarshal(configData, &projectConfig); err != nil {
		return err
	}

	// Check if old key is in project config
	found := false
	for i, key := range projectConfig.Keys {
		if key == r.oldKey {
			projectConfig.Keys[i] = r.newKey
			found = true
			break
		}
	}

	if !found {
		// Silently skip if the old key isn't referenced in the project's .better-env
		return nil
	}

	// Ask user if they want to update
	fmt.Print("\n.better-env file detected in current directory. Update current project's .better-env file? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("skipped updating current project")
	}

	// Save updated config
	updatedData, err := json.MarshalIndent(projectConfig, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return err
	}

	fmt.Println("Updated current project's .better-env file")
	return nil
}

var renameCmd = &cobra.Command{
	Use:   "rename OLD_KEY NEW_KEY",
	Short: "Rename a key in the global store",
	Long:  "Rename a key in the global encrypted store. You will be prompted for your passphrase. Note: This does not automatically update project .better-env files.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		rc := NewRenameCommand(args[0], args[1])
		return rc.Run()
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
