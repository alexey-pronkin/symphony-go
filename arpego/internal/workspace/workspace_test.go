package workspace_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/workspace"
)

func TestSanitizeKey(t *testing.T) {
	if got, want := workspace.SanitizeKey("MT 101/alpha"), "MT_101_alpha"; got != want {
		t.Fatalf("sanitize = %q want %q", got, want)
	}
}

func TestValidatePathRejectsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "escape")
	if err := workspace.ValidatePath(root, outside); err == nil {
		t.Fatal("expected outside-root error")
	}
}

func TestEnsureWorkspaceCreateAndReuse(t *testing.T) {
	root := t.TempDir()

	first, err := workspace.EnsureWorkspace(root, "MT-101")
	if err != nil {
		t.Fatalf("ensure first: %v", err)
	}
	if !first.CreatedNow {
		t.Fatal("expected first workspace to be newly created")
	}
	if _, err := os.Stat(first.Path); err != nil {
		t.Fatalf("stat workspace: %v", err)
	}

	second, err := workspace.EnsureWorkspace(root, "MT-101")
	if err != nil {
		t.Fatalf("ensure second: %v", err)
	}
	if second.CreatedNow {
		t.Fatal("expected second workspace to be reused")
	}
	if first.Path != second.Path {
		t.Fatalf("path mismatch: %q vs %q", first.Path, second.Path)
	}
}

func TestRunHookTimeout(t *testing.T) {
	err := workspace.RunHook("sleep 2", t.TempDir(), 100)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestRunHookNonZeroExit(t *testing.T) {
	err := workspace.RunHook("exit 7", t.TempDir(), int((2 * time.Second).Milliseconds()))
	if err == nil {
		t.Fatal("expected exit error")
	}
}
