package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// planCmd represents the plan command.
var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Compute the changes to be applied without making any changes.",
	Long:  `The plan command calculates the modifications to be made to your cluster, letting you preview what will happen before actually applying changes.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running plan phase...")
		// Retrieve the kubeconfig path if needed:
		// kubeconfigPath := viper.GetString("kubeconfig")
		// TODO: Insert plan logic here (e.g., parsing files, computing diffs, etc.)
	},
}

func init() {
	RootCmd.AddCommand(planCmd)
}
