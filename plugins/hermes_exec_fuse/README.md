# Hermes Exec Fuse

**Batch once. Reuse safe results. Keep terminal noise out of the agent context.**

Hermes Exec Fuse is a native [Hermes Agent](https://github.com/NousResearch/hermes-agent) plugin for reducing redundant shell work. It gives the model one structured tool for batching foreground commands, safely reusing exact read-only results, preventing duplicate terminal calls, and compacting large outputs before they return to the model.

> [!IMPORTANT]
> Status: **alpha / incubating**. The core behavior is tested, but the plugin still needs broader end-to-end validation across Hermes versions and terminal backends before a stable release.

## Why this exists

Tool-using agents often spend extra iterations and tokens on patterns such as:

```text
terminal("git status --short")
terminal("git diff --stat")
terminal("rg TODO")
terminal("git status --short")  # repeated
```

Hermes Exec Fuse provides a deterministic execution layer for this pattern:

```text
exec_fuse([status, diff, todos])
```

The plugin does not ask another model to optimize the commands. It applies conservative, inspectable rules for classification, scheduling, caching, invalidation, and output reduction.

## Highlights

| Capability | Behavior |
| --- | --- |
| Command batching | Accepts up to 24 foreground commands in one tool call. |
| Safe concurrency | Runs independent commands concurrently only when they are classified as read-only. |
| Intra-batch deduplication | Executes normalized exact read-only duplicates once and reuses the first result. |
| Session cache | Reuses successful read-only results for five minutes while the workspace generation is unchanged. |
| Duplicate terminal guard | Blocks a repeated direct `terminal` call and returns its cached compact result. |
| Dependency ordering | Supports `depends_on` for explicit command DAGs. |
| Conservative invalidation | Mutating and unknown operations clear cached workspace reads. |
| Output compaction | Preserves the beginning, ending, and important failure/success lines within a configurable character budget. |
| Metrics | Reports executions, cache hits, duplicate hits, avoided calls, and estimated characters saved. |
| Hermes-native execution | Delegates every actual command through `ctx.dispatch_tool("terminal", ...)`. |

Because commands still pass through Hermes, the plugin preserves the normal approval, credential, redaction, timeout, and terminal-backend behavior.

## Installation

### Requirements

- Python 3.10 or newer.
- A recent Hermes Agent build with general Python plugin and lifecycle-hook support.
- A trusted local checkout of this repository.

### Install from the repository checkout

```bash
mkdir -p ~/.hermes/plugins/hermes-exec-fuse
cp -R plugins/hermes_exec_fuse/. ~/.hermes/plugins/hermes-exec-fuse/
hermes plugins enable hermes-exec-fuse
```

Restart Hermes, then confirm that the plugin loaded:

```text
/plugins
```

For discovery diagnostics:

```bash
HERMES_PLUGINS_DEBUG=1 hermes plugins list
```

## Quick start

Ask Hermes to batch independent repository inspections:

```text
Inspect this repository efficiently. Use exec_fuse to batch git status,
git diff --stat, TODO search, and pytest test collection.
```

Equivalent tool arguments:

```json
{
  "commands": [
    {"id": "status", "command": "git status --short"},
    {"id": "diff", "command": "git diff --stat"},
    {"id": "todos", "command": "rg TODO"},
    {"id": "collect", "command": "pytest --collect-only -q"}
  ],
  "parallel": true,
  "cache": true,
  "max_output_chars": 4000
}
```

For ordered work, declare dependencies explicitly:

```json
{
  "commands": [
    {
      "id": "generate",
      "command": "python generate.py"
    },
    {
      "id": "inspect",
      "command": "git diff --stat",
      "depends_on": ["generate"]
    }
  ],
  "fail_fast": true
}
```

`generate` is classified as mutating/unknown and therefore runs sequentially. Its completion invalidates older workspace reads before `inspect` executes.

## Tools

### `exec_fuse`

Runs a batch of foreground terminal commands and returns compact structured results.

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `commands` | array | required | One to 24 command objects. IDs must be unique. |
| `parallel` | boolean | `true` | Run ready read-only commands concurrently, with at most eight workers. |
| `cache` | boolean | `true` | Enable session-scoped reuse for eligible read-only commands. |
| `fail_fast` | boolean | `false` | Skip commands that become ready after any earlier failure. Dependencies always skip when their prerequisite failed. |
| `max_output_chars` | integer | `4000` | Per-command compact-output budget, clamped to 500–20,000 characters. |

Each command object supports:

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | Short unique identifier used in dependencies and results. |
| `command` | string | yes | Foreground shell command delegated to Hermes' terminal tool. |
| `cwd` | string | no | Working directory applied with a safely quoted `cd -- ... &&` prefix. |
| `timeout` | integer | no | Foreground timeout in seconds, clamped to 1–600. |
| `depends_on` | string[] | no | Command IDs that must complete successfully first. |
| `cache` | boolean | no | Per-command opt-out from safe read-only result reuse. |

Result statuses are:

- `executed` — the terminal command ran.
- `cache_hit` — an existing session result was reused.
- `deduplicated` — the result of an earlier equivalent command in the same batch was reused.
- `skipped` — a dependency failed or `fail_fast` stopped later work.

### `exec_fuse_stats`

Returns session metrics including:

- workspace generation;
- active cache entries;
- executed commands;
- cache and duplicate hits;
- avoided terminal calls;
- raw versus returned character counts;
- estimated characters saved.

## Execution policy

| Classification | Parallel eligible | Cache eligible | Intra-batch deduplication | Invalidates workspace cache |
| --- | ---: | ---: | ---: | ---: |
| `read_only` | Yes | Yes | Yes | No |
| `mutating` | No | No | No | Yes |
| `unknown` | No | No | No | Yes |

The classifier is intentionally conservative. Examples currently recognized as read-only include common inspection utilities, safe Git queries, non-fixing linters, and test collection. Commands involving interpreters, package managers, network tools, shell redirection, command substitution, unknown Git actions, or ambiguous shell behavior are never cached or parallelized.

Unknown commands are still allowed. They simply run sequentially and invalidate the current workspace generation afterward.

## Cache and invalidation model

Cache identity includes:

```text
normalized command + cwd + relevant options + workspace generation
```

The cache is:

- scoped to the active Hermes task/session;
- held in memory only;
- limited to 128 entries per session;
- limited to 64 tracked sessions;
- expired after 300 seconds;
- cleared whenever the workspace generation changes.

Generation changes are triggered by mutating or unknown terminal commands and by selected Hermes tools that can change local state, including `write_file`, `patch`, `execute_code`, and `skill_manage`.

This model favors stale-result prevention over maximum cache hit rate.

## Direct terminal-call guard

The plugin also observes ordinary `terminal` calls. After a successful recognized read-only command has been seen, an identical non-background call in the same session and generation is blocked before execution. Hermes receives a cache-hit message containing the previously compacted result.

This is an exact normalized-command guard, not semantic equivalence. Commands that merely produce similar results are not merged.

## Output compaction

When a result exceeds its character budget, the deterministic compactor:

1. removes ANSI control sequences;
2. keeps the first 24 lines;
3. keeps up to 36 middle lines containing signals such as errors, failures, warnings, tracebacks, assertions, timeouts, passes, or successes;
4. keeps the final 18 lines;
5. inserts a compression marker and applies a final head/tail limit when necessary.

The full raw output is not persisted by this plugin.

## Safety properties

- No direct `subprocess`, `os.system`, or shell execution inside the plugin.
- Every real command is dispatched through Hermes' registered `terminal` tool.
- Mutating and unknown commands are serialized.
- Background commands are not supported by `exec_fuse`.
- Dependency cycles, duplicate IDs, unknown dependency IDs, and invalid batches return structured errors.
- Plugin hooks accept forward-compatible keyword arguments and avoid crashing the agent loop.

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full design.

## Known limitations

- Alpha software; compatibility has not yet been validated against every Hermes release or terminal backend.
- Classification is rule-based and will intentionally mark many safe-but-unrecognized commands as `unknown`.
- Cache state does not survive a Hermes process restart.
- Cache invalidation observes known mutation surfaces; external filesystem changes cannot be detected automatically.
- Direct duplicate prevention uses Hermes' `pre_tool_call` block response because plugin hooks cannot transparently replace a tool result.
- Interactive commands, streaming sessions, and background processes are outside the current scope.
- Character savings are estimates, not tokenizer-specific measurements.

## Development

```bash
python -m pip install --upgrade pytest ruff
ruff check plugins/hermes_exec_fuse tests
python -m compileall -q plugins/hermes_exec_fuse
pytest
```

Important contribution rules:

- Every classifier change needs tests for both the intended command and nearby unsafe variants.
- Commands must continue to flow through `ctx.dispatch_tool("terminal", ...)`.
- New mutation surfaces must invalidate the workspace generation.
- Output reduction must preserve diagnostic lines and deterministic behavior.

Read [CONTRIBUTING.md](CONTRIBUTING.md) before changing runtime behavior.

## Roadmap

- End-to-end tests against a real Hermes installation.
- Configurable TTL, cache size, and classifier extensions.
- Better terminal-result success detection across backends.
- Optional persisted metrics without persisted command output.
- Standalone repository and one-command installation.
- Benchmarks using real agent traces and tokenizer-aware savings.

## Project documents

- [Architecture](ARCHITECTURE.md)
- [Changelog](CHANGELOG.md)
- [Contributing](CONTRIBUTING.md)
- [Security](SECURITY.md)
- [Examples](examples)

## License

MIT. See the repository-level [LICENSE](../../LICENSE).