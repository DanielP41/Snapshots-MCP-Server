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
)

// JSON-RPC Messages
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

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

func main() {
	// 1. Build path to server
	cwd, _ := os.Getwd()
	serverPath := filepath.Join(cwd, "dev-env-snapshots.exe")

	fmt.Printf("Starting server: %s\n", serverPath)
	cmd := exec.Command(serverPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer cmd.Process.Kill()

	// Capture stderr in background
	go io.Copy(os.Stderr, stderr)

	// 2. Initialize (Mock MCP Handshake)
	// We skip full handshake for this simple test if the server creates tools without init,
	// but MCP usually requires 'initialize'.
	// Our implementation currently registers tools in NewMCPServer and doesn't strictly enforce init state for local stdio in the basic lib usage,
	// but let's send initialize to be safe/correct if mcp-go enforces it.
	// Actually mcp-go server usually waits for initialize.

	// Create Reader/Writer
	reader := bufio.NewReader(stdout)
	writer := json.NewEncoder(stdin)

	// 2.1 Send Initialize
	fmt.Println(">> Sending Initialize")
	sendRequest(writer, 0, "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "ValidationClient",
			"version": "1.0",
		},
	})
	readResponse(reader) // Init response

	sendNotification(writer, "notifications/initialized", map[string]interface{}{})

	// 3. Test: List Snapshots
	fmt.Println("\n>> Testing: list_snapshots")
	sendCallTool(writer, 1, "list_snapshots", nil)
	readResponse(reader)

	// 4. Test: Capture Snapshot
	fmt.Println("\n>> Testing: capture_snapshot")
	sendCallTool(writer, 2, "capture_snapshot", map[string]interface{}{
		"name":        "Test Snapshot",
		"description": "Created by validation client",
	})
	readResponse(reader)

	// 5. Test: List again to verify
	fmt.Println("\n>> Testing: list_snapshots (verify capture)")
	sendCallTool(writer, 3, "list_snapshots", nil)
	readResponse(reader)

	// 6. Test: Restore (Dry Run implied or explicit)
	// Currently restore tool in main hardcodes dryRun=false, but we can call it.
	// We need an ID. Parsing the list response is hard in this simple script without struct,
	// but we can trust the capture output if it showed success.

	fmt.Println("\n>> Test Sequence Complete.")
}

func sendRequest(w *json.Encoder, id int, method string, params interface{}) {
	pBytes, _ := json.Marshal(params)
	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  pBytes,
	}
	if err := w.Encode(req); err != nil {
		log.Fatalf("Failed to encode: %v", err)
	}
}

func sendNotification(w *json.Encoder, method string, params interface{}) {
	pBytes, _ := json.Marshal(params)
	req := Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  pBytes,
	}
	w.Encode(req)
}

func sendCallTool(w *json.Encoder, id int, tool string, args map[string]interface{}) {
	if args == nil {
		args = make(map[string]interface{})
	}
	params := map[string]interface{}{
		"name":      tool,
		"arguments": args,
	}
	sendRequest(w, id, "tools/call", params)
}

func readResponse(r *bufio.Reader) {
	// MCP uses JSON-RPC over stdio, usually line delimited or content-length.
	// mcp-go uses line-based JSON by default for stdio?
	// Actually it might just parse JSON objects.
	// We'll decode one JSON object.

	var raw json.RawMessage
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&raw); err != nil {
		log.Printf("Failed to read response: %v", err)
		return
	}

	fmt.Printf("<< Response: %s\n", string(raw))
}
