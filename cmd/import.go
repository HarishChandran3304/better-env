package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/spf13/cobra"
)

type ImportCommand struct {
	envFilePath string
}

func NewImportCommand(envFilePath string) *ImportCommand {
	return &ImportCommand{
		envFilePath: envFilePath,
	}
}

func (i *ImportCommand) Run() error {
	// 1. Check if .env file exists
	if _, err := os.Stat(i.envFilePath); os.IsNotExist(err) {
		return fmt.Errorf(".env file not found at: %s", i.envFilePath)
	}

	// 2. Parse .env file
	envVars, err := i.parseEnvFile()
	if err != nil {
		return fmt.Errorf("failed to parse .env file: %w", err)
	}

	if len(envVars) == 0 {
		return fmt.Errorf("no environment variables found in .env file")
	}

	fmt.Printf("Found %d environment variables in .env file\n", len(envVars))

	// 3. Get store path and load config
	storePath, err := getStorePath()
	if err != nil {
		return fmt.Errorf("failed to determine store path: %w", err)
	}

	configPath := filepath.Join(storePath, ConfigFileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("better-env is not configured. Run 'bnv setup' first")
	}

	// 4. Load the private key
	privateKeyPath := filepath.Join(storePath, "private.key")
	privateKeyArmored, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	// 5. Prompt for passphrase
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

	// 6. Load and decrypt existing secrets
	secretsPath := filepath.Join(storePath, SecretsFileName)
	var secrets map[string]string

	encryptedData, err := os.ReadFile(secretsPath)
	if err != nil {
		// If secrets file doesn't exist, start with empty map
		secrets = make(map[string]string)
	} else {
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

		if err := json.Unmarshal(decrypted.Bytes(), &secrets); err != nil {
			return fmt.Errorf("failed to parse secrets: %w", err)
		}
	}

	// 7. Merge env vars into secrets (with conflict detection)
	conflicts := []string{}
	newKeys := []string{}

	for key, value := range envVars {
		if _, exists := secrets[key]; exists {
			conflicts = append(conflicts, key)
		} else {
			newKeys = append(newKeys, key)
		}
		secrets[key] = value
	}

	if len(conflicts) > 0 {
		fmt.Printf("Warning: The following keys already exist and will be overwritten: %v\n", conflicts)
	}
	if len(newKeys) > 0 {
		fmt.Printf("Adding %d new keys: %v\n", len(newKeys), newKeys)
	}

	// 8. Re-encrypt with public key
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
	pgp := crypto.PGP()
	encHandle, err := pgp.Encryption().Recipient(publicKey).New()
	if err != nil {
		return fmt.Errorf("failed to create encryption handle: %w", err)
	}

	pgpMessage, err := encHandle.Encrypt(updatedData)
	if err != nil {
		return fmt.Errorf("failed to encrypt secrets: %w", err)
	}

	// 9. Save encrypted secrets
	if err := os.WriteFile(secretsPath, pgpMessage.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write secrets file: %w", err)
	}

	fmt.Printf("Imported %d environment variables\n", len(envVars))

	// 10. Cleanup prompts
	if err := i.promptCleanup(envVars); err != nil {
		return err
	}

	return nil
}

func (i *ImportCommand) parseEnvFile() (map[string]string, error) {
	file, err := os.Open(i.envFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	envVars := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Warning: Skipping invalid line %d: %s\n", lineNum, line)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, "\"'")

		if key == "" {
			fmt.Fprintf(os.Stderr, "Warning: Skipping line %d with empty key\n", lineNum)
			continue
		}

		envVars[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return envVars, nil
}

func (i *ImportCommand) promptCleanup(envVars map[string]string) error {
	// 1. Prompt to delete .env file
	fmt.Print("\nDelete the original .env file? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "y" || response == "yes" {
		if err := os.Remove(i.envFilePath); err != nil {
			return fmt.Errorf("failed to delete .env file: %w", err)
		}
		fmt.Println("Deleted .env file")
	} else {
		fmt.Println("Kept .env file")
	}

	// 2. Prompt to create .better-env file
	fmt.Print("\nCreate a .better-env file in the same location? (y/n): ")
	response, err = reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "y" || response == "yes" {
		if err := i.createBetterEnvFile(envVars); err != nil {
			return fmt.Errorf("failed to create .better-env file: %w", err)
		}
		fmt.Println("Created .better-env file")
	} else {
		fmt.Println("Skipped creating .better-env file")
	}

	return nil
}

func (i *ImportCommand) createBetterEnvFile(envVars map[string]string) error {
	// Get store path for the config
	storePath, err := getStorePath()
	if err != nil {
		return err
	}

	// Extract keys from envVars
	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		keys = append(keys, key)
	}

	// Create ProjectConfig
	projectConfig := ProjectConfig{
		StorePath: storePath,
		Keys:      keys,
	}

	// Marshal to JSON
	configData, err := json.MarshalIndent(projectConfig, "", "  ")
	if err != nil {
		return err
	}

	// Write to .better-env in the same directory as the .env file
	envDir := filepath.Dir(i.envFilePath)
	betterEnvPath := filepath.Join(envDir, ".better-env")

	if err := os.WriteFile(betterEnvPath, configData, 0644); err != nil {
		return err
	}

	return nil
}

var importCmd = &cobra.Command{
	Use:   "import ENV_FILE_PATH",
	Short: "Import environment variables from a .env file",
	Long:  "Parse an existing .env file and migrate all variables to the better-env global store. Optionally delete the .env file and create a .better-env file.",
	Example: `  bnv import ./.env  
  bnv import /path/to/.env`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ic := NewImportCommand(args[0])
		return ic.Run()
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
