package manifest

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
)

// ---------------------------------------------------------
// Interactive generation
// ---------------------------------------------------------
// This file builds a brand-new Manifest by asking the user questions in the
// terminal (pack name, type, authors, script dependencies, ...). It is the
// logic behind `bcli manifest generate`; the cobra command in cmd/ is just
// a thin wrapper that calls InteractiveGenerate and writes the result.

const (
	packTypeBP     = "BP"
	packTypeRP     = "RP"
	packTypeScript = "Script"

	scriptEntry    = "main.js"
	scriptLanguage = "javascript"
	minecraftScope = "@minecraft/"
)

var (
	defaultMinEngineVersion = EngineVersion{1, 26, 0}
	defaultVersion          = Version{1, 0, 0}

	// Accepts "1.13.0", "2.0.0" and suffixed versions such as "1.10.0-beta".
	moduleVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.\-]+)?$`)
)

// packInfo holds the answers of the main pack form.
type packInfo struct {
	Name        string
	Description string
	Type        string
	Authors     string
}

// InteractiveGenerate launches the interactive terminal prompts and builds a Manifest.
func InteractiveGenerate() (*Manifest, error) {
	info, err := promptPackInfo()
	if err != nil {
		return nil, err
	}

	modules, err := buildModules(info.Type)
	if err != nil {
		return nil, err
	}

	var dependencies []Dependency

	if info.Type == packTypeScript {
		dependencies, err = promptScriptDependencies()
		if err != nil {
			return nil, err
		}
	}

	return &Manifest{
		FormatVersion: FormatVersion2,
		Header: Header{
			Name:             info.Name,
			Description:      info.Description,
			UUID:             uuid.New().String(),
			Version:          defaultVersion,
			MinEngineVersion: defaultMinEngineVersion,
		},
		Modules:      modules,
		Dependencies: dependencies,
		Metadata:     buildMetadata(info.Authors),
	}, nil
}

// promptPackInfo asks for the main pack information.
func promptPackInfo() (packInfo, error) {
	info := packInfo{Type: packTypeBP}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Pack Name").
				Value(&info.Name).
				Validate(func(value string) error {
					if strings.TrimSpace(value) == "" {
						return fmt.Errorf("the pack name cannot be empty")
					}

					return nil
				}),

			huh.NewInput().
				Title("Pack Description (optional)").
				Value(&info.Description),

			huh.NewSelect[string]().
				Title("Pack Type").
				Options(
					huh.NewOption("Behavior Pack (BP)", packTypeBP),
					huh.NewOption("Resource Pack (RP)", packTypeRP),
					huh.NewOption("Script Add-on (BP + Script)", packTypeScript),
				).
				Value(&info.Type),

			huh.NewInput().
				Title("Author(s) (separate by comma, optional)").
				Placeholder("e.g. AwesomeArtist, AwesomeDev").
				Value(&info.Authors),
		),
	)

	if err := form.Run(); err != nil {
		return packInfo{}, err
	}

	info.Name = strings.TrimSpace(info.Name)
	info.Description = strings.TrimSpace(info.Description)
	info.Authors = strings.TrimSpace(info.Authors)

	return info, nil
}

// buildModules creates the modules according to the selected pack type.
func buildModules(packType string) ([]Module, error) {
	switch packType {
	case packTypeBP:
		return []Module{
			newModule(ModuleTypeData, "Behavior module"),
		}, nil

	case packTypeRP:
		return []Module{
			newModule(ModuleTypeResources, "Resource module"),
		}, nil

	case packTypeScript:
		scriptModule := newModule(ModuleTypeScript, "Scripting module")
		scriptModule.Language = scriptLanguage
		scriptModule.Entry = scriptEntry

		return []Module{
			newModule(ModuleTypeData, "Behavior module"),
			scriptModule,
		}, nil

	default:
		return nil, fmt.Errorf("invalid pack type: %q", packType)
	}
}

// newModule creates a module with a fresh UUID and the default version.
func newModule(moduleType ModuleType, description string) Module {
	return Module{
		Type:        moduleType,
		UUID:        uuid.New().String(),
		Version:     defaultVersion,
		Description: description,
	}
}

// promptScriptDependencies asks for script modules until the user stops adding them.
// At least one module is always requested.
func promptScriptDependencies() ([]Dependency, error) {
	var dependencies []Dependency

	added := make(map[string]struct{})

	for {
		var (
			name    string
			version string
			addMore bool
		)

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Script Module").
					Placeholder("server or @minecraft/server").
					Value(&name).
					Validate(func(value string) error {
						normalized := normalizeModuleName(value)

						if normalized == "" {
							return fmt.Errorf("the module name cannot be empty")
						}

						if _, exists := added[normalized]; exists {
							return fmt.Errorf("script module %q has already been added", normalized)
						}

						return nil
					}),

				huh.NewInput().
					Title("Module Version").
					Placeholder("2.0.0").
					Value(&version).
					Validate(func(value string) error {
						if !moduleVersionPattern.MatchString(strings.TrimSpace(value)) {
							return fmt.Errorf("invalid version, expected e.g. 1.13.0 or 1.10.0-beta")
						}

						return nil
					}),

				huh.NewConfirm().
					Title("Insert Another Script Module?").
					Value(&addMore),
			),
		)

		if err := form.Run(); err != nil {
			return nil, err
		}

		name = normalizeModuleName(name)
		added[name] = struct{}{}

		dependencies = append(dependencies, Dependency{
			ModuleName: name,
			Version:    strings.TrimSpace(version),
		})

		if !addMore {
			return dependencies, nil
		}
	}
}

// buildMetadata returns nil when no valid author was provided.
func buildMetadata(authorsInput string) *Metadata {
	authors := parseCommaList(authorsInput)

	if len(authors) == 0 {
		return nil
	}

	return &Metadata{Authors: authors}
}

// normalizeModuleName expands short aliases ("server" -> "@minecraft/server").
func normalizeModuleName(name string) string {
	name = strings.TrimSpace(name)

	if name != "" && !strings.HasPrefix(name, "@") {
		return minecraftScope + name
	}

	return name
}

// parseCommaList splits a comma-separated string, trimming whitespace and
// dropping empty and duplicate (case-insensitive) entries.
func parseCommaList(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part == "" {
			continue
		}

		key := strings.ToLower(part)

		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		result = append(result, part)
	}

	return result
}
