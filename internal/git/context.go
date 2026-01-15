package git

import (
	"context"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type Context struct {
	RepoPath string `json:"repo_path"`
	Branch   string `json:"branch"`
	IsDirty  bool   `json:"is_dirty"`
	HeadHash string `json:"head_hash"`
}

type Detector struct{}

func NewDetector() *Detector {
	return &Detector{}
}

// DetectContext attempts to find the git context for a given path
// For MVP, we pass the path explicitly or use CWD
func (d *Detector) DetectContext(ctx context.Context, path string) (*Context, error) {
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		path = cwd
	}

	r, err := git.PlainOpen(path)
	if err == git.ErrRepositoryNotExists {
		return nil, nil // No git repo here
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open git repo: %w", err)
	}

	head, err := r.Head()
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			// Empty repo or detached
			return &Context{RepoPath: path, Branch: "HEAD (detached)"}, nil
		}
		return nil, err
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	status, err := w.Status()
	if err != nil {
		return nil, err
	}

	return &Context{
		RepoPath: path,
		Branch:   head.Name().Short(),
		IsDirty:  !status.IsClean(),
		HeadHash: head.Hash().String(),
	}, nil
}
