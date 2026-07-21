# MCP Radar

[![CI](https://github.com/agentpixelated/mcp-radar/actions/workflows/ci.yml/badge.svg)](https://github.com/agentpixelated/mcp-radar/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Status: experimental](https://img.shields.io/badge/status-experimental-orange.svg)](#project-status)

> **Behavior lockfiles and supply-chain drift detection for MCP servers.**

MCP Radar snapshots the behavior an MCP server advertises, stores it in a deterministic lockfile, and tells you exactly what changed after an update.

```text
MCP server ──snapshot──> mcp-radar.lock.json
     │                         │
     └────── update ───────────┘
                               │
                         verify / diff
                               │
                    clean ✓  or  drift ⚠
```

Use it to review new tools, broader schemas, changed instructions, added resources, and other interface changes **before** an updated server reaches your agent or CI environment.

> [!IMPORTANT]
> MCP Radar v0.1 fingerprints advertised MCP behavior. It does not execute tools, sandbox the server, or prove that a server is safe.

## Why MCP Radar?

Package versions and checksums tell you which bytes changed. They do not explain how the interface exposed to your agent changed.

An MCP server can keep the same package name while it:

- adds a command-execution tool;
- broadens a filesystem path schema;
- changes tool descriptions or server instructions;
- exposes new prompts, resources, or templates;
- negotiates a different protocol or capability set.

MCP Radar turns those changes into a reviewable, version-controlled artifact.

## Quick start

Install the CLI:

```bash
go install github.com/agentpixelated/mcp-radar/cmd/mcp-radar@latest
```

Approve the current behavior of a stdio MCP server:

```bash
mcp-radar approve -- npx -y @modelcontextprotocol/server-filesystem "$PWD"
```

Commit the generated baseline:

```text
mcp-radar.lock.json
```

Verify the server later using the invocation stored in that lockfile:

```bash
mcp-radar verify
```

A clean verification exits `0`. Behavior drift exits `1`, which makes the command CI-friendly.

## What it detects

| Surface | Examples of detected drift |
| --- | --- |
| Server | protocol version, implementation metadata, capabilities, instructions |
| Tools | added or removed tools, changed descriptions, schemas, annotations, metadata |
| Prompts | additions, removals, and definition changes |
| Resources | additions, removals, URI or metadata changes |
| Resource templates | additions, removals, and definition changes |
| Risk hints | deterministic categories for tools that advertise sensitive behavior |

Every advertised item receives a SHA-256 hash. MCP Radar also produces one aggregate behavior fingerprint for the server.

## Example drift report

This repository includes a small mock MCP server for local testing:

```bash
go build -o ./mcp-radar ./cmd/mcp-radar
go build -o ./mock-server ./examples/mock-server

./mcp-radar snapshot --lock v1.lock.json -- ./mock-server --behavior v1
./mcp-radar snapshot --lock v2.lock.json -- ./mock-server --behavior v2
./mcp-radar diff v1.lock.json v2.lock.json
```

Expected output:

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

- `snapshot` writes a candidate lockfile without treating it as approved.
- `approve` writes or replaces the trusted baseline.
- `verify` snapshots current behavior and compares it with the approved baseline.
- `diff` compares two existing lockfiles without launching a server.

Common flags:

```text
--lock PATH          lockfile path (default: mcp-radar.lock.json)
--protocol VERSION   requested MCP protocol version (default: 2025-11-25)
--timeout DURATION   overall discovery timeout (default: 15s)
--env KEY=VALUE      child-process environment override; repeatable
```

Environment **values** are never stored in the lockfile. Only explicitly supplied key names are recorded so the server can be launched consistently.

## Exit codes

| Code | Meaning |
| ---: | --- |
| `0` | verification is clean, or the requested operation completed successfully |
| `1` | behavior drift was detected |
| `2` | the command could not complete because of invalid input, launch failure, timeout, or another operational error |

## CI usage

Commit `mcp-radar.lock.json` beside your project, install MCP Radar, and verify the server during pull requests:

```yaml
- name: Install MCP Radar
  run: go install github.com/agentpixelated/mcp-radar/cmd/mcp-radar@latest

- name: Verify MCP behavior
  run: mcp-radar verify
```

For servers that require credentials, inject secrets through your CI provider and pass them with `--env`. Never commit secret values to the lockfile or workflow.

## Trust model

MCP Radar answers one focused question:

> **Did the behavior advertised through MCP change?**

It does **not** currently observe hidden runtime behavior such as filesystem reads, network requests, subprocess execution, environment access, or side effects behind an unchanged schema. See the [threat model](docs/threat-model.md) for the exact trust boundary.

Risk scores are deterministic review hints, not vulnerability verdicts.

## Lockfile design

The lockfile is designed to be:

- deterministic and suitable for version control;
- readable enough for human review;
- strict about incompatible format versions;
- free of timestamps and secret values;
- stable when only the local executable path changes.

See the [lockfile format](docs/lockfile.md) for field and compatibility details.

## Project status

MCP Radar is an experimental v0.1 project. The current release focuses on a small, auditable CLI and stdio MCP discovery.

Planned direction:

- **v0.1** — stdio discovery, deterministic lockfiles, diff, verify, and risk hints;
- **v0.2** — sandboxed runtime canaries for filesystem, process, environment, and network access;
- **v0.3** — reusable GitHub Action and pull-request annotations;
- **v0.4** — signed attestations and organization-level policy.

The roadmap describes intent, not guaranteed release dates.

## Contributing

Bug reports, threat-model feedback, compatibility findings, and focused pull requests are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a change.

For suspected vulnerabilities, follow [SECURITY.md](SECURITY.md) and avoid public disclosure until a fix is available.

## License

MCP Radar is available under the [MIT License](LICENSE).
