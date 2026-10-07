package manifest

import (
	"strings"

	"github.com/google/uuid"
)

// EnsureUUIDs checks header and all modules in the manifest.
// It generates and assigns a new UUID v4 if and only if a field is empty.
// Returns the number of UUIDs generated.
func EnsureUUIDs(m *Manifest) int {
	generated := 0

	// Check header.uuid
	if strings.TrimSpace(m.Header.UUID) == "" {
		m.Header.UUID = uuid.New().String()
		generated++
	}

	// Check each module's UUID
	for i := range m.Modules {
		if strings.TrimSpace(m.Modules[i].UUID) == "" {
			m.Modules[i].UUID = uuid.New().String()
			generated++
		}
	}

	return generated
}
