# Lockfile format

`mcp-radar.lock.json` is a deterministic record of an MCP server's advertised behavior.

## Stability

The top-level `lockfileVersion` controls compatibility. Version `1` records:

- negotiated protocol version;
- server implementation metadata and capabilities;
- server instructions;
- tools, prompts, resources, and resource templates;
- a SHA-256 hash for every advertised item;
- an explainable risk assessment for tools;
- one aggregate behavior fingerprint;
- the stdio invocation needed to repeat verification.

The aggregate fingerprint intentionally excludes the local command path, arguments, and environment-key names. Moving an unchanged server binary does not create behavioral drift. Those invocation fields remain in the file so `mcp-radar verify` can rerun the server.

## Secrets

Values passed with `--env KEY=VALUE` are supplied to the child process but are never written to the lockfile. Only the key names are recorded. Environment variables inherited from the parent process are not enumerated.

## Determinism

Object keys are encoded deterministically, advertised collections are sorted by stable identity, and timestamps are not stored. Repeated snapshots of unchanged behavior should produce byte-for-byte identical files when the invocation is unchanged.

## Risk scores

Risk scores are deterministic keyword heuristics over the complete advertised tool definition. They help prioritize review; they are not vulnerability findings and do not prove that a tool is safe.
