package manifest

import (
	"encoding/json"
	"fmt"
	"os"
)

// Read reads the manifest file from disk and parses it into a *Manifest struct.
// The original document is retained so unknown fields can survive a rewrite.
func Read(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file at '%s': %w", path, err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("error parsing manifest JSON: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("error parsing manifest JSON: %w", err)
	}
	m.raw = raw

	return &m, nil
}
