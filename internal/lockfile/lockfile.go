package lockfile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/agentpixelated/mcp-radar/internal/risk"
)

const CurrentVersion = 1

type Invocation struct {
	Command         string   `json:"command"`
	Args            []string `json:"args,omitempty"`
	EnvironmentKeys []string `json:"environmentKeys,omitempty"`
}

type Item struct {
	Name       string           `json:"name"`
	Hash       string           `json:"hash"`
	Risk       *risk.Assessment `json:"risk,omitempty"`
	Definition map[string]any   `json:"definition"`
}

type Lockfile struct {
	LockfileVersion   int            `json:"lockfileVersion"`
	Transport         string         `json:"transport"`
	Invocation        Invocation     `json:"invocation"`
	ProtocolVersion   string         `json:"protocolVersion"`
	ServerInfo        map[string]any `json:"serverInfo,omitempty"`
	Capabilities      map[string]any `json:"capabilities,omitempty"`
	Instructions      string         `json:"instructions,omitempty"`
	Tools             []Item         `json:"tools,omitempty"`
	Prompts           []Item         `json:"prompts,omitempty"`
	Resources         []Item         `json:"resources,omitempty"`
	ResourceTemplates []Item         `json:"resourceTemplates,omitempty"`
	Fingerprint       string         `json:"fingerprint"`
}

// Finalize normalizes ordering, computes per-item hashes and calculates the
// behavior fingerprint. It should be called immediately before writing or
// comparing a lockfile.
func (l *Lockfile) Finalize() error {
	if l.LockfileVersion == 0 {
		l.LockfileVersion = CurrentVersion
	}
	if l.Transport == "" {
		l.Transport = "stdio"
	}

	for i := range l.Tools {
		if err := finalizeItem(&l.Tools[i], true); err != nil {
			return err
		}
	}
	for i := range l.Prompts {
		if err := finalizeItem(&l.Prompts[i], false); err != nil {
			return err
		}
	}
	for i := range l.Resources {
		if err := finalizeItem(&l.Resources[i], false); err != nil {
			return err
		}
	}
	for i := range l.ResourceTemplates {
		if err := finalizeItem(&l.ResourceTemplates[i], false); err != nil {
			return err
		}
	}

	sortItems(l.Tools)
	sortItems(l.Prompts)
	sortItems(l.Resources)
	sortItems(l.ResourceTemplates)
	sort.Strings(l.Invocation.EnvironmentKeys)

	payload := struct {
		Transport         string         `json:"transport"`
		ProtocolVersion   string         `json:"protocolVersion"`
		ServerInfo        map[string]any `json:"serverInfo,omitempty"`
		Capabilities      map[string]any `json:"capabilities,omitempty"`
		Instructions      string         `json:"instructions,omitempty"`
		Tools             []Item         `json:"tools,omitempty"`
		Prompts           []Item         `json:"prompts,omitempty"`
		Resources         []Item         `json:"resources,omitempty"`
		ResourceTemplates []Item         `json:"resourceTemplates,omitempty"`
	}{
		Transport:         l.Transport,
		ProtocolVersion:   l.ProtocolVersion,
		ServerInfo:        l.ServerInfo,
		Capabilities:      l.Capabilities,
		Instructions:      l.Instructions,
		Tools:             l.Tools,
		Prompts:           l.Prompts,
		Resources:         l.Resources,
		ResourceTemplates: l.ResourceTemplates,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode fingerprint payload: %w", err)
	}
	l.Fingerprint = digest(encoded)
	return nil
}

func finalizeItem(item *Item, withRisk bool) error {
	encoded, err := json.Marshal(item.Definition)
	if err != nil {
		return fmt.Errorf("encode %q: %w", item.Name, err)
	}
	item.Hash = digest(encoded)
	if withRisk {
		assessment := risk.Classify(item.Definition)
		item.Risk = &assessment
	} else {
		item.Risk = nil
	}
	return nil
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].Hash < items[j].Hash
		}
		return items[i].Name < items[j].Name
	})
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func Read(path string) (*Lockfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}
	var lock Lockfile
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("decode lockfile: %w", err)
	}
	if lock.LockfileVersion != CurrentVersion {
		return nil, fmt.Errorf("unsupported lockfile version %d (supported: %d)", lock.LockfileVersion, CurrentVersion)
	}
	if err := lock.Finalize(); err != nil {
		return nil, err
	}
	return &lock, nil
}

func Write(path string, lock *Lockfile) error {
	if err := lock.Finalize(); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("encode lockfile: %w", err)
	}
	encoded = append(encoded, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create lockfile directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary lockfile: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(encoded); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary lockfile: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary lockfile: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace lockfile: %w", err)
	}
	return nil
}
