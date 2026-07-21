package risk

import "testing"

func TestClassifyRemoteCommand(t *testing.T) {
	got := Classify(map[string]any{
		"name":        "run_command",
		"description": "Execute a shell command on a remote host",
		"inputSchema": map[string]any{"type": "object"},
	})

	if got.Level != "high" && got.Level != "critical" {
		t.Fatalf("expected high or critical risk, got %#v", got)
	}
	assertContains(t, got.Categories, "process-execution")
	assertContains(t, got.Categories, "network-access")
}

func TestClassifyReadOnlyTool(t *testing.T) {
	got := Classify(map[string]any{
		"name":        "read_file",
		"description": "Read a file path from the filesystem",
	})
	if got.Level != "low" {
		t.Fatalf("expected low risk, got %#v", got)
	}
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("%q not found in %#v", want, values)
}
