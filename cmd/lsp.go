package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// lspCmd represents the language server command.
var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Launch the language server for Nostos.",
	Long:  `The lsp command starts the built-in language server, enabling features like autocompletion, diagnostics, and more in supported editors.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting language server...")
		// TODO: Insert language server initialization logic here.
	},
}

func init() {
	RootCmd.AddCommand(lspCmd)
}
