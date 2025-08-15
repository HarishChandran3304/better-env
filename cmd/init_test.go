package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// executeRoot runs the root command with the provided args and returns the error.
func executeRoot(args ...string) error {
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func TestInit_CreatesFileInCwdByDefault(t *testing.T) {
	dir := t.TempDir()

	// Ensure clean state for globals across tests
	initPath = "."
	initForce = false

	// Change into temp directory so default path "." writes here
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := executeRoot("init"); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	target := filepath.Join(dir, ".better-env")
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat %s: %v", target, err)
	}
	if info.IsDir() {
		t.Fatalf("expected file, got directory: %s", target)
	}
	// defaultConfig is empty string
	if info.Size() != 0 {
		t.Fatalf("expected empty file, size=%d", info.Size())
	}
}

func TestInit_RespectsPathFlagAndCreatesDir(t *testing.T) {
	tmp := t.TempDir()
	targetDir := filepath.Join(tmp, "nested/project")

	initPath = "."
	initForce = false

	if err := executeRoot("init", "--path", targetDir); err != nil {
		t.Fatalf("init --path returned error: %v", err)
	}

	if _, err := os.Stat(targetDir); err != nil {
		t.Fatalf("expected target dir to exist: %v", err)
	}
	f := filepath.Join(targetDir, ".better-env")
	info, err := os.Stat(f)
	if err != nil {
		t.Fatalf("stat %s: %v", f, err)
	}
	if info.Size() != 0 {
		t.Fatalf("expected empty file, size=%d", info.Size())
	}
}

func TestInit_DoesNotOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".better-env")
	if err := os.WriteFile(file, []byte("existing"), 0o644); err != nil {
		t.Fatalf("prewrite: %v", err)
	}

	initPath = "."
	initForce = false

	err := executeRoot("init", "--path", dir)
	if err == nil {
		t.Fatalf("expected error when file exists without --force, got nil")
	}

	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(b) != "existing" {
		t.Fatalf("file was modified unexpectedly: %q", string(b))
	}
}

func TestInit_OverwritesWithForce(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".better-env")
	if err := os.WriteFile(file, []byte("existing"), 0o644); err != nil {
		t.Fatalf("prewrite: %v", err)
	}

	initPath = "."
	initForce = false

	if err := executeRoot("init", "--path", dir, "--force"); err != nil {
		t.Fatalf("init --force returned error: %v", err)
	}

	info, err := os.Stat(file)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("expected empty file after overwrite, size=%d", info.Size())
	}
}

func TestInit_ErrorsWhenPathIsAFile(t *testing.T) {
	dir := t.TempDir()
	notDir := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(notDir, []byte("x"), 0o644); err != nil {
		t.Fatalf("prewrite: %v", err)
	}

	initPath = "."
	initForce = false

	if err := executeRoot("init", "--path", notDir); err == nil {
		t.Fatalf("expected error when --path points to a file, got nil")
	}
}
