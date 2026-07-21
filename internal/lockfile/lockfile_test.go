package lockfile

import (
	"bytes"
	"strings"
	"testing"
)

func TestFingerprintIsDeterministic(t *testing.T) {
	first := sampleLock([]Item{
		{Name: "z", Definition: map[string]any{"description": "second", "name": "z"}},
		{Name: "a", Definition: map[string]any{"name": "a", "description": "first"}},
	})
	second := sampleLock([]Item{
		{Name: "a", Definition: map[string]any{"description": "first", "name": "a"}},
		{Name: "z", Definition: map[string]any{"name": "z", "description": "second"}},
	})
	if first.Fingerprint != second.Fingerprint {
		t.Fatalf("fingerprints differ: %s != %s", first.Fingerprint, second.Fingerprint)
	}
}

func TestCompareDetectsHighRiskAddedTool(t *testing.T) {
	before := sampleLock([]Item{{Name: "read_file", Definition: map[string]any{"name": "read_file", "description": "Read a file path"}}})
	after := sampleLock([]Item{
		{Name: "read_file", Definition: map[string]any{"name": "read_file", "description": "Read a file path"}},
		{Name: "run_command", Definition: map[string]any{"name": "run_command", "description": "Execute a shell command on a remote host"}},
	})

	diff := Compare(&before, &after)
	if !diff.Changed() {
		t.Fatal("expected drift")
	}
	if diff.HighestRisk != "high" && diff.HighestRisk != "critical" {
		t.Fatalf("expected high or critical risk, got %s", diff.HighestRisk)
	}
	var out bytes.Buffer
	diff.WriteText(&out)
	if !strings.Contains(out.String(), "+ run_command") {
		t.Fatalf("missing added tool in output:\n%s", out.String())
	}
}

func sampleLock(tools []Item) Lockfile {
	lock := Lockfile{
		LockfileVersion: CurrentVersion,
		Transport:       "stdio",
		ProtocolVersion: "2025-11-25",
		Capabilities:    map[string]any{"tools": map[string]any{}},
		Tools:           tools,
	}
	if err := lock.Finalize(); err != nil {
		panic(err)
	}
	return lock
}
