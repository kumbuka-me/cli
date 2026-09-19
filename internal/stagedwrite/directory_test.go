package stagedwrite

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceDirectoryInstallsPreparedOutput(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "site")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := ReplaceDirectory(target, func(staging string) error {
		return os.WriteFile(filepath.Join(staging, "new.txt"), []byte("new"), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(target, "new.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("new output = %q, want %q", data, "new")
	}
	if _, err := os.Stat(filepath.Join(target, "old.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old output still exists: %v", err)
	}
}

func TestReplaceDirectoryPreservesExistingOutputWhenPopulateFails(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "site")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "current.txt"), []byte("current"), 0o644); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("generation failed")

	err := ReplaceDirectory(target, func(staging string) error {
		if err := os.WriteFile(filepath.Join(staging, "partial.txt"), []byte("partial"), 0o644); err != nil {
			t.Fatal(err)
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("error = %v, want %v", err, failure)
	}

	data, readErr := os.ReadFile(filepath.Join(target, "current.txt"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "current" {
		t.Fatalf("current output = %q, want %q", data, "current")
	}
	if _, err := os.Stat(filepath.Join(target, "partial.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial output leaked into target: %v", err)
	}
}
