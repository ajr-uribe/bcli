package manifest

// Tests for the manifest package, grouped by concern:
//   - UUID filling (EnsureUUIDs, FillMissingUUIDs, FillUUIDsInFile)
//   - Business validation (Validate)
//   - Disk I/O round-trips (Read + Write, unknown-field preservation)

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// ---------------------------------------------------------
// Test helpers
// ---------------------------------------------------------

func validManifest() *Manifest {
	return &Manifest{
		FormatVersion: FormatVersion2,
		Header:        Header{Name: "pack", UUID: "458f6e14-40fd-4109-99aa-2ed058eff1fd", MinEngineVersion: EngineVersion{1, 26, 0}},
		Modules:       []Module{{Type: ModuleTypeData, UUID: "efa63950-602a-4c10-af00-37ec0be8f346"}},
	}
}

// ---------------------------------------------------------
// UUID filling
// ---------------------------------------------------------

func TestEnsureUUIDsFillsMissing(t *testing.T) {
	m := &Manifest{
		Header: Header{Name: "pack"},
		Modules: []Module{
			{Type: ModuleTypeData, UUID: "existing-uuid"},
			{Type: ModuleTypeResources, UUID: "  "},
		},
	}

	added := EnsureUUIDs(m)

	if added != 2 {
		t.Fatalf("expected 2 UUIDs generated, got %d", added)
	}
	if m.Header.UUID == "" {
		t.Error("expected header UUID to be filled")
	}
	if m.Modules[0].UUID != "existing-uuid" {
		t.Error("expected existing module UUID to be preserved")
	}
	if strings.TrimSpace(m.Modules[1].UUID) == "" {
		t.Error("expected blank module UUID to be replaced")
	}
}

func TestEnsureUUIDsNoOp(t *testing.T) {
	m := &Manifest{
		Header:  Header{UUID: "a"},
		Modules: []Module{{UUID: "b"}},
	}
	if added := EnsureUUIDs(m); added != 0 {
		t.Fatalf("expected 0 UUIDs generated, got %d", added)
	}
}

func TestFillMissingUUIDsReportsChanges(t *testing.T) {
	m := &Manifest{
		Header:  Header{Name: "pack"},
		Modules: []Module{{Type: ModuleTypeData}},
	}

	changes := FillMissingUUIDs(m)

	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
	if changes[0].Field != "header.uuid" || changes[1].Field != "modules[0].uuid" {
		t.Errorf("unexpected fields: %v", changes)
	}
	if changes[0].UUID != m.Header.UUID || changes[1].UUID != m.Modules[0].UUID {
		t.Error("expected changes to reference the generated UUIDs")
	}
}

func TestFillUUIDsInFileDryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	input := `{
  "format_version": 2,
  "header": {"name": "pack", "description": "d", "uuid": "", "version": [1, 0, 0]},
  "modules": []
}
`
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	changes, err := FillUUIDsInFile(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	out, _ := os.ReadFile(path)
	if string(out) != input {
		t.Error("dry-run modified the file on disk")
	}
}

// ---------------------------------------------------------
// Business validation
// ---------------------------------------------------------

func TestValidateValidManifest(t *testing.T) {
	if issues := Validate(validManifest()); len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}

func TestValidateRequiresModules(t *testing.T) {
	m := validManifest()
	m.Modules = nil
	issues := Validate(m)
	if len(issues) == 0 || !strings.Contains(issues[0].Message, "1 or 2 modules") {
		t.Fatalf("expected module count issue, got %v", issues)
	}
}

func TestValidateRejectsDataAndResourcesTogether(t *testing.T) {
	m := validManifest()
	m.Modules = append(m.Modules, Module{Type: ModuleTypeResources, UUID: "43e45286-3df6-4b27-b6e3-e64796a5bdab"})
	found := false
	for _, issue := range Validate(m) {
		if strings.Contains(issue.Message, "both 'data' and 'resources'") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected data+resources issue")
	}
}

func TestValidateDependencyXOR(t *testing.T) {
	m := validManifest()
	m.Dependencies = []Dependency{
		{UUID: "a", ModuleName: "b", Version: "1.0.0"},
		{Version: "1.0.0"},
	}
	issues := Validate(m)
	if len(issues) != 2 {
		t.Fatalf("expected 2 dependency issues, got %v", issues)
	}
}

// ---------------------------------------------------------
// Disk I/O round-trips
// ---------------------------------------------------------

// TestWritePreservesUnknownFields ensures Read+Write keeps fields the
// typed struct does not know about, at the top level and nested.
func TestWritePreservesUnknownFields(t *testing.T) {
	input := `{
  "format_version": 2,
  "header": {
    "name": "pack",
    "description": "desc",
    "uuid": "458f6e14-40fd-4109-99aa-2ed058eff1fd",
    "version": [1, 0, 0],
    "min_engine_version": [1, 26, 0],
    "custom_header_field": "keep-me"
  },
  "modules": [
    {
      "type": "data",
      "uuid": "efa63950-602a-4c10-af00-37ec0be8f346",
      "version": [1, 0, 0],
      "capabilities": ["chemistry"]
    }
  ],
  "world_template_options": { "template_uuid": "abc" }
}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(path, m); err != nil {
		t.Fatal(err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var want, got map[string]any
	if err := json.Unmarshal([]byte(input), &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("round-trip changed the document.\nwant: %v\ngot:  %v", want, got)
	}
}

func TestWriteMergesKnownFieldUpdates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	input := `{
  "format_version": 2,
  "header": {"name": "pack", "description": "d", "uuid": "", "version": [1, 0, 0]},
  "modules": [{"type": "data", "uuid": "", "version": [1, 0, 0]}],
  "extra": true
}
`
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	EnsureUUIDs(m)
	if err := Write(path, m); err != nil {
		t.Fatal(err)
	}

	out, _ := os.ReadFile(path)
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["extra"] != true {
		t.Error("expected unknown top-level field to survive")
	}
	header := doc["header"].(map[string]any)
	if header["uuid"] == "" {
		t.Error("expected header uuid to be updated")
	}
}
