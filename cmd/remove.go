package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type RemoveCommand struct {
	keys []string
}

func NewRemoveCommand(keys []string) *RemoveCommand {
	return &RemoveCommand{
		keys: keys,
	}
}

func (r *RemoveCommand) Run() error {
	// 1. Read project config from .better-env
	configPath := filepath.Join(".", ".better-env")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
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

	// 2. Create a set of keys to remove for efficient lookup
	keysToRemove := make(map[string]bool)
	for _, key := range r.keys {
		keysToRemove[key] = true
	}

	// 3. Find and remove the keys
	notFound := []string{}
	removed := []string{}
	newKeys := make([]string, 0, len(projectConfig.Keys))

	for _, k := range projectConfig.Keys {
		if keysToRemove[k] {
			removed = append(removed, k)
		} else {
			newKeys = append(newKeys, k)
		}
	}

	// Check for keys that weren't found
	for _, key := range r.keys {
		found := false
		for _, removedKey := range removed {
			if key == removedKey {
				found = true
				break
			}
		}
		if !found {
			notFound = append(notFound, key)
		}
	}

	// If no keys were removed, exit with error
	if len(removed) == 0 {
		fmt.Fprintf(os.Stderr, "❌ None of the specified keys were found in project configuration\n")
		fmt.Fprintf(os.Stderr, "ℹ️ Use 'bnv list' to see configured keys\n")
		os.Exit(1)
	}

	projectConfig.Keys = newKeys

	// 4. Save updated config
	updatedData, err := json.MarshalIndent(projectConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write .better-env: %w", err)
	}

	// 5. Print results
	if len(removed) == 1 {
		fmt.Printf("✅ Removed '%s' from project configuration\n", removed[0])
	} else {
		fmt.Printf("✅ Removed %d keys from project configuration: %v\n", len(removed), removed)
	}

	if len(notFound) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: The following keys were not found: %v\n", notFound)
	}

	fmt.Println("ℹ️  Note: The keys still exist in the global store. Use 'bnv delete' to remove them completely.")

	return nil
}

var removeCmd = &cobra.Command{
	Use:   "remove KEY [KEY...]",
	Short: "Remove one or more keys from the current project",
	Long:  "Remove keys from the current project's .better-env file. The keys will still exist in the global store.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rc := NewRemoveCommand(args)
		return rc.Run()
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
