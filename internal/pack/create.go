// Package pack scaffolds new Minecraft Bedrock packs on disk: it creates
// the pack folder tree, generates each manifest.json through
// manifest.New, and writes everything to the filesystem.
package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bcli/internal/manifest"
)

// ---------------------------------------------------------
// Options
// ---------------------------------------------------------
// Options control what CreateBP, CreateRP, and CreateBoth scaffold.
// Script adds a script module plus scripts/main.js to a behavior pack.
// Advanced adds materials, render_controllers, and subpacks to a
// resource pack.

type Options struct {
	Name          string
	Description   string
	Authors       []string
	Script        bool
	Advanced      bool
	ScriptModules []manifest.ScriptDependency
}

func (o Options) validate() error {
	name := strings.TrimSpace(o.Name)
	if name == "" {
		return fmt.Errorf("pack name cannot be empty")
	}
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return fmt.Errorf("pack name %q is not a valid folder name", o.Name)
	}
	return nil
}

func (o Options) kind() manifest.PackKind {
	if o.Script {
		return manifest.KindScript
	}
	return manifest.KindBehavior
}

// ---------------------------------------------------------
// Public entry points
// ---------------------------------------------------------

// CreateBP scaffolds a minimal behavior pack:
//
//	/{name}
//	  scripts/main.js - only when Script is enabled
//	  items/
//	  blocks/
//	  entities/
//	  manifest.json
func CreateBP(path string, opts Options) error {
	if err := opts.validate(); err != nil {
		return err
	}

	root := filepath.Join(path, strings.TrimSpace(opts.Name))
	if err := ensureMissing(root); err != nil {
		return err
	}

	m, err := manifest.New(manifest.NewOptions{
		Name:        strings.TrimSpace(opts.Name),
		Description: opts.Description,
		Kind:        opts.kind(),
		Authors:     opts.Authors,
		ScriptDeps:  opts.ScriptModules,
	})
	if err != nil {
		return err
	}

	if err := scaffoldBPDisc(root, opts); err != nil {
		return err
	}

	return manifest.Write(filepath.Join(root, "manifest.json"), m)
}

// CreateRP scaffolds a minimal resource pack:
//
//	/{name}
//	  manifest.json
//	  blocks.json
//	  animations/
//	  animation_controllers/
//	  attachables/
//	  models/
//	  textures/
//
// Only when Advanced is enabled, it also creates:
//
//	materials/
//	render_controllers/
//	subpacks/
func CreateRP(path string, opts Options) error {
	if err := opts.validate(); err != nil {
		return err
	}

	root := filepath.Join(path, strings.TrimSpace(opts.Name))
	if err := ensureMissing(root); err != nil {
		return err
	}

	m, err := manifest.New(manifest.NewOptions{
		Name:        strings.TrimSpace(opts.Name),
		Description: opts.Description,
		Kind:        manifest.KindResource,
		Authors:     opts.Authors,
	})
	if err != nil {
		return err
	}

	if err := scaffoldRPDisc(root, opts); err != nil {
		return err
	}

	return manifest.Write(filepath.Join(root, "manifest.json"), m)
}

// CreateBoth scaffolds a behavior pack and a resource pack side by side
// (/{name}_bp and /{name}_rp, each with the layout described above) and
// links their manifests through their UUIDs in dependencies, so both
// packs load as a pair in-game.
func CreateBoth(path string, opts Options) error {
	if err := opts.validate(); err != nil {
		return err
	}

	name := strings.TrimSpace(opts.Name)
	bpRoot := filepath.Join(path, name+"_bp")
	rpRoot := filepath.Join(path, name+"_rp")
	if err := ensureMissing(bpRoot); err != nil {
		return err
	}
	if err := ensureMissing(rpRoot); err != nil {
		return err
	}

	bpManifest, err := manifest.New(manifest.NewOptions{
		Name:        name,
		Description: opts.Description,
		Kind:        opts.kind(),
		Authors:     opts.Authors,
		ScriptDeps:  opts.ScriptModules,
	})
	if err != nil {
		return err
	}

	rpManifest, err := manifest.New(manifest.NewOptions{
		Name:        name,
		Description: opts.Description,
		Kind:        manifest.KindResource,
		Authors:     opts.Authors,
	})
	if err != nil {
		return err
	}

	if err := manifest.LinkManifests(bpManifest, rpManifest); err != nil {
		return err
	}
	if issues := manifest.Validate(bpManifest); len(issues) > 0 {
		return fmt.Errorf("linked behavior manifest has %d validation issue(s): %v", len(issues), issues)
	}
	if issues := manifest.Validate(rpManifest); len(issues) > 0 {
		return fmt.Errorf("linked resource manifest has %d validation issue(s): %v", len(issues), issues)
	}

	if err := scaffoldBPDisc(bpRoot, opts); err != nil {
		return err
	}
	if err := scaffoldRPDisc(rpRoot, opts); err != nil {
		return err
	}

	if err := manifest.Write(filepath.Join(bpRoot, "manifest.json"), bpManifest); err != nil {
		return err
	}
	return manifest.Write(filepath.Join(rpRoot, "manifest.json"), rpManifest)
}

// ---------------------------------------------------------
// Disk helpers (unexported)
// ---------------------------------------------------------

func scaffoldBPDisc(root string, opts Options) error {
	dirs := []string{"items", "blocks", "entities"}
	if opts.Script {
		dirs = append(dirs, "scripts")
	}
	if err := makeDirs(root, dirs); err != nil {
		return err
	}

	if opts.Script {
		entry := "import { world } from \"@minecraft/server\";\n\nworld.afterEvents.worldLoad.subscribe(() => {\n  console.warn(\"Hello from " + jsStringBody(strings.TrimSpace(opts.Name)) + "!\");\n});\n"
		if err := writeFile(filepath.Join(root, "scripts", "main.js"), []byte(entry)); err != nil {
			return err
		}
	}

	return nil
}

func scaffoldRPDisc(root string, opts Options) error {
	dirs := []string{"animations", "animation_controllers", "attachables", "models", "textures"}
	if opts.Advanced {
		dirs = append(dirs, "materials", "render_controllers", "subpacks")
	}
	if err := makeDirs(root, dirs); err != nil {
		return err
	}

	// Empty starting point for block texture/sound mappings.
	return writeFile(filepath.Join(root, "blocks.json"), []byte("{}\n"))
}

func makeDirs(root string, dirs []string) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("creating directory %q: %w", root, err)
	}
	for _, dir := range dirs {
		p := filepath.Join(root, dir)
		if err := os.MkdirAll(p, 0o755); err != nil {
			return fmt.Errorf("creating directory %q: %w", p, err)
		}
	}
	return nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating parent directory for %q: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}
	return nil
}

// ensureMissing refuses to scaffold into a directory that already exists,
// so a create command never silently clobbers an existing pack.
func ensureMissing(dir string) error {
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("%q already exists, remove it or choose another name", dir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking %q: %w", dir, err)
	}
	return nil
}

// jsStringBody escapes a pack name so it can sit inside a double-quoted
// JavaScript string in the generated main.js.
func jsStringBody(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}
