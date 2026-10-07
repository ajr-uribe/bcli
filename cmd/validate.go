package cmd

import (
	"fmt"

	"bcli/internal/manifest"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <manifest.json>",
	Short: "Validate a manifest.json against Bedrock add-on rules",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := manifest.Read(args[0])
		if err != nil {
			return err
		}

		issues := manifest.Validate(m)
		if len(issues) == 0 {
			fmt.Println("OK: no issues found.")
			return nil
		}

		fmt.Printf("Found %d issue(s):\n", len(issues))
		for _, issue := range issues {
			fmt.Println(issue.String())
		}
		return fmt.Errorf("validation failed")
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
