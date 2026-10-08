package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"bcli/internal/manifest"
	"bcli/internal/pack"
)

var (
	createPath        string
	createType        string
	createDescription string
	createAuthors     string
	createScript      bool
	createAdvanced    bool
	createModules     []string
)

var packCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Scaffold a new behavior pack, resource pack, or both",
	Long: `Scaffolds the folder tree and manifest.json for a new pack.

Types:
  bp    behavior pack only (/{name})
  rp    resource pack only (/{name})
  both  linked behavior + resource packs (/{name}_bp and /{name}_rp)

Examples:
  bcli pack create MyAddon
  bcli pack create MyAddon -t bp -s -m server@2.0.0
  bcli pack create MyAddon -t rp -a --authors "Alice, Bob"`,
	Args: cobra.ExactArgs(1),
	RunE: runPackCreate,
}

func init() {
	flags := packCreateCmd.Flags()

	flags.StringVarP(&createPath, "path", "p", ".", "parent directory to scaffold into")
	flags.StringVarP(&createType, "type", "t", "both", "pack type: bp, rp, or both")
	flags.StringVarP(&createDescription, "description", "d", "", "pack description")
	flags.StringVar(&createAuthors, "authors", "", "comma-separated author list")
	flags.BoolVarP(&createScript, "script", "s", false, "add a script module and scripts/main.js (behavior pack)")
	flags.BoolVarP(&createAdvanced, "advanced", "a", false, "add materials, render_controllers, and subpacks (resource pack)")
	flags.StringSliceVarP(&createModules, "module", "m", nil, "Script API dependency as name@version, e.g. server@2.0.0 (repeatable)")

	packCmd.AddCommand(packCreateCmd)
}

func runPackCreate(cmd *cobra.Command, args []string) error {
	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("pack name cannot be empty")
	}

	scriptDeps, err := parseScriptModules(createModules)
	if err != nil {
		return err
	}

	var authors []string
	if strings.TrimSpace(createAuthors) != "" {
		authors = strings.Split(createAuthors, ",")
	}

	opts := pack.Options{
		Name:          name,
		Description:   createDescription,
		Authors:       authors,
		Script:        createScript,
		Advanced:      createAdvanced,
		ScriptModules: scriptDeps,
	}

	switch strings.ToLower(strings.TrimSpace(createType)) {
	case "bp":
		if err := pack.CreateBP(createPath, opts); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Behavior pack created at %s\n", filepath.Join(createPath, name))
	case "rp":
		if err := pack.CreateRP(createPath, opts); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Resource pack created at %s\n", filepath.Join(createPath, name))
	case "both":
		if err := pack.CreateBoth(createPath, opts); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Packs created at %s and %s\n",
			filepath.Join(createPath, name+"_bp"), filepath.Join(createPath, name+"_rp"))
	default:
		return fmt.Errorf("invalid type %q (allowed: bp, rp, both)", createType)
	}

	return nil
}

// parseScriptModules parses repeatable name@version flags. Names may carry a
// scope ("@minecraft/server@2.0.0"), so the version is split off at the last
// "@". Deeper validation (name normalization, version format) happens in
// manifest.New.
func parseScriptModules(flags []string) ([]manifest.ScriptDependency, error) {
	deps := make([]manifest.ScriptDependency, 0, len(flags))
	for _, flag := range flags {
		// Scoped names ("@minecraft/server@2.0.0") contain "@", so split
		// at the last one; the name keeps its scope.
		idx := strings.LastIndex(flag, "@")
		if idx <= 0 {
			return nil, fmt.Errorf("invalid module %q, expected name@version (e.g. server@2.0.0)", flag)
		}
		name, version := flag[:idx], flag[idx+1:]
		if strings.TrimSpace(name) == "" || strings.TrimSpace(version) == "" {
			return nil, fmt.Errorf("invalid module %q, expected name@version (e.g. server@2.0.0)", flag)
		}
		deps = append(deps, manifest.ScriptDependency{Name: name, Version: version})
	}
	return deps, nil
}
