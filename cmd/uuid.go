package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"bcli/internal/manifest"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var uuidCmd = &cobra.Command{
	Use:   "uuid",
	Short: "Generate standalone UUIDs or insert missing UUIDs into a manifest.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Mode 1: Insert missing UUIDs into a manifest (-i / --insert)
		if cmd.Flags().Changed("insert") {
			filePath, _ := cmd.Flags().GetString("insert")
			if strings.TrimSpace(filePath) == "" {
				return fmt.Errorf("flag '--insert' requires a non-empty file path argument")
			}
			cleanPath := filepath.Clean(strings.TrimSpace(filePath))

			// 1. Read manifest
			m, err := manifest.Read(cleanPath)
			if err != nil {
				return fmt.Errorf("error reading manifest: %w", err)
			}

			// 2. Insert missing UUIDs
			addedCount := manifest.EnsureUUIDs(m)

			if addedCount == 0 {
				fmt.Println("No missing UUIDs found. All header and module UUIDs are already set.")
				return nil
			}

			dryRun, _ := cmd.Flags().GetBool("dry-run")
			if dryRun {
				fmt.Printf("[dry-run] Would insert %d missing UUID(s) into '%s'\n", addedCount, cleanPath)
				return nil
			}

			// 3. Save changes
			if err := manifest.Write(cleanPath, m); err != nil {
				return fmt.Errorf("error updating manifest: %w", err)
			}

			fmt.Printf("Successfully inserted %d missing UUID(s) into '%s'\n", addedCount, cleanPath)
			return nil
		}

		// Mode 2: Generate and print UUIDs to the console (default)
		count, _ := cmd.Flags().GetInt("count")
		if count < 1 {
			return fmt.Errorf("--count must be a positive number, got %d", count)
		}
		for i := 0; i < count; i++ {
			fmt.Println(uuid.New().String())
		}
		return nil
	},
}

func init() {
	uuidCmd.Flags().StringP("insert", "i", "", "Path to manifest.json to insert missing UUIDs into")
	uuidCmd.Flags().IntP("count", "c", 1, "Number of UUIDs to generate")
	uuidCmd.Flags().Bool("dry-run", false, "Show what would change without writing to disk")

	rootCmd.AddCommand(uuidCmd)
}
