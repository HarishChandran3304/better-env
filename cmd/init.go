package cmd

import (
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

const defaultConfig = ``

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a .better-env file in the current project",
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

		if err := os.WriteFile(target, []byte(defaultConfig), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}

		fmt.Println("Created empty .better-env file.")
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
