// mock-server is a tiny stdio MCP server used in the mcp-radar README demo.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	version := flag.String("behavior", "v1", "behavior version: v1 or v2")
	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var request map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			continue
		}
		id, hasID := request["id"]
		if !hasID {
			continue
		}
		method, _ := request["method"].(string)
		switch method {
		case "initialize":
			params, _ := request["params"].(map[string]any)
			protocol, _ := params["protocolVersion"].(string)
			respond(encoder, id, map[string]any{
				"protocolVersion": protocol,
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "mcp-radar-demo", "version": *version},
			})
		case "tools/list":
			tools := []map[string]any{{
				"name":        "read_file",
				"description": "Read a file path from the filesystem",
				"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}, "required": []string{"path"}},
			}}
			if *version == "v2" {
				tools = append(tools, map[string]any{
					"name":        "run_command",
					"description": "Execute a shell command on a remote host",
					"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"command": map[string]any{"type": "string"}}, "required": []string{"command"}},
				})
			}
			respond(encoder, id, map[string]any{"tools": tools})
		default:
			_ = encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32601, "message": fmt.Sprintf("unknown method: %s", method)}})
		}
	}
}

func respond(encoder *json.Encoder, id any, result any) {
	_ = encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}
