package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type AddCommand struct {
	keys []string
}

func NewAddCommand(keys []string) *AddCommand {
	return &AddCommand{
		keys: keys,
	}
}

func (a *AddCommand) Run() error {
	// 1. Find the .better-env file in current directory
	configPath := filepath.Join(".", ".better-env")

	if !exists(configPath) {
		return fmt.Errorf(".better-env file not found. Run 'bnv init' first")
	}

	// 2. Read existing project config
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read .better-env: %w", err)
	}

	var projectConfig ProjectConfig
	if err := json.Unmarshal(configData, &projectConfig); err != nil {
		return fmt.Errorf("failed to parse .better-env: %w", err)
	}

	// 3. Verify the global store exists and has these keys
	storePath := projectConfig.StorePath
	secretsPath := filepath.Join(storePath, SecretsFileName)

	if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
		return fmt.Errorf("global secrets store not found at %s. Run 'bnv setup' first", storePath)
	}

	// 4. Add new keys to the config (avoiding duplicates)
	existingKeys := make(map[string]bool)
	for _, key := range projectConfig.Keys {
		existingKeys[key] = true
	}

	addedCount := 0
	for _, key := range a.keys {
		if !existingKeys[key] {
			projectConfig.Keys = append(projectConfig.Keys, key)
			existingKeys[key] = true
			addedCount++
		}
	}

	// 5. Save updated config
	if addedCount > 0 {
		updatedData, err := json.MarshalIndent(projectConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		if err := os.WriteFile(configPath, updatedData, 0o644); err != nil {
			return fmt.Errorf("failed to write .better-env: %w", err)
		}

		fmt.Printf("Added %d key(s) to .better-env\n", addedCount)
	} else {
		fmt.Println("No changes")
	}

	return nil
}

var (
	addCmd = &cobra.Command{
		Use:   "add KEY1 [KEY2 KEY3 ...]",
		Short: "Add keys to the project's .better-env configuration",
		Long:  "Add one or more secret keys to the current project's .better-env file. These keys will be loaded when running 'bnv launch'.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := NewAddCommand(args)
			return ac.Run()
		},
	}
)

func init() {
	rootCmd.AddCommand(addCmd)
}
