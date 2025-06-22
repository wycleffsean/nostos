package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wycleffsean/nostos/internal/lsp"
	"go.uber.org/zap"
)

// lspCmd represents the language server command.
var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Launch the language server for Nostos.",
	Long:  `The lsp command starts the built-in language server, enabling features like autocompletion, diagnostics, and more in supported editors.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger, _ := zap.NewDevelopmentConfig().Build()
		// registry := FetchAndFillRegistry(logger.Sugar())
		// _ = registry
		lsp.StartServer(logger)
	},
}

func init() {
	RootCmd.AddCommand(lspCmd)
}
