package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/tuusuario/dev-env-snapshots/internal/core"
	"github.com/tuusuario/dev-env-snapshots/internal/db"
	"github.com/tuusuario/dev-env-snapshots/internal/platform"
	"github.com/tuusuario/dev-env-snapshots/internal/server"
	"github.com/tuusuario/dev-env-snapshots/internal/snapshot"
)

func main() {
	// 1. Setup DB
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	dbPath := filepath.Join(home, ".dev-env-snapshots", "snapshots.db")

	database, err := db.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	repo := db.NewRepository(database)

	// 2. Setup Platform Adapter
	var adapter core.PlatformAdapter
	if os.Getenv("USE_MOCK") == "1" {
		adapter = platform.NewMockAdapter()
	} else {
		// Automatically select the platform adapter.
		// Detailed implementation of windows.go allows native execution on Windows.
		// Note: Build tags would be used in a cross-compilation setup.
		// Current assumption: Running on Windows.
		adapter = platform.NewWindowsAdapter()
		log.Println("Using Windows Adapter V2 (Renamed to Canonical)")
	}

	// 3. Setup Logic
	manager := snapshot.NewManager(repo, adapter)

	// 4. Start MCP Server
	mcpServer := server.NewMCPServer(manager)

	log.Printf("Starting Dev Environment Snapshots MCP Server... DB: %s", dbPath)
	if err := mcpServer.Start(); err != nil {
		log.Fatal(err)
	}
}
