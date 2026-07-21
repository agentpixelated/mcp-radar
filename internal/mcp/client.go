package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/agentpixelated/mcp-radar/internal/lockfile"
)

const DefaultProtocolVersion = "2025-11-25"

type Options struct {
	Command         string
	Args            []string
	Env             []string
	EnvironmentKeys []string
	ProtocolVersion string
	Timeout         time.Duration
}

type client struct {
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	stderr  *limitedBuffer
	nextID  int64
}

type envelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

type initializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      map[string]any `json:"serverInfo"`
	Instructions    string         `json:"instructions,omitempty"`
}

// Capture launches a stdio MCP server, performs the lifecycle handshake, lists
// advertised behavior, and returns a deterministic lockfile model.
func Capture(parent context.Context, options Options) (*lockfile.Lockfile, error) {
	if options.Command == "" {
		return nil, errors.New("server command is required")
	}
	if options.ProtocolVersion == "" {
		options.ProtocolVersion = DefaultProtocolVersion
	}
	if options.Timeout <= 0 {
		options.Timeout = 15 * time.Second
	}

	ctx, cancel := context.WithTimeout(parent, options.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, options.Command, options.Args...)
	if options.Env != nil {
		cmd.Env = options.Env
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open server stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open server stdout: %w", err)
	}
	stderr := &limitedBuffer{limit: 32 * 1024}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start MCP server: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	c := &client{stdin: stdin, scanner: scanner, stderr: stderr}
	defer c.shutdown(cmd)

	var initialized initializeResult
	if err := c.call("initialize", map[string]any{
		"protocolVersion": options.ProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "mcp-radar",
			"version": "0.1.0",
		},
	}, &initialized); err != nil {
		return nil, c.wrap("initialize", err)
	}
	if err := c.notify("notifications/initialized", nil); err != nil {
		return nil, c.wrap("send initialized notification", err)
	}

	lock := &lockfile.Lockfile{
		LockfileVersion: lockfile.CurrentVersion,
		Transport:       "stdio",
		Invocation: lockfile.Invocation{
			Command:         options.Command,
			Args:            append([]string(nil), options.Args...),
			EnvironmentKeys: append([]string(nil), options.EnvironmentKeys...),
		},
		ProtocolVersion: initialized.ProtocolVersion,
		ServerInfo:      initialized.ServerInfo,
		Capabilities:    initialized.Capabilities,
		Instructions:    initialized.Instructions,
	}

	if _, ok := initialized.Capabilities["tools"]; ok {
		items, err := c.list("tools/list", "tools", "name")
		if err != nil {
			return nil, c.wrap("list tools", err)
		}
		lock.Tools = items
	}
	if _, ok := initialized.Capabilities["prompts"]; ok {
		items, err := c.list("prompts/list", "prompts", "name")
		if err != nil {
			return nil, c.wrap("list prompts", err)
		}
		lock.Prompts = items
	}
	if _, ok := initialized.Capabilities["resources"]; ok {
		items, err := c.list("resources/list", "resources", "uri")
		if err != nil {
			return nil, c.wrap("list resources", err)
		}
		lock.Resources = items
		templates, err := c.listOptional("resources/templates/list", "resourceTemplates", "uriTemplate")
		if err != nil {
			return nil, c.wrap("list resource templates", err)
		}
		lock.ResourceTemplates = templates
	}
	if err := lock.Finalize(); err != nil {
		return nil, err
	}
	return lock, nil
}

func (c *client) list(method, resultKey, identityKey string) ([]lockfile.Item, error) {
	return c.listPage(method, resultKey, identityKey, false)
}

func (c *client) listOptional(method, resultKey, identityKey string) ([]lockfile.Item, error) {
	return c.listPage(method, resultKey, identityKey, true)
}

func (c *client) listPage(method, resultKey, identityKey string, optional bool) ([]lockfile.Item, error) {
	var all []lockfile.Item
	cursor := ""
	for {
		params := map[string]any{}
		if cursor != "" {
			params["cursor"] = cursor
		}
		var result map[string]json.RawMessage
		err := c.call(method, params, &result)
		if err != nil {
			var rpcErr *rpcError
			if optional && errors.As(err, &rpcErr) && rpcErr.Code == -32601 {
				return nil, nil
			}
			return nil, err
		}
		var definitions []map[string]any
		if raw := result[resultKey]; len(raw) > 0 {
			if err := json.Unmarshal(raw, &definitions); err != nil {
				return nil, fmt.Errorf("decode %s: %w", resultKey, err)
			}
		}
		for index, definition := range definitions {
			name := stringValue(definition[identityKey])
			if name == "" {
				name = stringValue(definition["name"])
			}
			if name == "" {
				name = fmt.Sprintf("unnamed-%d", len(all)+index+1)
			}
			all = append(all, lockfile.Item{Name: name, Definition: definition})
		}
		cursor = ""
		if raw := result["nextCursor"]; len(raw) > 0 {
			_ = json.Unmarshal(raw, &cursor)
		}
		if cursor == "" {
			break
		}
	}
	return all, nil
}

func (c *client) call(method string, params any, target any) error {
	c.nextID++
	id := c.nextID
	message := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		message["params"] = params
	}
	if err := c.write(message); err != nil {
		return err
	}

	for c.scanner.Scan() {
		line := bytes.TrimSpace(c.scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var incoming envelope
		if err := json.Unmarshal(line, &incoming); err != nil {
			return fmt.Errorf("server stdout contained invalid JSON-RPC: %w", err)
		}
		if incoming.Method != "" && len(incoming.ID) > 0 {
			if err := c.write(map[string]any{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(incoming.ID),
				"error": map[string]any{
					"code":    -32601,
					"message": "Method not supported by mcp-radar client",
				},
			}); err != nil {
				return err
			}
			continue
		}
		if strings.TrimSpace(string(incoming.ID)) != strconv.FormatInt(id, 10) {
			continue
		}
		if incoming.Error != nil {
			return incoming.Error
		}
		if target != nil {
			if err := json.Unmarshal(incoming.Result, target); err != nil {
				return fmt.Errorf("decode %s result: %w", method, err)
			}
		}
		return nil
	}
	if err := c.scanner.Err(); err != nil {
		return fmt.Errorf("read server response: %w", err)
	}
	return errors.New("server closed stdout before replying")
}

func (c *client) notify(method string, params any) error {
	message := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		message["params"] = params
	}
	return c.write(message)
}

func (c *client) write(value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if _, err := c.stdin.Write(encoded); err != nil {
		return fmt.Errorf("write server request: %w", err)
	}
	return nil
}

func (c *client) wrap(action string, err error) error {
	if log := strings.TrimSpace(c.stderr.String()); log != "" {
		return fmt.Errorf("%s: %w\nserver stderr:\n%s", action, err, log)
	}
	return fmt.Errorf("%s: %w", action, err)
}

func (c *client) shutdown(cmd *exec.Cmd) {
	_ = c.stdin.Close()
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(750 * time.Millisecond):
		_ = cmd.Process.Kill()
		<-done
	}
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

type limitedBuffer struct {
	buf   bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := b.limit - b.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.buf.Write(p)
	}
	return original, nil
}

func (b *limitedBuffer) String() string { return b.buf.String() }

func EnvKeys(overrides []string) []string {
	keys := make([]string, 0, len(overrides))
	seen := map[string]bool{}
	for _, override := range overrides {
		key, _, ok := strings.Cut(override, "=")
		if ok && key != "" && !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
