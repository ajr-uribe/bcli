package pack

// Tests for pack scaffolding: folder layouts, manifest linking, and
// refusal to clobber existing directories.

import (
	"os"
	"path/filepath"
	"testing"

	"bcli/internal/manifest"
)

func TestCreateBPMinimalLayout(t *testing.T) {
	dir := t.TempDir()

	if err := CreateBP(dir, Options{Name: "Demo"}); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{
		"Demo/manifest.json",
		"Demo/items",
		"Demo/blocks",
		"Demo/entities",
	} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}

	m, err := manifest.Read(filepath.Join(dir, "Demo", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if issues := manifest.Validate(m); len(issues) != 0 {
		t.Fatalf("expected valid manifest, got %v", issues)
	}
}

func TestCreateBPWithScript(t *testing.T) {
	dir := t.TempDir()

	opts := Options{
		Name:          "Demo",
		Script:        true,
		ScriptModules: []manifest.ScriptDependency{{Name: "server", Version: "2.0.0"}},
	}
	if err := CreateBP(dir, opts); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "Demo", "scripts", "main.js")); err != nil {
		t.Errorf("expected scripts/main.js to exist: %v", err)
	}
}

func TestCreateRPAdvancedLayout(t *testing.T) {
	dir := t.TempDir()

	if err := CreateRP(dir, Options{Name: "Demo", Advanced: true}); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{
		"Demo/manifest.json",
		"Demo/blocks.json",
		"Demo/materials",
		"Demo/render_controllers",
		"Demo/subpacks",
	} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}
}

func TestCreateBothLinksManifests(t *testing.T) {
	dir := t.TempDir()

	if err := CreateBoth(dir, Options{Name: "Demo"}); err != nil {
		t.Fatal(err)
	}

	bp, err := manifest.Read(filepath.Join(dir, "Demo_bp", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	rp, err := manifest.Read(filepath.Join(dir, "Demo_rp", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}

	if len(bp.Dependencies) != 1 || bp.Dependencies[0].UUID != rp.Header.UUID {
		t.Errorf("expected BP to depend on RP header UUID, got %v", bp.Dependencies)
	}
	if len(rp.Dependencies) != 1 || rp.Dependencies[0].UUID != bp.Header.UUID {
		t.Errorf("expected RP to depend on BP header UUID, got %v", rp.Dependencies)
	}
	if issues := manifest.Validate(bp); len(issues) != 0 {
		t.Errorf("expected valid BP manifest, got %v", issues)
	}
	if issues := manifest.Validate(rp); len(issues) != 0 {
		t.Errorf("expected valid RP manifest, got %v", issues)
	}
}

func TestCreateRefusesExistingDir(t *testing.T) {
	dir := t.TempDir()

	if err := CreateBP(dir, Options{Name: "Demo"}); err != nil {
		t.Fatal(err)
	}
	if err := CreateBP(dir, Options{Name: "Demo"}); err == nil {
		t.Fatal("expected error when target directory already exists")
	}
}
