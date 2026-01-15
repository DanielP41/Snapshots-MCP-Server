package server

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuusuario/dev-env-snapshots/internal/snapshot"
)

type MCPServer struct {
	manager *snapshot.Manager
	server  *server.MCPServer
}

func NewMCPServer(manager *snapshot.Manager) *MCPServer {
	s := server.NewMCPServer(
		"Dev Environment Snapshots",
		"1.0.0",
		server.WithLogging(),
	)

	m := &MCPServer{
		manager: manager,
		server:  s,
	}

	m.registerTools()
	return m
}

func (s *MCPServer) Start() error {
	// stdio transport
	return server.ServeStdio(s.server)
}

func (s *MCPServer) registerTools() {
	// capture_snapshot
	s.server.AddTool(mcp.NewTool("capture_snapshot",
		mcp.WithDescription("Captures the current development environment state"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the snapshot")),
		mcp.WithString("description", mcp.Description("Description")),
	), s.handleCaptureSnapshot)

	// restore_snapshot
	s.server.AddTool(mcp.NewTool("restore_snapshot",
		mcp.WithDescription("Restores a previously captured snapshot"),
		mcp.WithString("snapshot_id", mcp.Required(), mcp.Description("ID of the snapshot to restore")),
	), s.handleRestoreSnapshot)

	// list_snapshots
	s.server.AddTool(mcp.NewTool("list_snapshots",
		mcp.WithDescription("Lists available snapshots"),
	), s.handleListSnapshots)

	// delete_snapshot
	s.server.AddTool(mcp.NewTool("delete_snapshot",
		mcp.WithDescription("Deletes a snapshot by ID"),
		mcp.WithString("snapshot_id", mcp.Required(), mcp.Description("ID of the snapshot to delete")),
	), s.handleDeleteSnapshot)

	// diff_snapshots
	s.server.AddTool(mcp.NewTool("diff_snapshots",
		mcp.WithDescription("Diffs two snapshots"),
		mcp.WithString("source_id", mcp.Required(), mcp.Description("Source Snapshot ID")),
		mcp.WithString("target_id", mcp.Required(), mcp.Description("Target Snapshot ID")),
	), s.handleDiffSnapshots)
}

func (s *MCPServer) handleCaptureSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var name, desc string
	if request.Params.Arguments != nil {
		if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if v, ok := args["name"].(string); ok {
				name = v
			}
			if v, ok := args["description"].(string); ok {
				desc = v
			}
		}
	}

	snap, err := s.manager.Capture(ctx, snapshot.CaptureOptions{
		Name:        name,
		Description: desc,
		// Defaults
		IncludeBrowsable: true,
		IncludeTerminals: true,
		Sanitize:         true,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to capture: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Snapshot captured successfully! ID: %s, Name: %s", snap.ID, snap.Name)), nil
}

func (s *MCPServer) handleRestoreSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var id string
	if request.Params.Arguments != nil {
		if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
			id, _ = args["snapshot_id"].(string)
		}
	}

	report, err := s.manager.Restore(ctx, id, snapshot.RestoreOptions{
		ValidateBeforeRestore: false, // Default false for basic restore tool
		SkipMissingApps:       true,
		DryRun:                false,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to restore: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Restore Completed: %s", report.Message)), nil
}

func (s *MCPServer) handleListSnapshots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	snaps, err := s.manager.List(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list snapshots: %v", err)), nil
	}

	// Format as JSON or Table
	// Simple text list for now
	var result string
	for _, snap := range snaps {
		result += fmt.Sprintf("- [%s] %s (%s)\n", snap.ID, snap.Name, snap.CreatedAt.Format(time.RFC822))
	}
	if result == "" {
		result = "No snapshots found."
	}

	return mcp.NewToolResultText(result), nil
}

func (s *MCPServer) handleDeleteSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var id string
	if request.Params.Arguments != nil {
		if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
			id, _ = args["snapshot_id"].(string)
		}
	}

	err := s.manager.Delete(ctx, id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Snapshot %s deleted successfully", id)), nil
}

func (s *MCPServer) handleDiffSnapshots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var id1, id2 string
	if request.Params.Arguments != nil {
		if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
			id1, _ = args["source_id"].(string)
			id2, _ = args["target_id"].(string)
		}
	}

	diff, err := s.manager.Diff(ctx, id1, id2)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to diff: %v", err)), nil
	}

	result := fmt.Sprintf("Diff between %s and %s:\n", diff.SourceID, diff.TargetID)
	if diff.GitChanged {
		result += "- Git Context Changed: Yes\n"
	} else {
		result += "- Git Context Changed: No\n"
	}
	result += fmt.Sprintf("- Common Windows: %d\n", diff.CommonWindows)

	if len(diff.AddedWindows) > 0 {
		result += "- Added Windows:\n"
		for _, w := range diff.AddedWindows {
			result += fmt.Sprintf("  + %s\n", w)
		}
	}
	if len(diff.RemovedWindows) > 0 {
		result += "- Removed Windows:\n"
		for _, w := range diff.RemovedWindows {
			result += fmt.Sprintf("  - %s\n", w)
		}
	}

	return mcp.NewToolResultText(result), nil
}
