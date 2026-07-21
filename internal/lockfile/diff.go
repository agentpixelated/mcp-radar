package lockfile

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/agentpixelated/mcp-radar/internal/risk"
)

type Change struct {
	Area       string
	Name       string
	Kind       string
	BeforeHash string
	AfterHash  string
	BeforeRisk *risk.Assessment
	AfterRisk  *risk.Assessment
}

type Diff struct {
	BeforeFingerprint string
	AfterFingerprint  string
	TopLevel          []string
	Changes           []Change
	HighestRisk       string
}

func (d Diff) Changed() bool {
	return len(d.TopLevel) > 0 || len(d.Changes) > 0
}

func Compare(before, after *Lockfile) Diff {
	d := Diff{
		BeforeFingerprint: before.Fingerprint,
		AfterFingerprint:  after.Fingerprint,
		HighestRisk:       "low",
	}
	if before.ProtocolVersion != after.ProtocolVersion {
		d.TopLevel = append(d.TopLevel, fmt.Sprintf("protocol version: %s -> %s", before.ProtocolVersion, after.ProtocolVersion))
		d.raise("medium")
	}
	if hashValue(before.ServerInfo) != hashValue(after.ServerInfo) {
		d.TopLevel = append(d.TopLevel, "server metadata changed")
		d.raise("medium")
	}
	if hashValue(before.Capabilities) != hashValue(after.Capabilities) {
		d.TopLevel = append(d.TopLevel, "server capabilities changed")
		d.raise("high")
	}
	if before.Instructions != after.Instructions {
		d.TopLevel = append(d.TopLevel, "server instructions changed")
		d.raise("medium")
	}

	d.compareItems("tools", before.Tools, after.Tools)
	d.compareItems("prompts", before.Prompts, after.Prompts)
	d.compareItems("resources", before.Resources, after.Resources)
	d.compareItems("resource templates", before.ResourceTemplates, after.ResourceTemplates)
	return d
}

func (d *Diff) compareItems(area string, before, after []Item) {
	left := make(map[string]Item, len(before))
	right := make(map[string]Item, len(after))
	for _, item := range before {
		left[item.Name] = item
	}
	for _, item := range after {
		right[item.Name] = item
	}

	names := make([]string, 0, len(left)+len(right))
	seen := map[string]bool{}
	for name := range left {
		names = append(names, name)
		seen[name] = true
	}
	for name := range right {
		if !seen[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	for _, name := range names {
		oldItem, oldOK := left[name]
		newItem, newOK := right[name]
		change := Change{Area: area, Name: name}
		switch {
		case !oldOK:
			change.Kind = "added"
			change.AfterHash = newItem.Hash
			change.AfterRisk = newItem.Risk
		case !newOK:
			change.Kind = "removed"
			change.BeforeHash = oldItem.Hash
			change.BeforeRisk = oldItem.Risk
		case oldItem.Hash != newItem.Hash:
			change.Kind = "changed"
			change.BeforeHash = oldItem.Hash
			change.AfterHash = newItem.Hash
			change.BeforeRisk = oldItem.Risk
			change.AfterRisk = newItem.Risk
		default:
			continue
		}
		d.Changes = append(d.Changes, change)
		if change.AfterRisk != nil {
			d.raise(change.AfterRisk.Level)
		} else if area == "tools" {
			d.raise("medium")
		} else {
			d.raise("low")
		}
	}
}

func (d *Diff) raise(level string) {
	if risk.Rank(level) > risk.Rank(d.HighestRisk) {
		d.HighestRisk = level
	}
}

func (d Diff) WriteText(w io.Writer) {
	if !d.Changed() {
		fmt.Fprintln(w, "✓ No MCP behavior drift detected.")
		fmt.Fprintf(w, "Fingerprint: %s\n", d.AfterFingerprint)
		return
	}

	added, removed, changed := 0, 0, 0
	for _, change := range d.Changes {
		switch change.Kind {
		case "added":
			added++
		case "removed":
			removed++
		case "changed":
			changed++
		}
	}
	fmt.Fprintln(w, "⚠ MCP behavior drift detected")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Risk: %s\n", strings.ToUpper(d.HighestRisk))
	fmt.Fprintf(w, "Summary: %d added, %d removed, %d changed\n", added, removed, changed)
	fmt.Fprintf(w, "Fingerprint: %s -> %s\n", shortHash(d.BeforeFingerprint), shortHash(d.AfterFingerprint))

	if len(d.TopLevel) > 0 {
		fmt.Fprintln(w, "\nServer")
		for _, change := range d.TopLevel {
			fmt.Fprintf(w, "~ %s\n", change)
		}
	}

	lastArea := ""
	for _, change := range d.Changes {
		if change.Area != lastArea {
			fmt.Fprintf(w, "\n%s\n", title(change.Area))
			lastArea = change.Area
		}
		symbol := map[string]string{"added": "+", "removed": "-", "changed": "~"}[change.Kind]
		fmt.Fprintf(w, "%s %s", symbol, change.Name)
		if change.AfterRisk != nil {
			fmt.Fprintf(w, " [%s %d/10]", change.AfterRisk.Level, change.AfterRisk.Score)
			if len(change.AfterRisk.Categories) > 0 {
				fmt.Fprintf(w, " %s", strings.Join(change.AfterRisk.Categories, ", "))
			}
		}
		fmt.Fprintln(w)
	}
}

func title(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func shortHash(value string) string {
	const prefix = "sha256:"
	value = strings.TrimPrefix(value, prefix)
	if len(value) > 12 {
		value = value[:12]
	}
	return prefix + value
}
