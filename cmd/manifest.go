package cmd

import "github.com/spf13/cobra"

var manifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Manage Minecraft Bedrock manifest.json files",
}

func init() {
	rootCmd.AddCommand(manifestCmd)
}
