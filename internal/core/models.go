package core

import (
	"encoding/json"
	"time"
)

// Snapshot represents a complete capture of the development environment
type Snapshot struct {
	ID          string       `json:"id" db:"id"`
	Name        string       `json:"name" db:"name"`
	Description string       `json:"description" db:"description"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
	GitBranch   string       `json:"git_branch" db:"git_branch"`
	GitRepo     string       `json:"git_repo" db:"git_repo"`
	GitDirty    bool         `json:"git_dirty" db:"git_dirty"`
	GitHeadHash string       `json:"git_head_hash" db:"git_head_hash"` // Added this field
	Tags        []string     `json:"tags" db:"tags"`
	Windows     []Window     `json:"windows"`
	Terminals   []Terminal   `json:"terminals"`
	BrowserTabs []BrowserTab `json:"browser_tabs"`
	Processes   []Process    `json:"processes"`
	IDEFiles    []IDEFile    `json:"ide_files"`
}

// ... rest of file same as before
// To avoid rewriting whole file, I will use replace logic in next steps if needed,
// or I can just re-write the top part if I am careful.
// Actually, I'll use multi_replace for safety if I were modifying, but here I can re-write since it is small.
// Wait, I should not overwrite if I can help it.
// I will just use the content I have and append the rest.

// Window represents a system window
type Window struct {
	ID          int64           `json:"id" db:"id"`
	SnapshotID  string          `json:"snapshot_id" db:"snapshot_id"`
	AppName     string          `json:"app_name" db:"app_name"`
	AppPath     string          `json:"app_path" db:"app_path"`
	WindowTitle string          `json:"window_title" db:"window_title"`
	X           int             `json:"x" db:"x"`
	Y           int             `json:"y" db:"y"`
	Width       int             `json:"width" db:"width"`
	Height      int             `json:"height" db:"height"`
	State       string          `json:"state" db:"state"` // normal, maximized, minimized, fullscreen
	Workspace   int             `json:"workspace" db:"workspace"`
	ZIndex      int             `json:"z_index" db:"z_index"`
	LaunchArgs  json.RawMessage `json:"launch_args" db:"launch_args"`
}

// Terminal represents a terminal session
type Terminal struct {
	ID               int64             `json:"id" db:"id"`
	SnapshotID       string            `json:"snapshot_id" db:"snapshot_id"`
	TerminalApp      string            `json:"terminal_app" db:"terminal_app"`
	WorkingDirectory string            `json:"working_directory" db:"working_directory"`
	ActiveCommand    string            `json:"active_command" db:"active_command"`
	ShellType        string            `json:"shell_type" db:"shell_type"`
	EnvVars          map[string]string `json:"env_vars" db:"env_vars"` // Stored as JSON
}

// BrowserTab represents a browser tab
type BrowserTab struct {
	ID          int64  `json:"id" db:"id"`
	SnapshotID  string `json:"snapshot_id" db:"snapshot_id"`
	BrowserName string `json:"browser_name" db:"browser_name"`
	URL         string `json:"url" db:"url"`
	Title       string `json:"title" db:"title"`
	TabIndex    int    `json:"tab_index" db:"tab_index"`
	WindowIndex int    `json:"window_index" db:"window_index"`
	IsPinned    bool   `json:"is_pinned" db:"is_pinned"`
}

// Process represents a background process
type Process struct {
	ID               int64  `json:"id" db:"id"`
	SnapshotID       string `json:"snapshot_id" db:"snapshot_id"`
	ProcessName      string `json:"process_name" db:"process_name"`
	Command          string `json:"command" db:"command"`
	WorkingDirectory string `json:"working_directory" db:"working_directory"`
	Pid              int    `json:"pid" db:"pid"`
	AutoRestart      bool   `json:"auto_restart" db:"auto_restart"`
}

// IDEFile represents an open file in an editor
type IDEFile struct {
	ID           int64  `json:"id" db:"id"`
	SnapshotID   string `json:"snapshot_id" db:"snapshot_id"`
	IDEName      string `json:"ide_name" db:"ide_name"`
	FilePath     string `json:"file_path" db:"file_path"`
	CursorLine   int    `json:"cursor_line" db:"cursor_line"`
	CursorColumn int    `json:"cursor_column" db:"cursor_column"`
	IsActive     bool   `json:"is_active" db:"is_active"`
}
