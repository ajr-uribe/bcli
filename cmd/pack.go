package cmd

import "github.com/spf13/cobra"

var packCmd = &cobra.Command{
	Use:   "pack",
	Short: "Scaffold new Minecraft Bedrock packs",
}

func init() {
	rootCmd.AddCommand(packCmd)
}
