package manifest

import (
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
