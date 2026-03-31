package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "ow",
	Short: "OpenWebUI + Ollama setup tool",
	Long:  "Automates installation and management of OpenWebUI and Ollama with GPU support.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and build info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ow %s %s %s %s/%s\n", version, commit, date, runtime.GOOS, runtime.GOARCH)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print resolved configuration",
	RunE:  runConfigShow,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
}
