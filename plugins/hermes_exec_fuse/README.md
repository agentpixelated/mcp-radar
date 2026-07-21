# Hermes Exec Fuse

A native Hermes Agent plugin that reduces repeated terminal calls and tool-result tokens.

## What it does

- Adds `exec_fuse` for batching up to 24 commands in one tool call.
- Runs independent, conservatively classified read-only commands concurrently.
- Deduplicates exact read-only commands inside a batch.
- Reuses session-scoped read-only results for five minutes while the workspace generation is unchanged.
- Blocks repeated direct `terminal` calls and returns the cached compact result to the model.
- Invalidates cached reads after mutating or unknown terminal commands, `write_file`, `patch`, `execute_code`, or `skill_manage`.
- Compresses large outputs while retaining error, failure, warning, assertion, and success lines.
- Exposes `exec_fuse_stats` for execution and savings metrics.

Every actual command is delegated through `ctx.dispatch_tool("terminal", ...)`, so Hermes keeps its normal approval, credential, redaction, and execution pipelines.

## Install from this incubator

```bash
mkdir -p ~/.hermes/plugins/hermes_exec_fuse
cp -R plugins/hermes_exec_fuse/. ~/.hermes/plugins/hermes_exec_fuse/
hermes plugins enable hermes-exec-fuse
```

Restart Hermes and verify with:

```text
/plugins
```

## Example

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

For ordered work:

```json
{
  "commands": [
    {"id": "generate", "command": "python generate.py"},
    {"id": "inspect", "command": "git diff --stat", "depends_on": ["generate"]}
  ]
}
```

Unknown commands are allowed but never cached or deduplicated. Mutating commands are always sequential.

## Safety model

Classification is deliberately conservative. Only known read-only commands are eligible for caching or parallel execution. Shell redirection, command substitution, unknown interpreters, package operations, and mutating Git commands invalidate the workspace cache.

## Development

```bash
python -m pip install pytest ruff
ruff check plugins/hermes_exec_fuse tests
pytest
```
