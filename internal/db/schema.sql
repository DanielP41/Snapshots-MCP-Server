-- Tabla principal de snapshots
CREATE TABLE IF NOT EXISTS snapshots (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    git_branch TEXT,
    git_repo TEXT,
    git_dirty BOOLEAN,
    git_head_hash TEXT,
    tags TEXT -- JSON array
);

-- Ventanas capturadas
CREATE TABLE IF NOT EXISTS windows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    snapshot_id TEXT NOT NULL,
    app_name TEXT NOT NULL,
    app_path TEXT,
    window_title TEXT,
    x INTEGER,
    y INTEGER,
    width INTEGER,
    height INTEGER,
    state TEXT, -- normal, maximized, minimized, fullscreen
    workspace INTEGER,
    z_index INTEGER,
    launch_args TEXT, -- JSON
    FOREIGN KEY (snapshot_id) REFERENCES snapshots(id) ON DELETE CASCADE
);

-- Sesiones de terminal
CREATE TABLE IF NOT EXISTS terminals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    snapshot_id TEXT NOT NULL,
    terminal_app TEXT,
    working_directory TEXT,
    active_command TEXT,
    shell_type TEXT,
    env_vars TEXT, -- JSON
    FOREIGN KEY (snapshot_id) REFERENCES snapshots(id) ON DELETE CASCADE
);

-- Tabs de navegador
CREATE TABLE IF NOT EXISTS browser_tabs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    snapshot_id TEXT NOT NULL,
    browser_name TEXT,
    url TEXT,
    title TEXT,
    tab_index INTEGER,
    window_index INTEGER,
    is_pinned BOOLEAN,
    FOREIGN KEY (snapshot_id) REFERENCES snapshots(id) ON DELETE CASCADE
);

-- Procesos en background
CREATE TABLE IF NOT EXISTS processes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    snapshot_id TEXT NOT NULL,
    process_name TEXT,
    command TEXT,
    working_directory TEXT,
    pid INTEGER,
    auto_restart BOOLEAN,
    FOREIGN KEY (snapshot_id) REFERENCES snapshots(id) ON DELETE CASCADE
);

-- Archivos abiertos en IDE
CREATE TABLE IF NOT EXISTS ide_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    snapshot_id TEXT NOT NULL,
    ide_name TEXT,
    file_path TEXT,
    cursor_line INTEGER,
    cursor_column INTEGER,
    is_active BOOLEAN,
    FOREIGN KEY (snapshot_id) REFERENCES snapshots(id) ON DELETE CASCADE
);
