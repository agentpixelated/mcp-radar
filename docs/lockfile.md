# Lockfile format

`mcp-radar.lock.json` is a deterministic record of the behavior advertised by one MCP server invocation.

The lockfile is intended to be committed to version control and reviewed like a dependency lockfile.

## Compatibility

The top-level `lockfileVersion` field controls format compatibility.

MCP Radar v0.1 reads and writes lockfile version `1`. A newer, unsupported version is rejected instead of being interpreted loosely.

## Top-level fields

| Field | Purpose |
| --- | --- |
| `lockfileVersion` | lockfile schema version |
| `transport` | transport used for discovery; currently `stdio` |
| `invocation` | command, arguments, and explicitly supplied environment-key names |
| `protocolVersion` | protocol version negotiated during initialization |
| `serverInfo` | implementation metadata returned by the server |
| `capabilities` | server capabilities returned during initialization |
| `instructions` | server instructions, when advertised |
| `tools` | canonicalized tool definitions, hashes, and risk hints |
| `prompts` | canonicalized prompt definitions and hashes |
| `resources` | canonicalized resource definitions and hashes |
| `resourceTemplates` | canonicalized resource-template definitions and hashes |
| `fingerprint` | aggregate SHA-256 behavior fingerprint |

## Item format

Tools, prompts, resources, and resource templates are stored as items with this shape:

```json
{
  "name": "read_file",
  "hash": "sha256:…",
  "definition": {
    "name": "read_file",
    "description": "Read a file",
    "inputSchema": {}
  }
}
```

Tools also contain a deterministic `risk` object. The risk assessment is derived from the complete advertised tool definition and is included to prioritize review.

## Fingerprinting rules

Before writing or comparing a lockfile, MCP Radar:

1. canonicalizes JSON object encoding;
2. computes each item hash from its complete `definition`;
3. sorts advertised collections by stable identity;
4. sorts explicitly supplied environment-key names;
5. computes the aggregate fingerprint from advertised behavior.

The aggregate fingerprint includes:

- transport;
- negotiated protocol version;
- server metadata and capabilities;
- server instructions;
- tools, prompts, resources, and resource templates.

It intentionally excludes the local command path, command arguments, and environment-key names. Moving an unchanged server binary does not therefore create behavioral drift. Invocation data remains in the file so `mcp-radar verify` can launch the same server again.

## Determinism

Lockfiles do not contain timestamps. Repeated snapshots of unchanged behavior should produce byte-for-byte identical files when the invocation is unchanged.

A lockfile may still change when the server emits semantically equivalent but structurally different definitions. MCP Radar intentionally compares the advertised JSON interface rather than attempting lossy semantic normalization.

## Secret handling

Values passed with `--env KEY=VALUE` are supplied to the child process but are never written to the lockfile. Only the key names are recorded.

Environment variables inherited from the parent process are not enumerated.

Do not place credentials, tokens, private paths, or customer data inside server descriptions, schemas, prompts, resources, command arguments, or other fields that the server advertises. Those advertised values may legitimately appear in the lockfile.

## Review guidance

Treat these changes as especially sensitive:

- a newly advertised tool;
- broader input schemas or fewer validation constraints;
- tools associated with process execution, filesystem writes, environment access, credentials, or external network access;
- changed server instructions;
- new resource URI patterns;
- a changed protocol version or capability set.

A clean lockfile comparison means the advertised MCP interface is unchanged. It does not prove that the server implementation or runtime behavior is unchanged.
