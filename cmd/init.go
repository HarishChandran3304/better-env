package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	initPath  string
	initForce bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a .better-env file in the current project",
	Long:  "Initialize better-env for this project by creating a .better-env configuration file that links to your global encrypted store.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if initPath != "" {
			dir = initPath
		}

		if err := ensureDir(dir); err != nil {
			return err
		}

		target := filepath.Join(dir, ".better-env")

		if exists(target) && !initForce {
			return fmt.Errorf("%s already exists (use --force to overwrite)", target)
		}

		// Get the global store path
		storePath, err := getStorePath()
		if err != nil {
			return fmt.Errorf("failed to determine store path: %w", err)
		}

		// Verify that setup has been run
		configPath := filepath.Join(storePath, ConfigFileName)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("better-env is not configured. Run 'bnv setup' first")
		}

		// Create project config (no store path; commit-friendly)
		projectConfig := ProjectConfig{
			Keys: []string{}, // Empty initially, user can add specific keys later
		}

		configData, err := json.MarshalIndent(projectConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}

		if err := os.WriteFile(target, configData, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}

		fmt.Println("Created .better-env configuration file.")
		fmt.Printf("Linked to store: %s\n", storePath)
		fmt.Println()
		// fmt.Println("Next steps:")
		// fmt.Println("  1. Add secrets to your store: bnv add KEY")
		// fmt.Println("  2. Load secrets in this project: bnv load")
		return nil
	},
}

func init() {
	initCmd.Flags().StringVarP(&initPath, "path", "p", ".", "directory to place .better-env")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite if the file already exists")
	rootCmd.AddCommand(initCmd)
}

func ensureDir(p string) error {
	info, err := os.Stat(p)
	if err == nil && info.IsDir() {
		return nil
	}
	if err == nil && !info.IsDir() {
		return errors.New("path exists and is not a directory")
	}
	if os.IsNotExist(err) {
		return os.MkdirAll(p, 0o755)
	}
	return err
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
