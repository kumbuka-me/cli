package preview

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotDetectsFileChangesAndMissingFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	missing := filepath.Join(root, "kumbuka-site.toml")
	before, err := snapshot([]string{root, missing})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(missing, []byte("site_name = \"Docs\""), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := snapshot([]string{root, missing})
	if err != nil {
		t.Fatal(err)
	}
	if sameSnapshot(before, after) {
		t.Fatal("new file was not detected")
	}
}

func TestSnapshotDoesNotRecurseIntoHiddenDirectories(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	git := filepath.Join(root, ".git")
	if err := os.Mkdir(git, 0o755); err != nil {
		t.Fatal(err)
	}
	states, err := snapshot([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := states[git]; found {
		t.Fatal("hidden directory should not be scanned")
	}
}
