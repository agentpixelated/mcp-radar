# Changelog

All notable changes to MCP Radar will be documented in this file.

The project is currently experimental and has not published a stable release.

## Unreleased

### Added

- stdio MCP initialization and discovery;
- deterministic behavior lockfiles;
- `snapshot`, `approve`, `verify`, and `diff` commands;
- per-item and aggregate SHA-256 fingerprints;
- deterministic tool-risk hints;
- CI-friendly drift exit codes;
- mock MCP server and GitHub Actions validation;
- lockfile, threat-model, contribution, and security documentation.

### Known limitations

- stdio transport only;
- advertised behavior only; tools are not invoked;
- no runtime sandboxing or filesystem, network, process, or environment observation;
- no signed attestations or organization policy layer.
