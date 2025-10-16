package cmd

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sort"

    beinternal "github.com/HarishChandran3304/better-env/internal"
    "github.com/HarishChandran3304/better-env/internal/ui"
    "github.com/ProtonMail/gopenpgp/v3/crypto"
    "github.com/spf13/cobra"
)

type ListCommand struct {
    showAll bool
}

func NewListCommand(showAll bool) *ListCommand {
    return &ListCommand{
        showAll: showAll,
    }
}

func (l *ListCommand) Run() error {
    if l.showAll {
        return l.listGlobalStore()
    }
    return l.listProjectKeys()
}

func (l *ListCommand) listProjectKeys() error {
    // Read project config from .better-env
    configPath := filepath.Join(".", ".better-env")
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        return fmt.Errorf(".better-env file not found. Run 'bnv init' first or use --all to list global store")
    }

    configData, err := os.ReadFile(configPath)
    if err != nil {
        return fmt.Errorf("failed to read .better-env: %w", err)
    }

    var projectConfig ProjectConfig
    if err := json.Unmarshal(configData, &projectConfig); err != nil {
        return fmt.Errorf("failed to parse .better-env: %w", err)
    }

    if len(projectConfig.Keys) == 0 {
        fmt.Println(ui.Dim("No project keys configured yet."))
        return nil
    }

    sort.Strings(projectConfig.Keys)

    fmt.Println(ui.Section("Project keys"))
    fmt.Println(ui.BulletList(projectConfig.Keys))

    return nil
}

func (l *ListCommand) listGlobalStore() error {
    // Get store path
    storePath, err := getStorePath()
    if err != nil {
        return fmt.Errorf("failed to determine store path: %w", err)
    }

    configPath := filepath.Join(storePath, ConfigFileName)
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        return fmt.Errorf("better-env is not configured. Run 'bnv setup' first")
    }

    // Load the private key
    privateKeyPath := filepath.Join(storePath, "private.key")
    privateKeyArmored, err := os.ReadFile(privateKeyPath)
    if err != nil {
        return fmt.Errorf("failed to read private key: %w", err)
    }

    // Prompt for passphrase and unlock (3 attempts)
    privateKey, err := beinternal.PromptAndUnlockPrivateKey(privateKeyArmored, false, 3)
    if err != nil {
        return fmt.Errorf("failed to unlock private key: %w", err)
    }
    defer privateKey.ClearPrivateParams()

    // Decrypt secrets
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

    // Parse secrets
    var secrets map[string]string
    if err := json.Unmarshal(decrypted.Bytes(), &secrets); err != nil {
        return fmt.Errorf("failed to parse secrets: %w", err)
    }

    if len(secrets) == 0 {
        fmt.Println(ui.Dim("No secrets stored yet."))
        return nil
    }

    // Sort keys for consistent output
    keys := make([]string, 0, len(secrets))
    for key := range secrets {
        keys = append(keys, key)
    }
    sort.Strings(keys)

    fmt.Println(ui.Section("Global secrets"))
    fmt.Println(ui.BulletList(keys))

    return nil
}

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List secret keys",
    Long:  "List keys from the current project's .better-env file, or all keys from the global store with --all flag.",
    Example: `  bnv list           # List keys in current project  
  bnv list --all     # List all keys in global store`,
    RunE: func(cmd *cobra.Command, args []string) error {
        showAll, _ := cmd.Flags().GetBool("all")
        lc := NewListCommand(showAll)
        return lc.Run()
    },
}

func init() {
    listCmd.Flags().BoolP("all", "a", false, "List all keys from global store")
    rootCmd.AddCommand(listCmd)
}
