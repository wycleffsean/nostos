package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// applyCmd represents the apply command.
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the computed changes to your cluster.",
	Long:  `The apply command executes the modifications determined during the plan phase, updating your Kubernetes cluster accordingly.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running apply phase...")
		// Optionally retrieve kubeconfig:
		// kubeconfigPath := viper.GetString("kubeconfig")
		// TODO: Insert apply logic here (e.g., using k8s.io/client-go to update resources)
	},
}

func init() {
	RootCmd.AddCommand(applyCmd)
}
