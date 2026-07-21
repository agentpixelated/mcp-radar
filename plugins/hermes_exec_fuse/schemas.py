"""LLM-facing tool schemas for hermes-exec-fuse."""

EXEC_FUSE = {
    "name": "exec_fuse",
    "description": (
        "Execute two or more shell commands in one tool call. Use this instead of "
        "repeated terminal calls. Independent read-only commands may run in parallel; "
        "depends_on preserves required ordering. Exact safe duplicates are executed once, "
        "and safe read-only results are reused while the workspace is unchanged."
    ),
    "parameters": {
        "type": "object",
        "properties": {
            "commands": {
                "type": "array",
                "minItems": 1,
                "maxItems": 24,
                "description": "Commands to execute. IDs must be unique.",
                "items": {
                    "type": "object",
                    "properties": {
                        "id": {
                            "type": "string",
                            "description": "Short unique identifier for this command.",
                        },
                        "command": {
                            "type": "string",
                            "description": "Shell command to execute through Hermes terminal.",
                        },
                        "cwd": {
                            "type": "string",
                            "description": "Optional working directory for this command.",
                        },
                        "timeout": {
                            "type": "integer",
                            "minimum": 1,
                            "maximum": 600,
                            "description": "Optional foreground timeout in seconds.",
                        },
                        "depends_on": {
                            "type": "array",
                            "items": {"type": "string"},
                            "description": "Command IDs that must finish successfully first.",
                        },
                        "cache": {
                            "type": "boolean",
                            "description": "Allow reuse when the command is classified read-only.",
                        },
                    },
                    "required": ["id", "command"],
                },
            },
            "parallel": {
                "type": "boolean",
                "description": "Run ready read-only commands concurrently. Default true.",
            },
            "cache": {
                "type": "boolean",
                "description": "Enable safe read-only cache for the batch. Default true.",
            },
            "fail_fast": {
                "type": "boolean",
                "description": "Skip remaining commands after a failed command. Default false.",
            },
            "max_output_chars": {
                "type": "integer",
                "minimum": 500,
                "maximum": 20000,
                "description": "Maximum compact output retained per command. Default 4000.",
            },
        },
        "required": ["commands"],
    },
}

EXEC_FUSE_STATS = {
    "name": "exec_fuse_stats",
    "description": (
        "Show hermes-exec-fuse cache and execution metrics for the current session, "
        "including avoided command executions and estimated output characters saved."
    ),
    "parameters": {"type": "object", "properties": {}},
}
