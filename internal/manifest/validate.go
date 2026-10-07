package manifest

import (
	"fmt"
	"strings"
)

type Issue struct {
	Field   string
	Message string
}

// ValidateFile reads the manifest at path and validates it.
func ValidateFile(path string) ([]Issue, error) {
	m, err := Read(path)
	if err != nil {
		return nil, err
	}
	return Validate(m), nil
}

func (i Issue) String() string {
	return fmt.Sprintf("- [%s]: %s", i.Field, i.Message)
}

// Validate checks business rules for the manifest and returns a list of issues
func Validate(m *Manifest) []Issue {
	var issues []Issue

	// 1. Validate Header
	if strings.TrimSpace(m.Header.UUID) == "" {
		issues = append(issues, Issue{
			Field:   "header.uuid",
			Message: "header UUID cannot be empty",
		})
	}

	if strings.TrimSpace(m.Header.Name) == "" {
		issues = append(issues, Issue{
			Field:   "header.name",
			Message: "header name cannot be empty",
		})
	}

	// 2. Validate Modules
	if len(m.Modules) == 0 || len(m.Modules) > 2 {
		issues = append(issues, Issue{
			Field:   "modules",
			Message: fmt.Sprintf("manifest must contain 1 or 2 modules, got %d", len(m.Modules)),
		})
	}

	if len(m.Modules) == 2 {
		types := map[ModuleType]bool{}
		for _, mod := range m.Modules {
			types[mod.Type] = true
		}
		if types[ModuleTypeData] && types[ModuleTypeResources] {
			issues = append(issues, Issue{
				Field:   "modules",
				Message: "You must not include both 'data' and 'resources' modules at the same time",
			})
		}
	}

	for i, mod := range m.Modules {
		if strings.TrimSpace(mod.UUID) == "" {
			issues = append(issues, Issue{
				Field:   fmt.Sprintf("modules[%d].uuid", i),
				Message: "module UUID cannot be empty",
			})
		}
	}

	// 3. Validate Dependencies (XOR Mutually Exclusive)
	for i, dep := range m.Dependencies {
		hasUUID := strings.TrimSpace(dep.UUID) != ""
		hasModule := strings.TrimSpace(dep.ModuleName) != ""

		fieldPath := fmt.Sprintf("dependencies[%d]", i)

		if hasUUID && hasModule {
			issues = append(issues, Issue{
				Field:   fieldPath,
				Message: "dependency cannot specify both 'uuid' and 'module_name'",
			})
		} else if !hasUUID && !hasModule {
			issues = append(issues, Issue{
				Field:   fieldPath,
				Message: "dependency must specify either 'uuid' or 'module_name'",
			})
		}
	}

	return issues
}
