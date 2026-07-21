# Threat model

MCP Radar v0.1 is an interface drift detector. It records and compares behavior that an MCP server advertises through the Model Context Protocol.

## Security goal

MCP Radar is designed to help a reviewer answer:

> Did this MCP server begin advertising different capabilities, instructions, tools, schemas, prompts, resources, or templates?

It provides a deterministic baseline and a CI-friendly failure when that advertised surface changes.

## Assets protected

The lockfile workflow is intended to reduce unnoticed changes to:

- the tools available to an agent;
- the arguments those tools accept;
- server-provided instructions;
- resources and URI templates exposed to a client;
- negotiated protocol and capability metadata.

## Trusted components

The reviewer must currently trust:

- the MCP Radar binary being executed;
- the operating system and Go runtime;
- the command used to launch the MCP server;
- the baseline lockfile at the point it is approved;
- the MCP server to truthfully describe its interface during discovery.

Approval is a security decision. Do not approve a baseline merely to make CI pass.

## In scope

MCP Radar v0.1 detects advertised changes including:

- added, removed, or changed tools;
- changed tool schemas, annotations, descriptions, or metadata;
- changed server instructions, implementation metadata, or capabilities;
- added, removed, or changed prompts, resources, and resource templates;
- protocol-version drift.

## Out of scope

MCP Radar v0.1 does not detect:

- malicious implementation changes hidden behind an unchanged MCP schema;
- filesystem, network, process, or environment access performed at runtime;
- behavior triggered only when a tool is invoked;
- compromised dependencies, binaries, registries, or build infrastructure;
- data exfiltration or prompt injection inside returned tool content;
- differences outside the discovery lifecycle;
- a server that intentionally presents benign behavior during inspection.

The CLI launches the server to perform discovery, so the server process itself is not sandboxed. Run untrusted servers only inside an isolation boundary you control.

## Risk hints

Tool risk scores are deterministic keyword heuristics over the advertised definition. They are intended to focus human review, not classify vulnerabilities or grant trust.

A low score does not mean a tool is safe. A high score does not mean it is malicious.

## Future direction

A later release may add runtime canaries and sandbox-backed observation for filesystem, environment, subprocess, and network behavior. Until those controls exist, treat MCP Radar as one review layer rather than a complete MCP security boundary.
