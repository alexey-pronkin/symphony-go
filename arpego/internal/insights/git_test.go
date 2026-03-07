package insights

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestGitGoInspectorCollectsBranchMetrics(t *testing.T) {
	repoPath := t.TempDir()
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}

	commitAt(t, worktree, repoPath, "README.md", "main\n", "initial", time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC))

	if err := worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("feature/login"),
		Create: true,
	}); err != nil {
		t.Fatalf("Checkout feature/login: %v", err)
	}
	commitAt(t, worktree, repoPath, "feature.txt", "feature\n", "feature", time.Date(2026, 3, 6, 9, 0, 0, 0, time.UTC))

	if err := worktree.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("master")}); err != nil {
		t.Fatalf("Checkout master: %v", err)
	}
	commitAt(t, worktree, repoPath, "README.md", "main v2\n", "main update", time.Date(2026, 3, 5, 9, 0, 0, 0, time.UTC))

	if err := worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("feature/stale"),
		Create: true,
	}); err != nil {
		t.Fatalf("Checkout feature/stale: %v", err)
	}
	commitAt(t, worktree, repoPath, "stale.txt", "stale\n", "stale branch", time.Date(2026, 2, 20, 9, 0, 0, 0, time.UTC))

	if err := worktree.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("master")}); err != nil {
		t.Fatalf("Checkout master: %v", err)
	}

	metrics, err := GitGoInspector{}.Inspect(context.Background(), SourceConfig{
		Kind:       "gitlab",
		Name:       "internal",
		RepoPath:   repoPath,
		MainBranch: "master",
	}, 72*time.Hour, time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if metrics.Branches != 2 {
		t.Fatalf("branches = %d want 2", metrics.Branches)
	}
	if metrics.UnmergedBranches != 2 {
		t.Fatalf("unmerged branches = %d want 2", metrics.UnmergedBranches)
	}
	if metrics.StaleBranches != 1 {
		t.Fatalf("stale branches = %d want 1", metrics.StaleBranches)
	}
	if metrics.DriftCommits == 0 {
		t.Fatal("expected drift commits")
	}
	if metrics.AheadCommits == 0 {
		t.Fatal("expected ahead commits")
	}
}

func commitAt(t *testing.T, worktree *git.Worktree, repoPath, name, content, message string, when time.Time) {
	t.Helper()
	path := filepath.Join(repoPath, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
	if _, err := worktree.Add(name); err != nil {
		t.Fatalf("Add %s: %v", name, err)
	}
	if _, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Symphony Test",
			Email: "symphony@example.com",
			When:  when,
		},
		Committer: &object.Signature{
			Name:  "Symphony Test",
			Email: "symphony@example.com",
			When:  when,
		},
	}); err != nil {
		t.Fatalf("Commit %s: %v", message, err)
	}
}
