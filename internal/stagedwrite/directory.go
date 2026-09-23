// Package stagedwrite provides filesystem helpers that keep existing generated output intact until replacement is ready.
package stagedwrite

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReplaceDirectory populates a temporary sibling directory and replaces target only after populate succeeds.
func ReplaceDirectory(target string, populate func(string) error) error {
	if err := validateReplacementTarget(target); err != nil {
		return err
	}

	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create output parent %s: %w", parent, err)
	}

	staging, err := os.MkdirTemp(parent, "."+filepath.Base(target)+".new-*")
	if err != nil {
		return fmt.Errorf("create staged output for %s: %w", target, err)
	}
	defer func() { _ = os.RemoveAll(staging) }()

	if err := populate(staging); err != nil {
		return err
	}

	return replaceDirectory(staging, target)
}

// validateReplacementTarget rejects empty paths and directories containing the current working directory.
func validateReplacementTarget(target string) error {
	if strings.TrimSpace(target) == "" {
		return errors.New("replacement target directory is required")
	}

	absolute, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	current, err := os.Getwd()
	if err != nil {
		return err
	}
	current, err = filepath.Abs(current)
	if err != nil {
		return err
	}

	if directoryContains(absolute, current) {
		return fmt.Errorf("replacement target %s cannot contain the current working directory", target)
	}
	return nil
}

// directoryContains reports whether child is equal to or nested below parent.
func directoryContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

// replaceDirectory swaps one prepared directory into place and restores the previous target if the final rename fails.
func replaceDirectory(staging, target string) error {
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return os.Rename(staging, target)
	}
	if err != nil {
		return fmt.Errorf("inspect existing output %s: %w", target, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output path %s exists and is not a directory", target)
	}

	parent := filepath.Dir(target)
	backupRoot, err := os.MkdirTemp(parent, "."+filepath.Base(target)+".old-*")
	if err != nil {
		return fmt.Errorf("create output backup for %s: %w", target, err)
	}
	defer func() { _ = os.RemoveAll(backupRoot) }()

	backup := filepath.Join(backupRoot, "previous")
	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("move previous output %s aside: %w", target, err)
	}
	if err := os.Rename(staging, target); err != nil {
		rollbackErr := os.Rename(backup, target)
		if rollbackErr != nil {
			return errors.Join(
				fmt.Errorf("install staged output %s: %w", target, err),
				fmt.Errorf("restore previous output %s: %w", target, rollbackErr),
			)
		}
		return fmt.Errorf("install staged output %s: %w", target, err)
	}

	if err := os.RemoveAll(backupRoot); err != nil {
		return fmt.Errorf("remove previous output backup %s: %w", backupRoot, err)
	}
	return nil
}
