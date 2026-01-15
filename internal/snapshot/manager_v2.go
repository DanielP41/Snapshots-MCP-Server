package snapshot

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tuusuario/dev-env-snapshots/internal/core"
	"github.com/tuusuario/dev-env-snapshots/internal/git"
	"github.com/tuusuario/dev-env-snapshots/internal/sanitize"
)

type ManagerV2 struct {
	repo      core.Repository
	platform  core.PlatformAdapter
	sanitizer *sanitize.Sanitizer
}

func NewManagerV2(repo core.Repository, platform core.PlatformAdapter) *ManagerV2 {
	return &ManagerV2{
		repo:      repo,
		platform:  platform,
		sanitizer: sanitize.NewSanitizer(sanitize.DefaultOptions()),
	}
}

// SetSanitizationOptions permite configurar la sanitización
func (m *ManagerV2) SetSanitizationOptions(opts sanitize.SanitizationOptions) {
	m.sanitizer = sanitize.NewSanitizer(opts)
}

type CaptureOptionsV2 struct {
	Name             string
	Description      string
	Tags             []string
	IncludeBrowsable bool
	IncludeTerminals bool
	Sanitize         bool // Si es true, sanitiza datos sensibles
}

func (m *ManagerV2) Capture(ctx context.Context, opts CaptureOptionsV2) (*core.Snapshot, error) {
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

	// 2. Capture Terminals
	if opts.IncludeTerminals {
		terminals, err := m.platform.GetTerminals(ctx)
		if err != nil {
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

	// 4. Capture Browsers
	if opts.IncludeBrowsable {
		browsers, err := m.platform.GetBrowserTabs(ctx)
		if err == nil && len(browsers) > 0 {
			s.BrowserTabs = browsers
		}
	}

	// 5. Capture IDEs
	ideFiles, err := m.platform.GetIDEFiles(ctx)
	if err == nil && len(ideFiles) > 0 {
		s.IDEFiles = ideFiles
	}

	// 6. Sanitize if requested
	if opts.Sanitize {
		m.sanitizer.SanitizeSnapshot(s)
	}

	// 7. Save to DB
	if err := m.repo.CreateSnapshot(ctx, s); err != nil {
		return nil, fmt.Errorf("failed to save snapshot metadata: %w", err)
	}

	if len(s.Windows) > 0 {
		if err := m.repo.SaveWindows(ctx, s.ID, s.Windows); err != nil {
			return nil, fmt.Errorf("failed to save windows: %w", err)
		}
	}

	if len(s.Terminals) > 0 {
		if err := m.repo.SaveTerminals(ctx, s.ID, s.Terminals); err != nil {
			return nil, fmt.Errorf("failed to save terminals: %w", err)
		}
	}

	if len(s.BrowserTabs) > 0 {
		if err := m.repo.SaveBrowserTabs(ctx, s.ID, s.BrowserTabs); err != nil {
			return nil, fmt.Errorf("failed to save browser tabs: %w", err)
		}
	}

	if len(s.IDEFiles) > 0 {
		if err := m.repo.SaveIDEFiles(ctx, s.ID, s.IDEFiles); err != nil {
			return nil, fmt.Errorf("failed to save ide files: %w", err)
		}
	}

	return s, nil
}

type RestoreOptionsV2 struct {
	ValidateBeforeRestore bool // Verifica que las apps existan antes de restaurar
	SkipMissingApps       bool // Si true, continúa aunque falten apps
	DryRun                bool // Si true, solo reporta qué haría sin ejecutar
}

func (m *ManagerV2) Restore(ctx context.Context, snapshotID string, opts RestoreOptionsV2) (*RestoreReport, error) {
	s, err := m.repo.GetSnapshotByID(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	if s == nil {
		return nil, fmt.Errorf("snapshot not found")
	}

	// Fetch windows from DB
	windows, err := m.repo.GetWindows(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get windows: %w", err)
	}
	s.Windows = windows

	report := &RestoreReport{
		SnapshotID:   snapshotID,
		TotalWindows: len(s.Windows),
		StartTime:    time.Now(),
	}

	// Validación pre-restore
	if opts.ValidateBeforeRestore {
		missing := m.validateApps(ctx, s.Windows)
		report.MissingApps = missing

		if len(missing) > 0 && !opts.SkipMissingApps {
			report.Success = false
			report.Error = fmt.Sprintf("missing applications: %v", missing)
			return report, fmt.Errorf("cannot restore: missing applications")
		}
	}

	// Dry run mode
	if opts.DryRun {
		report.Success = true
		report.DryRun = true
		report.Message = "Dry run completed - no changes made"
		return report, nil
	}

	// Restore windows
	for _, w := range s.Windows {
		if err := m.platform.RestoreWindow(ctx, w); err != nil {
			report.FailedWindows = append(report.FailedWindows, w.WindowTitle)
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", w.WindowTitle, err))
			continue
		}
		report.RestoredWindows++
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)
	report.Success = report.RestoredWindows > 0

	if report.RestoredWindows == report.TotalWindows {
		report.Message = "All windows restored successfully"
	} else {
		report.Message = fmt.Sprintf("Restored %d/%d windows", report.RestoredWindows, report.TotalWindows)
	}

	return report, nil
}

// RestoreReport contiene el resultado detallado de una restauración
type RestoreReport struct {
	SnapshotID      string
	TotalWindows    int
	RestoredWindows int
	FailedWindows   []string
	MissingApps     []string
	Errors          []string
	Success         bool
	DryRun          bool
	Error           string
	Message         string
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
}

// validateApps verifica qué aplicaciones están instaladas
func (m *ManagerV2) validateApps(ctx context.Context, windows []core.Window) []string {
	// Obtener ventanas actuales para ver qué apps están corriendo
	currentWindows, err := m.platform.GetWindows(ctx)
	if err != nil {
		return nil
	}

	// Crear set de apps disponibles
	availableApps := make(map[string]bool)
	for _, w := range currentWindows {
		availableApps[w.AppName] = true
	}

	// Verificar qué apps faltan
	var missing []string
	checked := make(map[string]bool)

	for _, w := range windows {
		if checked[w.AppName] {
			continue
		}
		checked[w.AppName] = true

		if !availableApps[w.AppName] {
			missing = append(missing, w.AppName)
		}
	}

	return missing
}

func (m *ManagerV2) List(ctx context.Context) ([]core.Snapshot, error) {
	return m.repo.ListSnapshots(ctx, core.SnapshotFilter{Limit: 50})
}

func (m *ManagerV2) Delete(ctx context.Context, id string) error {
	return m.repo.DeleteSnapshot(ctx, id)
}

func (m *ManagerV2) Diff(ctx context.Context, id1, id2 string) (*DiffResult, error) {
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
