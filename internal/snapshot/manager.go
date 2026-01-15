package snapshot

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tuusuario/dev-env-snapshots/internal/core"
	"github.com/tuusuario/dev-env-snapshots/internal/git"
)

type Manager struct {
	repo     core.Repository
	platform core.PlatformAdapter
}

func NewManager(repo core.Repository, platform core.PlatformAdapter) *Manager {
	return &Manager{
		repo:     repo,
		platform: platform,
	}
}

type CaptureOptions struct {
	Name             string
	Description      string
	Tags             []string
	IncludeBrowsable bool // Browsers
	IncludeTerminals bool
}

func (m *Manager) Capture(ctx context.Context, opts CaptureOptions) (*core.Snapshot, error) {
	s := &core.Snapshot{
		ID:          uuid.New().String(),
		Name:        opts.Name,
		Description: opts.Description,
		Tags:        opts.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 1. Capture Windows
	windows, err := m.platform.GetWindows(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to capture windows: %w", err)
	}
	s.Windows = windows

	// 2. Capture Terminals (if requested)
	if opts.IncludeTerminals {
		terminals, err := m.platform.GetTerminals(ctx)
		if err != nil {
			// Log error and return strictly if terminals capture fails.
			return nil, fmt.Errorf("failed to capture terminals: %w", err)
		}
		s.Terminals = terminals
	}

	// 3. Capture Git Context
	detector := git.NewDetector()
	gitCtx, err := detector.DetectContext(ctx, "")
	if err == nil && gitCtx != nil {
		s.GitBranch = gitCtx.Branch
		s.GitRepo = gitCtx.RepoPath
		s.GitDirty = gitCtx.IsDirty
		s.GitHeadHash = gitCtx.HeadHash
	}

	// 4. Save to DB
	if err := m.repo.CreateSnapshot(ctx, s); err != nil {
		return nil, fmt.Errorf("failed to save snapshot metadata: %w", err)
	}

	if len(s.Windows) > 0 {
		if err := m.repo.SaveWindows(ctx, s.ID, s.Windows); err != nil {
			return nil, fmt.Errorf("failed to save windows: %w", err)
		}
	}

	// Save Terminals
	if len(s.Terminals) > 0 {
		if err := m.repo.SaveTerminals(ctx, s.ID, s.Terminals); err != nil {
			return nil, fmt.Errorf("failed to save terminals: %w", err)
		}
	}

	// Capture and Save Browsers
	if opts.IncludeBrowsable {
		browsers, err := m.platform.GetBrowserTabs(ctx)
		if err == nil && len(browsers) > 0 {
			s.BrowserTabs = browsers
			if err := m.repo.SaveBrowserTabs(ctx, s.ID, s.BrowserTabs); err != nil {
				return nil, fmt.Errorf("failed to save browser tabs: %w", err)
			}
		}
	}

	// Capture and Save IDEs
	ideFiles, err := m.platform.GetIDEFiles(ctx)
	if err == nil && len(ideFiles) > 0 {
		s.IDEFiles = ideFiles
		if err := m.repo.SaveIDEFiles(ctx, s.ID, s.IDEFiles); err != nil {
			return nil, fmt.Errorf("failed to save ide files: %w", err)
		}
	}

	return s, nil
}

func (m *Manager) Restore(ctx context.Context, snapshotID string) error {
	s, err := m.repo.GetSnapshotByID(ctx, snapshotID)
	if err != nil {
		return fmt.Errorf("failed to get snapshot: %w", err)
	}
	if s == nil {
		return fmt.Errorf("snapshot not found")
	}

	// Restore logic
	// Note: In a production implementation, windows should be fetched from the database if not already populated.
	// For this version, we assume windows are either populated or we fetch them now.

	// Fetch windows if not populated
	windows, err := m.repo.GetWindows(ctx, snapshotID)
	if err == nil {
		s.Windows = windows
	}

	for _, w := range s.Windows {
		if err := m.platform.RestoreWindow(ctx, w); err != nil {
			fmt.Printf("Error restoring window %s: %v\n", w.AppName, err)
			continue
		}
	}

	return nil
}

func (m *Manager) List(ctx context.Context) ([]core.Snapshot, error) {
	return m.repo.ListSnapshots(ctx, core.SnapshotFilter{Limit: 50})
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	return m.repo.DeleteSnapshot(ctx, id)
}

type DiffResult struct {
	SourceID       string
	TargetID       string
	GitChanged     bool
	AddedWindows   []string
	RemovedWindows []string
	CommonWindows  int
}

func (m *Manager) Diff(ctx context.Context, id1, id2 string) (*DiffResult, error) {
	s1, err := m.repo.GetSnapshotByID(ctx, id1)
	if err != nil {
		return nil, err
	}
	s2, err := m.repo.GetSnapshotByID(ctx, id2)
	if err != nil {
		return nil, err
	}

	if s1 == nil || s2 == nil {
		return nil, fmt.Errorf("one or both snapshots not found")
	}

	// Get Windows for both (assuming loaded or need fetch)
	// For MVP doing simple list diff based on Title
	w1, _ := m.repo.GetWindows(ctx, id1)
	w2, _ := m.repo.GetWindows(ctx, id2)

	diff := &DiffResult{
		SourceID:   id1,
		TargetID:   id2,
		GitChanged: s1.GitBranch != s2.GitBranch || s1.GitRepo != s2.GitRepo,
	}

	titles1 := make(map[string]bool)
	for _, w := range w1 {
		titles1[w.WindowTitle] = true
	}

	titles2 := make(map[string]bool)
	for _, w := range w2 {
		titles2[w.WindowTitle] = true
	}

	for t := range titles2 {
		if !titles1[t] {
			diff.AddedWindows = append(diff.AddedWindows, t)
		} else {
			diff.CommonWindows++
		}
	}
	for t := range titles1 {
		if !titles2[t] {
			diff.RemovedWindows = append(diff.RemovedWindows, t)
		}
	}

	return diff, nil
}
