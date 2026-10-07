package manifest

import (
	"strings"
	"testing"
)

func validManifest() *Manifest {
	return &Manifest{
		FormatVersion: FormatVersion2,
		Header:        Header{Name: "pack", UUID: "uuid-1"},
		Modules:       []Module{{Type: ModuleTypeData, UUID: "uuid-2"}},
	}
}

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
	m.Modules = append(m.Modules, Module{Type: ModuleTypeResources, UUID: "uuid-3"})
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
