package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"bcli/internal/manifest"
)

var (
	generateOutput string
	generateForce  bool
)

var manifestGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Interactively generate a manifest.json",
	Args:  cobra.NoArgs,
	RunE:  runManifestGenerate,
}

func init() {
	flags := manifestGenerateCmd.Flags()

	flags.StringVarP(&generateOutput, "output", "o", "manifest.json", "output file path")
	flags.BoolVarP(&generateForce, "force", "f", false, "overwrite the output file if it already exists")

	manifestCmd.AddCommand(manifestGenerateCmd)
}

func runManifestGenerate(cmd *cobra.Command, _ []string) error {
	// Fail before prompting so the user does not lose their answers.
	if !generateForce {
		if _, err := os.Stat(generateOutput); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", generateOutput)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	m, err := manifest.InteractiveGenerate()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Fprintln(cmd.ErrOrStderr(), "Aborted.")

			return nil
		}

		return err
	}

	if issues := manifest.Validate(m); len(issues) > 0 {
		for _, issue := range issues {
			fmt.Fprintln(cmd.ErrOrStderr(), issue)
		}

		return fmt.Errorf("generated manifest has %d validation issue(s)", len(issues))
	}

	data, err := encodeManifest(m)
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(generateOutput), 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(generateOutput, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", generateOutput, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Manifest written to %s\n", generateOutput)

	return nil
}

// encodeManifest marshals the manifest as indented JSON without HTML escaping,
// so characters like & or < in descriptions are kept as-is.
func encodeManifest(m *manifest.Manifest) ([]byte, error) {
	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(m); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
