package manifest

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ---------------------------------------------------------
// UUID filling
// ---------------------------------------------------------
// These helpers find empty header/module UUID fields and fill them with
// fresh UUID v4 values. They power the `uuid --insert` command: file-level
// orchestration lives in FillUUIDsInFile, while the pure in-memory logic
// in FillMissingUUIDs is easy to unit test.

// UUIDChange describes a single UUID that was generated.
type UUIDChange struct {
	Field string
	UUID  string
}

// EnsureUUIDs checks header and all modules in the manifest.
// It generates and assigns a new UUID v4 if and only if a field is empty.
// Returns the number of UUIDs generated.
func EnsureUUIDs(m *Manifest) int {
	return len(FillMissingUUIDs(m))
}

// FillMissingUUIDs generates UUIDs for every empty field and returns what changed.
func FillMissingUUIDs(m *Manifest) []UUIDChange {
	changes := []UUIDChange{}

	if strings.TrimSpace(m.Header.UUID) == "" {
		m.Header.UUID = uuid.New().String()
		changes = append(changes, UUIDChange{Field: "header.uuid", UUID: m.Header.UUID})
	}

	for i := range m.Modules {
		if strings.TrimSpace(m.Modules[i].UUID) == "" {
			m.Modules[i].UUID = uuid.New().String()
			changes = append(changes, UUIDChange{
				Field: fmt.Sprintf("modules[%d].uuid", i),
				UUID:  m.Modules[i].UUID,
			})
		}
	}

	return changes
}

// FillUUIDsInFile reads the manifest at path, fills missing UUIDs, and writes
// it back unless dryRun is true. It reports what was (or would be) added.
func FillUUIDsInFile(path string, dryRun bool) ([]UUIDChange, error) {
	m, err := Read(path)
	if err != nil {
		return nil, fmt.Errorf("error reading manifest: %w", err)
	}

	changes := FillMissingUUIDs(m)
	if len(changes) == 0 || dryRun {
		return changes, nil
	}

	if err := Write(path, m); err != nil {
		return nil, fmt.Errorf("error updating manifest: %w", err)
	}
	return changes, nil
}
