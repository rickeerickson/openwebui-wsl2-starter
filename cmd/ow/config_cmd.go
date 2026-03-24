package main

import (
	"fmt"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve(nil)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	fmt.Print(string(out))
	return nil
}
