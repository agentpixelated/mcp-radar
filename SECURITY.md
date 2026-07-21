# Security policy

MCP Radar is experimental security tooling. A clean lockfile comparison means the advertised MCP interface is unchanged; it does not prove that the server implementation or runtime behavior is safe.

Read the [threat model](docs/threat-model.md) before relying on MCP Radar as part of a security workflow.

## Supported versions

Security fixes are applied to the latest code on `main` and to the most recent tagged release when releases are available. Pre-release branches may change without compatibility guarantees.

## Reporting a vulnerability

Please report suspected vulnerabilities privately through GitHub's **Security** tab using a private vulnerability report or security advisory.

Include, when possible:

- the affected commit or version;
- operating system and Go version;
- reproduction steps or a minimal proof of concept;
- expected and observed behavior;
- security impact;
- suggested mitigation, if known.

Do not include secrets, credentials, private MCP configurations, customer data, or exploitable details in a public issue.

## Public disclosure

Please allow time for investigation and remediation before publishing vulnerability details. Once a fix is available, the maintainers may coordinate an advisory and release notes with the reporter.

## Security boundaries

MCP Radar v0.1 launches an MCP server for discovery but does not sandbox it. Running an untrusted server may itself be dangerous. Use an isolation boundary appropriate to the server's trust level.
