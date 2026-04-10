package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "didi",
	Short: "Didi CLI - Jira ticket management tool",
	Long:  `A CLI tool for managing Jira tickets with local workspace integration.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(descCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(kanbanCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(saveCmd)
}
