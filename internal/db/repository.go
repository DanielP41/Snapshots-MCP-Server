package db

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/tuusuario/dev-env-snapshots/internal/core"
)

type SQLiteRepository struct {
	db *DB
}

func NewRepository(db *DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// Unmarshal helper
func unmarshalJSON(data string, v interface{}) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), v)
}

// Marshal helper
func marshalJSON(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *SQLiteRepository) CreateSnapshot(ctx context.Context, s *core.Snapshot) error {
	tagsJSON, err := marshalJSON(s.Tags)
	if err != nil {
		return err
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		query := `
			INSERT INTO snapshots (id, name, description, git_branch, git_repo, git_dirty, git_head_hash, tags)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := tx.ExecContext(ctx, query, s.ID, s.Name, s.Description, s.GitBranch, s.GitRepo, s.GitDirty, s.GitHeadHash, tagsJSON)
		if err != nil {
			return err
		}
		return nil
	})
}

func (r *SQLiteRepository) GetSnapshotByID(ctx context.Context, id string) (*core.Snapshot, error) {
	query := `SELECT id, name, description, created_at, updated_at, git_branch, git_repo, git_dirty, tags FROM snapshots WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	s := &core.Snapshot{}
	var tagsRaw string
	err := row.Scan(&s.ID, &s.Name, &s.Description, &s.CreatedAt, &s.UpdatedAt, &s.GitBranch, &s.GitRepo, &s.GitDirty, &tagsRaw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := unmarshalJSON(tagsRaw, &s.Tags); err != nil {
		return nil, err
	}

	return s, nil
}

func (r *SQLiteRepository) ListSnapshots(ctx context.Context, filter core.SnapshotFilter) ([]core.Snapshot, error) {
	query := `SELECT id, name, description, created_at, updated_at, git_branch, git_repo, git_dirty, tags FROM snapshots WHERE 1=1`
	var args []interface{}

	if filter.Project != "" {
		query += " AND git_repo LIKE ?"
		args = append(args, "%"+filter.Project+"%")
	}
	if filter.Branch != "" {
		query += " AND git_branch = ?"
		args = append(args, filter.Branch)
	}
	// Note: Tags filtering in SQLite with JSON text is limited; skipping for MVP or doing simple like

	query += " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []core.Snapshot
	for rows.Next() {
		s := core.Snapshot{}
		var tagsRaw string
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.CreatedAt, &s.UpdatedAt, &s.GitBranch, &s.GitRepo, &s.GitDirty, &tagsRaw); err != nil {
			return nil, err
		}
		unmarshalJSON(tagsRaw, &s.Tags)
		snapshots = append(snapshots, s)
	}

	return snapshots, nil
}

func (r *SQLiteRepository) DeleteSnapshot(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM snapshots WHERE id = ?", id)
	return err
}

func (r *SQLiteRepository) SaveWindows(ctx context.Context, snapshotID string, windows []core.Window) error {
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO windows (snapshot_id, app_name, app_path, window_title, x, y, width, height, state, workspace, z_index, launch_args)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, w := range windows {
			argsLabel, _ := marshalJSON(w.LaunchArgs)
			_, err := stmt.ExecContext(ctx, snapshotID, w.AppName, w.AppPath, w.WindowTitle, w.X, w.Y, w.Width, w.Height, w.State, w.Workspace, w.ZIndex, argsLabel)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *SQLiteRepository) SaveTerminals(ctx context.Context, snapshotID string, terminals []core.Terminal) error {
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO terminals (snapshot_id, terminal_app, working_directory, active_command, shell_type, env_vars)
			VALUES (?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, t := range terminals {
			envJSON, _ := marshalJSON(t.EnvVars)
			_, err := stmt.ExecContext(ctx, snapshotID, t.TerminalApp, t.WorkingDirectory, t.ActiveCommand, t.ShellType, envJSON)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *SQLiteRepository) SaveBrowserTabs(ctx context.Context, snapshotID string, tabs []core.BrowserTab) error {
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO browser_tabs (snapshot_id, browser_name, url, title, tab_index, window_index, is_pinned)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, t := range tabs {
			_, err := stmt.ExecContext(ctx, snapshotID, t.BrowserName, t.URL, t.Title, t.TabIndex, t.WindowIndex, t.IsPinned)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *SQLiteRepository) SaveIDEFiles(ctx context.Context, snapshotID string, files []core.IDEFile) error {
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO ide_files (snapshot_id, ide_name, file_path, cursor_line, cursor_column, is_active)
			VALUES (?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, f := range files {
			_, err := stmt.ExecContext(ctx, snapshotID, f.IDEName, f.FilePath, f.CursorLine, f.CursorColumn, f.IsActive)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *SQLiteRepository) GetWindows(ctx context.Context, snapshotID string) ([]core.Window, error) {
	query := `SELECT id, snapshot_id, app_name, app_path, window_title, x, y, width, height, state, workspace, z_index, launch_args FROM windows WHERE snapshot_id = ?`
	rows, err := r.db.QueryContext(ctx, query, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var windows []core.Window
	for rows.Next() {
		w := core.Window{}
		var argsRaw string
		if err := rows.Scan(&w.ID, &w.SnapshotID, &w.AppName, &w.AppPath, &w.WindowTitle, &w.X, &w.Y, &w.Width, &w.Height, &w.State, &w.Workspace, &w.ZIndex, &argsRaw); err != nil {
			return nil, err
		}
		if argsRaw != "" {
			w.LaunchArgs = json.RawMessage(argsRaw)
		}
		windows = append(windows, w)
	}
	return windows, nil
}
