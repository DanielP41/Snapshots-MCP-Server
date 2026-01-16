package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	cwd, _ := os.Getwd()
	serverPath := filepath.Join(cwd, "dev-env-snapshots.exe")

	fmt.Printf("--- STARTING ADVANCED TEST SUITE ---\n")
	fmt.Printf("Server Path: %s\n", serverPath)

	cmd := exec.Command(serverPath)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer cmd.Process.Kill()
	go io.Copy(os.Stderr, stderr)

	reader := bufio.NewReader(stdout)
	writer := json.NewEncoder(stdin)

	// 1. Initialize
	fmt.Println("\n[1] Protocol Handshake")
	call(writer, reader, 0, "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]string{"name": "TestSuite", "version": "2.0"},
	})

	// 2. Capture Snapshot A
	fmt.Println("\n[2] Tool: capture_snapshot (A)")
	respA := call(writer, reader, 1, "tools/call", map[string]interface{}{
		"name": "capture_snapshot",
		"arguments": map[string]interface{}{
			"name":        "Snapshot Alpha",
			"description": "First test point",
		},
	})
	idA := extractID(respA)
	fmt.Printf(">> Extracted ID A: %s\n", idA)

	// 3. Capture Snapshot B
	fmt.Println("\n[3] Tool: capture_snapshot (B)")
	respB := call(writer, reader, 2, "tools/call", map[string]interface{}{
		"name": "capture_snapshot",
		"arguments": map[string]interface{}{
			"name":        "Snapshot Beta",
			"description": "Second test point",
		},
	})
	idB := extractID(respB)
	fmt.Printf(">> Extracted ID B: %s\n", idB)

	// 4. Diff Snapshots
	fmt.Println("\n[4] Tool: diff_snapshots (A vs B)")
	call(writer, reader, 3, "tools/call", map[string]interface{}{
		"name": "diff_snapshots",
		"arguments": map[string]interface{}{
			"source_id": idA,
			"target_id": idB,
		},
	})

	// 5. Restore Report (Validation only)
	fmt.Println("\n[5] Tool: restore_snapshot (Report/Dry Mode)")
	// Note: Our current restore tool doesn't expose dryRun to MCP arguments yet,
	// but it returns a report. We test it here.
	call(writer, reader, 4, "tools/call", map[string]interface{}{
		"name": "restore_snapshot",
		"arguments": map[string]interface{}{
			"snapshot_id": idA,
		},
	})

	// 6. Delete
	fmt.Println("\n[6] Tool: delete_snapshot")
	call(writer, reader, 5, "tools/call", map[string]interface{}{
		"name": "delete_snapshot",
		"arguments": map[string]interface{}{
			"snapshot_id": idB,
		},
	})

	fmt.Println("\n--- TEST SUITE FINISHED ---")
}

func call(w *json.Encoder, r *bufio.Reader, id int, method string, params interface{}) string {
	pBytes, _ := json.Marshal(params)
	req := Request{JSONRPC: "2.0", ID: id, Method: method, Params: pBytes}
	w.Encode(req)

	var raw json.RawMessage
	decoder := json.NewDecoder(r)
	decoder.Decode(&raw)
	output := string(raw)
	fmt.Printf("<< %s\n", output)
	return output
}

func extractID(resp string) string {
	// Simple dirty extraction for "ID: <uuid>"
	if idx := strings.Index(resp, "ID: "); idx != -1 {
		end := strings.Index(resp[idx:], ",")
		if end == -1 {
			// Try end of line or quote
			end = strings.Index(resp[idx:], "\"")
			if end == -1 {
				return "unknown"
			}
		}
		id := resp[idx+4 : idx+end]
		return strings.Trim(id, " \n\r\"")
	}
	return "unknown"
}
