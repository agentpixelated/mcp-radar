# mcp-radar

Discover, compare, and evaluate Model Context Protocol (MCP) servers with intelligent scoring and practical recommendations.

> [!NOTE]
> `mcp-radar` is the primary project. Experimental developer tooling is isolated under `plugins/` and documented independently.

## Repository status

This repository is still early-stage. The MCP discovery product and the experiments below should be treated as works in progress rather than production releases.

## Repository map

| Path | Status | Purpose |
| --- | --- | --- |
| [`plugins/hermes_exec_fuse`](plugins/hermes_exec_fuse) | Alpha | Native Hermes Agent command batching, safe result reuse, duplicate-call prevention, and output compaction. |
| [`tests`](tests) | Active | Automated coverage for incubating experiments. |
| [`.github/workflows/ci.yml`](.github/workflows/ci.yml) | Active | Linting and test matrix for Python 3.10–3.12. |

## Labs

### Hermes Exec Fuse

Hermes Exec Fuse is a native Hermes Agent plugin designed for agents that repeatedly inspect the same workspace. It batches independent foreground commands, runs eligible read-only work concurrently, reuses exact safe results, and keeps oversized terminal output from consuming unnecessary context.

It is **not an MCP server** and does not replace Hermes' built-in terminal. Every real command is delegated through Hermes' normal tool pipeline.

- [Plugin documentation](plugins/hermes_exec_fuse/README.md)
- [Architecture](plugins/hermes_exec_fuse/ARCHITECTURE.md)
- [Changelog](plugins/hermes_exec_fuse/CHANGELOG.md)
- [Contributing](plugins/hermes_exec_fuse/CONTRIBUTING.md)

## License

Released under the [MIT License](LICENSE).