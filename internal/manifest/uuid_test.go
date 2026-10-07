package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
