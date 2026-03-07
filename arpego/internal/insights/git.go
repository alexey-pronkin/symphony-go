package insights

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type GitGoInspector struct{}

func (GitGoInspector) Inspect(
	_ context.Context,
	source SourceConfig,
	staleAfter time.Duration,
	now time.Time,
) (SCMSourceMetrics, error) {
	if source.RepoPath == "" {
		return SCMSourceMetrics{}, fmt.Errorf("repo path is empty")
	}
	if _, err := os.Stat(source.RepoPath); err != nil {
		return SCMSourceMetrics{}, err
	}

	repo, err := git.PlainOpen(source.RepoPath)
	if err != nil {
		return SCMSourceMetrics{}, err
	}
	mainBranch := source.MainBranch
	if mainBranch == "" {
		mainBranch = "main"
	}
	mainRef, err := repo.Reference(plumbing.NewBranchReferenceName(mainBranch), true)
	if err != nil {
		return SCMSourceMetrics{}, err
	}
	mainSet, err := commitSet(repo, mainRef.Hash())
	if err != nil {
		return SCMSourceMetrics{}, err
	}

	iter, err := repo.Branches()
	if err != nil {
		return SCMSourceMetrics{}, err
	}
	defer iter.Close()

	metrics := SCMSourceMetrics{
		Kind:       source.Kind,
		Name:       source.Name,
		RepoPath:   source.RepoPath,
		MainBranch: mainBranch,
	}

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		branchName := ref.Name().Short()
		if branchName == mainBranch {
			return nil
		}
		metrics.Branches++

		commit, err := repo.CommitObject(ref.Hash())
		if err != nil {
			return err
		}
		ageHours := now.Sub(commit.Committer.When).Hours()
		if ageHours > metrics.MaxAgeHours {
			metrics.MaxAgeHours = round2(ageHours)
		}
		if now.Sub(commit.Committer.When) > staleAfter {
			metrics.StaleBranches++
		}

		branchSet, err := commitSet(repo, ref.Hash())
		if err != nil {
			return err
		}
		behind, ahead := aheadBehind(mainSet, branchSet)
		metrics.DriftCommits += behind
		metrics.AheadCommits += ahead
		if _, merged := mainSet[ref.Hash()]; !merged {
			metrics.UnmergedBranches++
		}
		return nil
	})
	if err != nil {
		return SCMSourceMetrics{}, err
	}

	metrics.MergeReadiness = clampScore(
		100 *
			(0.40*(1-ratio(metrics.DriftCommits, maxInt(metrics.Branches*8, 1))) +
				0.35*(1-ratio(metrics.StaleBranches, maxInt(metrics.Branches, 1))) +
				0.25*(1-ratio(metrics.UnmergedBranches, maxInt(metrics.Branches, 1)))),
	)
	return metrics, nil
}

func commitSet(repo *git.Repository, from plumbing.Hash) (map[plumbing.Hash]struct{}, error) {
	iter, err := repo.Log(&git.LogOptions{From: from})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	out := map[plumbing.Hash]struct{}{}
	err = iter.ForEach(func(commit *object.Commit) error {
		out[commit.Hash] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func aheadBehind(mainSet, branchSet map[plumbing.Hash]struct{}) (behind, ahead int) {
	for hash := range mainSet {
		if _, ok := branchSet[hash]; !ok {
			behind++
		}
	}
	for hash := range branchSet {
		if _, ok := mainSet[hash]; !ok {
			ahead++
		}
	}
	return behind, ahead
}
