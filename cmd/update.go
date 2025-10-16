package cmd

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    beinternal "github.com/HarishChandran3304/better-env/internal"
    "github.com/HarishChandran3304/better-env/internal/ui"
    "github.com/ProtonMail/gopenpgp/v3/crypto"
    "github.com/spf13/cobra"
)

type UpdateCommand struct {
    key string
}

func NewUpdateCommand(key string) *UpdateCommand {
    return &UpdateCommand{
        key: key,
    }
}

func (u *UpdateCommand) Run() error {
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

    // 6. Check if key exists
    if _, exists := secrets[u.key]; !exists {
        return fmt.Errorf("key '%s' not found in global store. Use 'bnv set' to create a new key", u.key)
    }

    // 7. Prompt for new value
    fmt.Print(ui.Prompt("Enter new value: "))
    reader := bufio.NewReader(os.Stdin)
    value, err := reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("failed to read value: %w", err)
    }
    value = strings.TrimSpace(value)

    // 8. Update the key-value pair
    secrets[u.key] = value

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

    fmt.Println(ui.Success(fmt.Sprintf("Updated %s", u.key)))
    return nil
}

var updateCmd = &cobra.Command{
    Use:   "update KEY",
    Short: "Update an existing secret's value",
    Long:  "Update the value of an existing secret in the encrypted store. The new value will be read interactively. You will be prompted for your passphrase.",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        uc := NewUpdateCommand(args[0])
        return uc.Run()
    },
}

func init() {
    rootCmd.AddCommand(updateCmd)
}
