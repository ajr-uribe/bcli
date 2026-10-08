package manifest

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ---------------------------------------------------------
// Business validation
// ---------------------------------------------------------
// Validate checks the RULES of a manifest: required fields, UUID formats,
// allowed module combinations, dependency declarations, and script API
// requirements. (JSON shape checking happens earlier, in types.go, during
// parsing.) It returns every problem found as a list of Issues, so callers
// can report them all at once instead of failing on the first one.

const requiredScriptLanguage = "javascript"

type Issue struct {
	Field   string
	Message string
}

func (i Issue) String() string {
	return fmt.Sprintf("- [%s]: %s", i.Field, i.Message)
}

// ValidateFile reads the manifest at path and validates it.
func ValidateFile(path string) ([]Issue, error) {
	m, err := Read(path)
	if err != nil {
		return nil, err
	}

	return Validate(m), nil
}

// Validate checks business rules for the manifest and returns a list of issues.
func Validate(m *Manifest) []Issue {
	if m == nil {
		return []Issue{{Field: "manifest", Message: "manifest is nil"}}
	}

	var issues []Issue

	issues = append(issues, validateHeader(m.Header)...)
	issues = append(issues, validateModules(m.Modules)...)
	issues = append(issues, validateDependencies(m.Dependencies)...)
	issues = append(issues, validateScriptDependencies(m.Modules, m.Dependencies)...)

	return issues
}

func validateHeader(h Header) []Issue {
	var issues []Issue

	issues = append(issues, validateUUID("header.uuid", "header UUID", h.UUID)...)

	if strings.TrimSpace(h.Name) == "" {
		issues = append(issues, Issue{
			Field:   "header.name",
			Message: "header name cannot be empty",
		})
	}

	if h.MinEngineVersion == (EngineVersion{}) {
		issues = append(issues, Issue{
			Field:   "header.min_engine_version",
			Message: "min_engine_version is required and cannot be [0, 0, 0]",
		})
	}

	return issues
}

func validateModules(modules []Module) []Issue {
	var issues []Issue

	if len(modules) == 0 || len(modules) > 2 {
		issues = append(issues, Issue{
			Field:   "modules",
			Message: fmt.Sprintf("manifest must contain 1 or 2 modules, got %d", len(modules)),
		})
	}

	seenTypes := make(map[ModuleType]bool, len(modules))
	seenUUIDs := make(map[string]bool, len(modules))

	for i, mod := range modules {
		field := func(name string) string {
			return fmt.Sprintf("modules[%d].%s", i, name)
		}

		// Duplicate module types.
		if seenTypes[mod.Type] {
			issues = append(issues, Issue{
				Field:   field("type"),
				Message: fmt.Sprintf("duplicate module type %q", mod.Type),
			})
		}

		seenTypes[mod.Type] = true

		// UUID presence, format and uniqueness.
		issues = append(issues, validateUUID(field("uuid"), "module UUID", mod.UUID)...)

		if id := strings.ToLower(strings.TrimSpace(mod.UUID)); id != "" {
			if seenUUIDs[id] {
				issues = append(issues, Issue{
					Field:   field("uuid"),
					Message: fmt.Sprintf("duplicate module UUID %q", mod.UUID),
				})
			}

			seenUUIDs[id] = true
		}

		// Script module requirements.
		if mod.Type == ModuleTypeScript {
			if mod.Language != requiredScriptLanguage {
				issues = append(issues, Issue{
					Field:   field("language"),
					Message: fmt.Sprintf("script module language must be %q, got %q", requiredScriptLanguage, mod.Language),
				})
			}

			if strings.TrimSpace(mod.Entry) == "" {
				issues = append(issues, Issue{
					Field:   field("entry"),
					Message: "script module must define an 'entry' file",
				})
			}
		}
	}

	if seenTypes[ModuleTypeData] && seenTypes[ModuleTypeResources] {
		issues = append(issues, Issue{
			Field:   "modules",
			Message: "you must not include both 'data' and 'resources' modules at the same time",
		})
	}

	if seenTypes[ModuleTypeScript] && seenTypes[ModuleTypeResources] {
		issues = append(issues, Issue{
			Field:   "modules",
			Message: "you must not include both 'script' and 'resources' modules at the same time",
		})
	}

	return issues
}

// validateDependencies checks that each dependency is either uuid-based or
// module_name-based (mutually exclusive) and that it declares a version.
func validateDependencies(dependencies []Dependency) []Issue {
	var issues []Issue

	for i, dep := range dependencies {
		field := fmt.Sprintf("dependencies[%d]", i)

		hasUUID := strings.TrimSpace(dep.UUID) != ""
		hasModule := strings.TrimSpace(dep.ModuleName) != ""

		switch {
		case hasUUID && hasModule:
			issues = append(issues, Issue{
				Field:   field,
				Message: "dependency cannot specify both 'uuid' and 'module_name'",
			})

		case !hasUUID && !hasModule:
			issues = append(issues, Issue{
				Field:   field,
				Message: "dependency must specify either 'uuid' or 'module_name'",
			})

		case hasUUID:
			issues = append(issues, validateUUID(field+".uuid", "dependency UUID", dep.UUID)...)
		}

		if isEmptyVersion(dep.Version) {
			issues = append(issues, Issue{
				Field:   field + ".version",
				Message: "dependency version is required",
			})
		}
	}

	return issues
}

// validateScriptDependencies ensures that a script module comes with at least
// one Script API dependency declared by module_name.
func validateScriptDependencies(modules []Module, dependencies []Dependency) []Issue {
	hasScript := false

	for _, mod := range modules {
		if mod.Type == ModuleTypeScript {
			hasScript = true
			break
		}
	}

	if !hasScript {
		return nil
	}

	for _, dep := range dependencies {
		if strings.TrimSpace(dep.ModuleName) != "" {
			return nil
		}
	}

	return []Issue{{
		Field:   "dependencies",
		Message: "a script module requires at least one Script API dependency (module_name)",
	}}
}

func validateUUID(field, label, value string) []Issue {
	value = strings.TrimSpace(value)

	if value == "" {
		return []Issue{{Field: field, Message: label + " cannot be empty"}}
	}

	if _, err := uuid.Parse(value); err != nil {
		return []Issue{{Field: field, Message: fmt.Sprintf("%s %q is not a valid UUID", label, value)}}
	}

	return nil
}

func isEmptyVersion(v any) bool {
	switch val := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(val) == ""
	default:
		return false
	}
}
