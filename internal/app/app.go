package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/agentpixelated/mcp-radar/internal/lockfile"
	"github.com/agentpixelated/mcp-radar/internal/mcp"
)

const Version = "0.1.0"

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error {
	if !strings.Contains(value, "=") {
		return errors.New("environment override must use KEY=VALUE")
	}
	*s = append(*s, value)
	return nil
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "snapshot":
		return captureCommand(ctx, "snapshot", args[1:], stdout, stderr)
	case "approve":
		return captureCommand(ctx, "approve", args[1:], stdout, stderr)
	case "verify":
		return verifyCommand(ctx, args[1:], stdout, stderr)
	case "diff":
		return diffCommand(args[1:], stdout, stderr)
	case "version", "--version", "-version":
		fmt.Fprintf(stdout, "mcp-radar %s\n", Version)
		return 0
	case "help", "--help", "-h":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		usage(stderr)
		return 2
	}
}

func captureCommand(ctx context.Context, verb string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(verb, flag.ContinueOnError)
	fs.SetOutput(stderr)
	lockPath := fs.String("lock", "mcp-radar.lock.json", "lockfile path")
	protocol := fs.String("protocol", mcp.DefaultProtocolVersion, "MCP protocol version to request")
	timeout := fs.Duration("timeout", 15*time.Second, "overall discovery timeout")
	var env stringList
	fs.Var(&env, "env", "server environment override KEY=VALUE (repeatable; values are not stored)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	command := fs.Args()
	if len(command) == 0 {
		fmt.Fprintln(stderr, "server command is required after --")
		return 2
	}
	lock, err := capture(ctx, command, env, *protocol, *timeout)
	if err != nil {
		fmt.Fprintf(stderr, "mcp-radar: %v\n", err)
		return 2
	}
	if err := lockfile.Write(*lockPath, lock); err != nil {
		fmt.Fprintf(stderr, "mcp-radar: %v\n", err)
		return 2
	}
	if verb == "approve" {
		fmt.Fprintf(stdout, "✓ Approved MCP behavior in %s\n", *lockPath)
	} else {
		fmt.Fprintf(stdout, "✓ Wrote MCP behavior snapshot to %s\n", *lockPath)
	}
	fmt.Fprintf(stdout, "Fingerprint: %s\n", lock.Fingerprint)
	fmt.Fprintf(stdout, "Discovered: %d tools, %d prompts, %d resources, %d resource templates\n", len(lock.Tools), len(lock.Prompts), len(lock.Resources), len(lock.ResourceTemplates))
	return 0
}

func verifyCommand(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	lockPath := fs.String("lock", "mcp-radar.lock.json", "approved lockfile path")
	timeout := fs.Duration("timeout", 15*time.Second, "overall discovery timeout")
	protocol := fs.String("protocol", "", "MCP protocol version to request (defaults to lockfile version)")
	var env stringList
	fs.Var(&env, "env", "server environment override KEY=VALUE (repeatable; values are not stored)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	approved, err := lockfile.Read(*lockPath)
	if err != nil {
		fmt.Fprintf(stderr, "mcp-radar: %v\n", err)
		return 2
	}
	command := fs.Args()
	if len(command) == 0 {
		if approved.Invocation.Command == "" {
			fmt.Fprintln(stderr, "mcp-radar: no server command supplied or stored in lockfile")
			return 2
		}
		command = append([]string{approved.Invocation.Command}, approved.Invocation.Args...)
	}
	requestedProtocol := *protocol
	if requestedProtocol == "" {
		requestedProtocol = approved.ProtocolVersion
	}
	current, err := capture(ctx, command, env, requestedProtocol, *timeout)
	if err != nil {
		fmt.Fprintf(stderr, "mcp-radar: %v\n", err)
		return 2
	}
	diff := lockfile.Compare(approved, current)
	diff.WriteText(stdout)
	if diff.Changed() {
		return 1
	}
	return 0
}

func diffCommand(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 2 {
		fmt.Fprintln(stderr, "usage: mcp-radar diff <approved.lock.json> <candidate.lock.json>")
		return 2
	}
	before, err := lockfile.Read(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "mcp-radar: %v\n", err)
		return 2
	}
	after, err := lockfile.Read(fs.Arg(1))
	if err != nil {
		fmt.Fprintf(stderr, "mcp-radar: %v\n", err)
		return 2
	}
	diff := lockfile.Compare(before, after)
	diff.WriteText(stdout)
	if diff.Changed() {
		return 1
	}
	return 0
}

func capture(ctx context.Context, command []string, envOverrides []string, protocol string, timeout time.Duration) (*lockfile.Lockfile, error) {
	env := append([]string(nil), os.Environ()...)
	env = append(env, envOverrides...)
	return mcp.Capture(ctx, mcp.Options{
		Command:         command[0],
		Args:            command[1:],
		Env:             env,
		EnvironmentKeys: mcp.EnvKeys(envOverrides),
		ProtocolVersion: protocol,
		Timeout:         timeout,
	})
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `mcp-radar — behavior lockfiles for MCP servers

Usage:
  mcp-radar snapshot [flags] -- <server-command> [args...]
  mcp-radar approve  [flags] -- <server-command> [args...]
  mcp-radar verify   [flags] [-- <server-command> [args...]]
  mcp-radar diff <approved.lock.json> <candidate.lock.json>
  mcp-radar version

Core flags:
  --lock PATH          Lockfile path (default mcp-radar.lock.json)
  --protocol VERSION   Requested MCP protocol version
  --timeout DURATION   Discovery timeout (default 15s)
  --env KEY=VALUE      Server environment override; repeatable

Exit codes:
  0  no drift / success
  1  behavior drift detected
  2  usage, launch, protocol, or lockfile error`)
}
