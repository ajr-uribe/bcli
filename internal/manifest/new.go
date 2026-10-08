package manifest

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ---------------------------------------------------------
// Non-interactive construction
// ---------------------------------------------------------
// New builds a Manifest from explicit options without prompting the user.
// It is the logic behind `pack.CreateBP/CreateRP/CreateBoth`; the
// interactive sibling InteractiveGenerate (generate.go) collects answers
// in the terminal and then delegates here, so both paths share one
// construction and validation flow.

// PackKind selects which pack layout New generates.
type PackKind string

const (
	// KindBehavior is a behavior pack: one data module.
	KindBehavior PackKind = "bp"
	// KindResource is a resource pack: one resources module.
	KindResource PackKind = "rp"
	// KindScript is a script add-on: data + script modules plus Script API
	// dependencies (requires ScriptDeps).
	KindScript PackKind = "script"
)

// ScriptDependency is one Script API dependency, e.g. {Name: "server",
// Version: "2.0.0"}. The name may use the short form ("server") or the
// full form ("@minecraft/server"); it is normalized to the full form.
type ScriptDependency struct {
	Name    string
	Version string
}

// NewOptions are the inputs for New. Version and MinEngineVersion are
// pointers so callers can leave them nil to accept the defaults
// ([1, 0, 0] and [1, 26, 0]).
type NewOptions struct {
	Name             string
	Description      string
	Kind             PackKind
	Authors          []string
	ScriptDeps       []ScriptDependency
	Version          *Version
	MinEngineVersion *EngineVersion
}

// New builds and validates a Manifest without any user interaction.
// It returns an error if the inputs are invalid or the result fails
// business validation.
func New(opts NewOptions) (*Manifest, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return nil, fmt.Errorf("pack name cannot be empty")
	}

	var packType string
	switch opts.Kind {
	case KindBehavior:
		packType = packTypeBP
	case KindResource:
		packType = packTypeRP
	case KindScript:
		packType = packTypeScript
	default:
		return nil, fmt.Errorf("invalid pack kind %q (allowed: %q, %q, %q)",
			opts.Kind, KindBehavior, KindResource, KindScript)
	}

	version := defaultVersion
	if opts.Version != nil {
		version = *opts.Version
	}

	minEngine := defaultMinEngineVersion
	if opts.MinEngineVersion != nil {
		if *opts.MinEngineVersion == (EngineVersion{}) {
			return nil, fmt.Errorf("min_engine_version cannot be [0, 0, 0]")
		}
		minEngine = *opts.MinEngineVersion
	}

	modules, err := buildModules(packType)
	if err != nil {
		return nil, err
	}

	var dependencies []Dependency
	if packType == packTypeScript {
		dependencies, err = buildScriptDependencies(opts.ScriptDeps)
		if err != nil {
			return nil, err
		}
	} else if len(opts.ScriptDeps) > 0 {
		return nil, fmt.Errorf("script dependencies require pack kind %q", KindScript)
	}

	m := &Manifest{
		FormatVersion: FormatVersion2,
		Header: Header{
			Name:             name,
			Description:      strings.TrimSpace(opts.Description),
			UUID:             uuid.New().String(),
			Version:          version,
			MinEngineVersion: minEngine,
		},
		Modules:      modules,
		Dependencies: dependencies,
		Metadata:     buildMetadata(strings.Join(opts.Authors, ",")),
	}

	if issues := Validate(m); len(issues) > 0 {
		return nil, fmt.Errorf("generated manifest has %d validation issue(s): %v", len(issues), issues)
	}

	return m, nil
}

// buildScriptDependencies validates raw script dependency inputs and
// converts them to manifest Dependencies with normalized module names.
func buildScriptDependencies(deps []ScriptDependency) ([]Dependency, error) {
	if len(deps) == 0 {
		return nil, fmt.Errorf("a script pack requires at least one Script API dependency")
	}

	seen := make(map[string]struct{}, len(deps))
	out := make([]Dependency, 0, len(deps))

	for _, dep := range deps {
		name := normalizeModuleName(dep.Name)
		if name == "" {
			return nil, fmt.Errorf("script module name cannot be empty")
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("script module %q is duplicated", name)
		}
		seen[name] = struct{}{}

		version := strings.TrimSpace(dep.Version)
		if !moduleVersionPattern.MatchString(version) {
			return nil, fmt.Errorf("invalid version %q for module %q, expected e.g. 1.13.0 or 1.10.0-beta", dep.Version, name)
		}

		out = append(out, Dependency{ModuleName: name, Version: version})
	}

	return out, nil
}

// LinkManifests cross-links a behavior-pack and a resource-pack manifest
// so each one depends on the other's header UUID at the other's header
// version. This is what makes the two packs load as a pair in-game.
func LinkManifests(bp, rp *Manifest) error {
	if bp == nil || rp == nil {
		return fmt.Errorf("cannot link manifests: behavior and resource manifests must both be non-nil")
	}

	bp.Dependencies = append(bp.Dependencies, Dependency{
		UUID:    rp.Header.UUID,
		Version: rp.Header.Version,
	})
	rp.Dependencies = append(rp.Dependencies, Dependency{
		UUID:    bp.Header.UUID,
		Version: bp.Header.Version,
	})

	return nil
}
