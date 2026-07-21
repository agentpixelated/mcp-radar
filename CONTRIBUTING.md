# Contributing to MCP Radar

Thanks for helping improve MCP Radar. The project is intentionally small and security-focused, so focused changes with clear tests are preferred over large feature bundles.

## Before opening an issue

- Search existing issues and pull requests for the same problem.
- Confirm the behavior against the latest `main` branch.
- Remove credentials, private MCP configuration, customer data, and secret environment values from examples.
- Use the private reporting process in [SECURITY.md](SECURITY.md) for suspected vulnerabilities.

## Development setup

Requirements:

- Go 1.23 or newer;
- Git;
- an MCP stdio server or the included mock server for integration testing.

Clone the repository and run the checks:

```bash
git clone https://github.com/agentpixelated/mcp-radar.git
cd mcp-radar

gofmt -w .
go vet ./...
go test -race ./...
go build ./cmd/mcp-radar
```

Run the local drift demonstration:

```bash
go build -o ./mcp-radar ./cmd/mcp-radar
go build -o ./mock-server ./examples/mock-server

./mcp-radar snapshot --lock v1.lock.json -- ./mock-server --behavior v1
./mcp-radar snapshot --lock v2.lock.json -- ./mock-server --behavior v2
./mcp-radar diff v1.lock.json v2.lock.json
```

## Design principles

Changes should preserve these properties:

1. **Deterministic output** — unchanged advertised behavior should generate an unchanged lockfile.
2. **Explainable decisions** — risk hints and drift reports must be understandable without an LLM.
3. **No secret persistence** — environment values must never be written to lockfiles or logs.
4. **Strict compatibility** — unsupported lockfile versions should fail clearly.
5. **Small attack surface** — avoid network services, accounts, telemetry, databases, and unnecessary dependencies.
6. **Honest security claims** — distinguish advertised behavior from observed runtime behavior.

## Pull requests

Keep pull requests narrow and include:

- the problem being solved;
- the security or compatibility impact;
- tests for changed behavior;
- documentation updates when commands, output, or lockfile fields change;
- confirmation that formatting, vet, tests, and build pass.

Do not silently change the lockfile format. Any compatibility change must update `lockfileVersion`, the lockfile documentation, tests, and migration guidance.

## Commit style

Use short, imperative commit subjects, for example:

```text
Add resource template drift reporting
Clarify lockfile compatibility rules
Reject duplicate tool identities
```

## Scope guidance

Good first contributions include:

- MCP compatibility fixtures;
- clearer drift output;
- deterministic normalization tests;
- false-positive reductions in risk hints;
- documentation and threat-model improvements;
- bug fixes with regression tests.

Large roadmap features such as sandboxing, runtime interception, or signed attestations should begin with a design issue before implementation.
