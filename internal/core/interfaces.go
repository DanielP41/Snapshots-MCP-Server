package core

import "context"

// PlatformAdapter defines the contract for OS-specific operations
type PlatformAdapter interface {
	// Name returns the name of the platform (e.g., "linux-x11", "windows")
	Name() string

	// Windows
	GetWindows(ctx context.Context) ([]Window, error)
	RestoreWindow(ctx context.Context, window Window) error
	CloseWindow(ctx context.Context, window Window) error

	// Terminals
	GetTerminals(ctx context.Context) ([]Terminal, error)
	RestoreTerminal(ctx context.Context, terminal Terminal) error

	// Browsers
	GetBrowserTabs(ctx context.Context) ([]BrowserTab, error)
	OpenURL(ctx context.Context, url string, browser string) error

	// IDEs
	GetIDEFiles(ctx context.Context) ([]IDEFile, error)

	// Basic Process info
	GetProcesses(ctx context.Context) ([]Process, error)
	StartProcess(ctx context.Context, process Process) error
}

// Repository defines the persistence layer operations
type Repository interface {
	// Snapshots
	CreateSnapshot(ctx context.Context, snapshot *Snapshot) error
	GetSnapshotByID(ctx context.Context, id string) (*Snapshot, error)
	ListSnapshots(ctx context.Context, filter SnapshotFilter) ([]Snapshot, error)
	DeleteSnapshot(ctx context.Context, id string) error

	// Components
	SaveWindows(ctx context.Context, snapshotID string, windows []Window) error
	SaveTerminals(ctx context.Context, snapshotID string, terminals []Terminal) error
	SaveBrowserTabs(ctx context.Context, snapshotID string, tabs []BrowserTab) error
	SaveIDEFiles(ctx context.Context, snapshotID string, files []IDEFile) error
	GetWindows(ctx context.Context, snapshotID string) ([]Window, error)
	// Add other component methods as needed
}

// SnapshotFilter defines criteria for listing snapshots
type SnapshotFilter struct {
	Project string
	Branch  string
	Tags    []string
	Limit   int
	Offset  int
}
