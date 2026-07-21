package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestCaptureStdioServer(t *testing.T) {
	lock, err := Capture(context.Background(), Options{
		Command:         os.Args[0],
		Args:            []string{"-test.run=TestHelperProcess", "--", "v2"},
		Env:             append(os.Environ(), "GO_WANT_MCP_HELPER=1"),
		EnvironmentKeys: []string{"GO_WANT_MCP_HELPER"},
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if lock.ProtocolVersion != DefaultProtocolVersion {
		t.Fatalf("unexpected protocol version: %s", lock.ProtocolVersion)
	}
	if len(lock.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(lock.Tools))
	}
	if lock.Tools[1].Name != "run_command" {
		t.Fatalf("expected sorted run_command tool, got %#v", lock.Tools)
	}
	if lock.Tools[1].Risk == nil || lock.Tools[1].Risk.Score < 6 {
		t.Fatalf("expected high-risk command tool, got %#v", lock.Tools[1].Risk)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MCP_HELPER") != "1" {
		return
	}
	mode := "v1"
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			mode = os.Args[i+1]
		}
	}
	serveFixture(mode)
	os.Exit(0)
}

func serveFixture(mode string) {
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var request map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			continue
		}
		method, _ := request["method"].(string)
		id, hasID := request["id"]
		if !hasID {
			continue
		}
		switch method {
		case "initialize":
			params, _ := request["params"].(map[string]any)
			version, _ := params["protocolVersion"].(string)
			_ = encoder.Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": version,
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo":      map[string]any{"name": "fixture", "version": mode},
				},
			})
		case "tools/list":
			tools := []map[string]any{{
				"name":        "read_file",
				"description": "Read a file path from the filesystem",
				"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}},
			}}
			if mode == "v2" {
				tools = append(tools, map[string]any{
					"name":        "run_command",
					"description": "Execute a shell command on a remote host",
					"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"command": map[string]any{"type": "string"}}},
				})
			}
			_ = encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"tools": tools}})
		default:
			_ = encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32601, "message": fmt.Sprintf("unknown method %s", method)}})
		}
	}
}
