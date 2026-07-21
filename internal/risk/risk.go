package risk

import (
	"encoding/json"
	"sort"
	"strings"
)

// Assessment is a deterministic, explainable risk estimate derived from a tool's
// advertised metadata. It is not a vulnerability verdict.
type Assessment struct {
	Score      int      `json:"score"`
	Level      string   `json:"level"`
	Categories []string `json:"categories,omitempty"`
}

type rule struct {
	category string
	weight   int
	terms    []string
}

var rules = []rule{
	{category: "process-execution", weight: 5, terms: []string{"exec", "execute", "shell", "command", "subprocess", "spawn", "terminal", "process"}},
	{category: "destructive-operation", weight: 5, terms: []string{"delete", "remove", "drop", "truncate", "destroy", "wipe", "shutdown", "kill"}},
	{category: "credential-access", weight: 4, terms: []string{"secret", "token", "credential", "password", "api key", "private key", "environment variable", "env var"}},
	{category: "filesystem-write", weight: 4, terms: []string{"write file", "create file", "modify file", "rename file", "move file", "chmod", "upload file", "save file"}},
	{category: "network-access", weight: 3, terms: []string{"http", "https", "url", "network", "fetch", "request", "remote host", "hostname", "socket", "webhook", "endpoint", "external api"}},
	{category: "external-side-effect", weight: 3, terms: []string{"send email", "publish", "deploy", "post message", "create issue", "update record", "purchase", "payment"}},
	{category: "filesystem-read", weight: 2, terms: []string{"read file", "filesystem", "directory", "file path", "list files", "search files"}},
}

// Classify evaluates a complete tool definition. JSON encoding lets the
// classifier inspect names, descriptions, schemas, annotations, and metadata.
func Classify(definition map[string]any) Assessment {
	encoded, _ := json.Marshal(definition)
	text := strings.ToLower(string(encoded))

	assessment := Assessment{}
	for _, candidate := range rules {
		matched := false
		for _, term := range candidate.terms {
			if strings.Contains(text, term) {
				matched = true
				break
			}
		}
		if matched {
			assessment.Score += candidate.weight
			assessment.Categories = append(assessment.Categories, candidate.category)
		}
	}
	if assessment.Score > 10 {
		assessment.Score = 10
	}
	sort.Strings(assessment.Categories)
	assessment.Level = Level(assessment.Score)
	return assessment
}

// Level maps a score to a stable display level.
func Level(score int) string {
	switch {
	case score >= 9:
		return "critical"
	case score >= 6:
		return "high"
	case score >= 3:
		return "medium"
	default:
		return "low"
	}
}

// Rank converts a display level to an ordering value.
func Rank(level string) int {
	switch level {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
