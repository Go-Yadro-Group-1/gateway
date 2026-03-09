package main

import (
	"os"

	"github.com/Go-Yadro-Group-1/gateway/cmd/internal/cli/server"
	"github.com/spf13/cobra"
)

func main() {
	//nolint:exhaustruct
	rootCmd := &cobra.Command{
		Use:   "jira-gateway",
		Short: "Jira Gateway",
		Long:  "Jira Gateway is a gateway for Jira Analyzer.",
	}

	rootCmd.AddCommand(server.NewCommand())

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
