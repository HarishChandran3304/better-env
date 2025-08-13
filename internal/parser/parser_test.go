package parser

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempFile creates a temporary file with given contents and returns its path.
func writeTempFile(t *testing.T, dir string, contents string) string {
	t.Helper()
	tmp := filepath.Join(dir, ".better-env")
	if err := os.WriteFile(tmp, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return tmp
}

func TestParseBetterEnvFile_Basic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeTempFile(t, dir, "AWS_API_KEY\nGEMINI_API_KEY\nMONGODB_API_KEY\n")

	got, err := ParseBetterEnvFile(path)
	if err != nil {
		t.Fatalf("ParseBetterEnvFile returned error: %v", err)
	}

	want := []string{"AWS_API_KEY", "GEMINI_API_KEY", "MONGODB_API_KEY"}
	if len(got) != len(want) {
		t.Fatalf("unexpected length: got %d want %d (values: %#v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestParseBetterEnvFile_IgnoresCommentsAndBlanks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "\n# this is a comment\n  \nAPI_KEY\n   # inline comment line should be ignored as whole\n DB_URL  \n\n# end\n"
	path := writeTempFile(t, dir, content)

	got, err := ParseBetterEnvFile(path)
	if err != nil {
		t.Fatalf("ParseBetterEnvFile returned error: %v", err)
	}
	want := []string{"API_KEY", "DB_URL"}
	if len(got) != len(want) {
		t.Fatalf("unexpected length: got %d want %d (%#v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestParseBetterEnvFile_FileNotFound(t *testing.T) {
	t.Parallel()
	_, err := ParseBetterEnvFile(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
}
