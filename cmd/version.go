package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var gitVersion string

// lspCmd represents the language server command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get the git SHA for this build",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("build version:", gitVersion)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
