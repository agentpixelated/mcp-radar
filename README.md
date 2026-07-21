# mcp-radar

**Know exactly what changed before trusting an updated MCP server.**

`mcp-radar` creates behavior lockfiles for Model Context Protocol servers. It launches a stdio server, performs the MCP initialization lifecycle, records its advertised capabilities and schemas, and fails CI when that behavior drifts.

> Status: experimental v0.1. Review the lockfile and risk report; this tool does not sandbox tool execution or prove a server is safe.

## Why

An MCP package can keep the same name while adding a new tool, broadening an input schema, changing its instructions, or exposing new resources. Traditional dependency lockfiles pin package bytes. `mcp-radar` locks the interface your agent is being asked to trust.

## Install

```bash
go install github.com/agentpixelated/mcp-radar/cmd/mcp-radar@latest
```

Or build from source:

```bash
go build -o mcp-radar ./cmd/mcp-radar
```

## Quick start

Create an approved behavior snapshot:

```bash
mcp-radar approve -- npx -y @modelcontextprotocol/server-filesystem "$PWD"
```

Verify it later using the invocation stored in the lockfile:

```bash
mcp-radar verify
```

A clean verification exits `0`. Drift exits `1`, making it suitable for CI.

## See the drift detector locally

This repository includes a tiny mock MCP server:

```bash
go build -o ./mcp-radar ./cmd/mcp-radar
go build -o ./mock-server ./examples/mock-server

./mcp-radar snapshot --lock v1.lock.json -- ./mock-server --behavior v1
./mcp-radar snapshot --lock v2.lock.json -- ./mock-server --behavior v2
./mcp-radar diff v1.lock.json v2.lock.json
```

Expected report:

```text
⚠ MCP behavior drift detected

Risk: HIGH
Summary: 1 added, 0 removed, 0 changed

Tools
+ run_command [high 8/10] network-access, process-execution
```

## Commands

```text
mcp-radar snapshot [flags] -- <server-command> [args...]
mcp-radar approve  [flags] -- <server-command> [args...]
mcp-radar verify   [flags] [-- <server-command> [args...]]
mcp-radar diff <approved.lock.json> <candidate.lock.json>
```

- `snapshot` writes a candidate lockfile.
- `approve` writes the trusted baseline.
- `verify` captures current behavior and compares it with the baseline.
- `diff` compares two existing lockfiles without launching a server.

Useful flags:

```text
--lock PATH          lockfile path (default mcp-radar.lock.json)
--protocol VERSION   requested MCP protocol version (default 2025-11-25)
--timeout DURATION   overall discovery timeout (default 15s)
--env KEY=VALUE      child-process environment override; repeatable
```

Environment values are never stored in the lockfile.

## What v0.1 fingerprints

- negotiated protocol version;
- server metadata, capabilities, and instructions;
- tool names, descriptions, input/output schemas, annotations, and metadata;
- prompts;
- resources and resource templates;
- deterministic per-item SHA-256 hashes and an aggregate fingerprint;
- explainable tool-risk categories.

The server's advertised lists are discovered but tools are **not invoked**.

## CI example

Commit `mcp-radar.lock.json`, install the CLI, and verify:

```yaml
- name: Install mcp-radar
  run: go install github.com/agentpixelated/mcp-radar/cmd/mcp-radar@latest

- name: Verify MCP behavior
  run: mcp-radar verify
```

For servers that need secrets, inject them in CI and pass them using `--env`; only key names are recorded.

## Security model and limits

`mcp-radar` detects changes in advertised MCP behavior. It does not currently observe filesystem access, network calls, subprocesses, runtime side effects, or behavior hidden behind unchanged schemas. Those runtime controls are planned for a later sandboxed release.

Risk scores are deterministic review hints, not vulnerability verdicts.

See [the lockfile specification](docs/lockfile.md).

## Roadmap

- v0.1: stdio discovery, deterministic lockfiles, diff, verify, risk hints
- v0.2: sandboxed runtime canaries for filesystem, process, environment, and network access
- v0.3: reusable GitHub Action and review annotations
- v0.4: signed attestations and organization policy

## License

MIT
