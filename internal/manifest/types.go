// Package manifest models the manifest.json file used by Minecraft Bedrock
// add-ons and provides helpers to read, write, validate, and generate it.
//
// File layout (grouped by concern, not by type):
//
//	types.go    - all structs, custom types, and their JSON unmarshaling rules.
//	            This is the "shape" of a manifest: what fields exist and which
//	            raw JSON values are accepted at parse time.
//	io.go       - disk I/O: Read and Write, including preservation of unknown
//	            fields so editing a manifest never drops data we don't model.
//	validate.go - business rules that go beyond JSON shape (required fields,
//	            UUID formats, module combinations, dependency rules, ...).
//	uuid.go     - filling in missing UUIDs, reporting what changed.
//	generate.go - interactive manifest creation (terminal prompts).
//	new.go      - non-interactive manifest construction from explicit
//	            options, plus linking behavior/resource manifests.
package manifest

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ---------------------------------------------------------
// Custom scalar types
// ---------------------------------------------------------
// These wrap plain strings/ints so invalid values are rejected as soon as
// the JSON is parsed (in UnmarshalJSON below), instead of slipping through
// to validation later.

type (
	// ModuleType identifies the kind of a manifest module.
	ModuleType string
	// FormatVersion is the top-level manifest format version.
	FormatVersion int
	// Version is a [major, minor, patch] triple, e.g. [1, 0, 0].
	Version [3]int
	// EngineVersion is the minimum Minecraft engine version, e.g. [1, 26, 0].
	EngineVersion [3]int
	// ProductType describes the kind of product in the metadata block.
	ProductType string
)

const (
	ModuleTypeResources ModuleType = "resources"
	ModuleTypeData      ModuleType = "data"
	ModuleTypeScript    ModuleType = "script"

	FormatVersion1 FormatVersion = 1
	FormatVersion2 FormatVersion = 2
	FormatVersion3 FormatVersion = 3

	ProductTypeAddon ProductType = "addon"
)

// ---------------------------------------------------------
// Document structs
// ---------------------------------------------------------
// These mirror the manifest.json structure field by field. The `json` tags
// control how each field maps to JSON keys; `omitempty` means the key is
// skipped when the value is empty (only used for truly optional fields).

type Header struct {
	Name             string        `json:"name"`
	Description      string        `json:"description"`
	UUID             string        `json:"uuid"`
	Version          Version       `json:"version"`
	MinEngineVersion EngineVersion `json:"min_engine_version"`
}

type Module struct {
	Type        ModuleType `json:"type"`
	Description string     `json:"description,omitempty"`
	UUID        string     `json:"uuid"`
	Version     Version    `json:"version"`
	Language    string     `json:"language,omitempty"`
	Entry       string     `json:"entry,omitempty"`
}

type Dependency struct {
	UUID       string `json:"uuid,omitempty"`
	ModuleName string `json:"module_name,omitempty"`
	Version    any    `json:"version"`
}

type Metadata struct {
	Authors       []string            `json:"authors,omitempty"`
	License       string              `json:"license,omitempty"`
	URL           string              `json:"url,omitempty"`
	ProductType   ProductType         `json:"product_type,omitempty"`
	GeneratedWith map[string][]string `json:"generated_with,omitempty"`
}

type Manifest struct {
	FormatVersion FormatVersion `json:"format_version"`
	Header        Header        `json:"header"`
	Modules       []Module      `json:"modules"`
	Dependencies  []Dependency  `json:"dependencies,omitempty"`
	Metadata      *Metadata     `json:"metadata,omitempty"`
	// raw holds the original decoded document so unknown fields can be
	// preserved when writing the manifest back to disk (see io.go).
	raw map[string]any
}

// ---------------------------------------------------------
// JSON parsing rules (UnmarshalJSON)
// ---------------------------------------------------------
// These run automatically during json.Unmarshal. They reject malformed
// values early with a friendly error, before business validation runs.
// Rule of thumb: parsing checks the SHAPE ("is this value well-formed?"),
// while validate.go checks the RULES ("is this combination allowed?").

func (m *ModuleType) UnmarshalJSON(data []byte) error {
	var val string
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	switch ModuleType(val) {
	case ModuleTypeResources, ModuleTypeData, ModuleTypeScript:
		*m = ModuleType(val)
		return nil
	default:
		return fmt.Errorf("module.type '%s' not valid (allowed: resources, data, script)", val)
	}
}

func (m *FormatVersion) UnmarshalJSON(data []byte) error {
	var val int
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	switch FormatVersion(val) {
	case FormatVersion1, FormatVersion2, FormatVersion3:
		*m = FormatVersion(val)
		return nil
	default:
		return fmt.Errorf("format_version '%v' not valid (allowed: 1, 2, 3)", val)
	}
}

// UnmarshalJSON accepts both the array form ([1, 0, 0]) and the
// string form ("1.0.0"). Note that it is always written back as an array.
func (v *Version) UnmarshalJSON(data []byte) error {
	var arr []int
	if err := json.Unmarshal(data, &arr); err == nil {
		if len(arr) != 3 {
			return fmt.Errorf("version must have exactly 3 numbers, got %d", len(arr))
		}

		for _, n := range arr {
			if n < 0 {
				return fmt.Errorf("version numbers cannot be negative, got %d", n)
			}
		}

		copy(v[:], arr)

		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf(`version must be an array of three integers (e.g. [1, 0, 0]) or a string (e.g. "1.0.0")`)
	}

	parts := strings.Split(str, ".")
	if len(parts) != 3 {
		return fmt.Errorf("version string %q must look like 1.0.0", str)
	}

	var parsed Version

	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return fmt.Errorf("version string %q must look like 1.0.0", str)
		}

		parsed[i] = n
	}

	*v = parsed

	return nil
}

func (d *Dependency) UnmarshalJSON(data []byte) error {
	type Alias Dependency
	var aux Alias

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	hasUUID := aux.UUID != ""
	hasModule := aux.ModuleName != ""

	if hasUUID && hasModule {
		return fmt.Errorf("invalid dependency: has 'uuid' and 'module_name' declared at the same time")
	}
	if !hasUUID && !hasModule {
		return fmt.Errorf("invalid dependency: must have 'uuid' or 'module_name'")
	}

	*d = Dependency(aux)
	return nil
}

func (pt *ProductType) UnmarshalJSON(b []byte) error {
	var val string
	if err := json.Unmarshal(b, &val); err != nil {
		return fmt.Errorf("product_type must be a string")
	}

	if val != string(ProductTypeAddon) {
		return fmt.Errorf("product_type '%s' is invalid (allowed: 'addon')", val)
	}

	*pt = ProductType(val)
	return nil
}

// UnmarshalJSON only accepts the array form ([1, 26, 0]); strings are rejected.
func (v *EngineVersion) UnmarshalJSON(data []byte) error {
	var arr []int
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("min_engine_version must be an array of three integers, e.g. [1, 26, 0]")
	}

	if len(arr) != 3 {
		return fmt.Errorf("min_engine_version must have exactly 3 numbers, got %d", len(arr))
	}

	for _, n := range arr {
		if n < 0 {
			return fmt.Errorf("min_engine_version numbers cannot be negative, got %d", n)
		}
	}

	copy(v[:], arr)

	return nil
}
