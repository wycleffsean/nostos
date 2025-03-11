package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/wycleffsean/nostos/pkg/cluster"
	"github.com/wycleffsean/nostos/pkg/planner"
	"github.com/wycleffsean/nostos/pkg/registry"
)

func prepareTypeRegistry(config *cluster.ClientConfig) (*registry.TypeRegistry, error) {
	registry := registry.NewTypeRegistry()
	err := registry.AppendRegistryFromCRDs(config)
	if err != nil {
		return registry, err
	}

	return registry, nil
}

// planCmd represents the plan command.
var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Compute the changes to be applied without making any changes.",
	Long:  `The plan command calculates the modifications to be made to your cluster, letting you preview what will happen before actually applying changes.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get Kubernetes client configuration using the helper defined in root.go.
		config, err := GetClientConfig()
		if err != nil {
			log.Fatalf("No kubeconfig ¯\\_(ツ)_/¯: %v", err)
		}

		registry, err := prepareTypeRegistry(config)
		if err != nil {
			log.Fatalf("fetching types failed: %v", err)
		}

		registry.Types.Range(func(key, value any) bool {
			fmt.Printf("key: %v\n", key)
			return true
		})

		// Build the plan from the cluster state.
		plan, err := planner.BuildPlanFromCluster(config)
		if err != nil {
			log.Fatalf("Error building plan from cluster: %v", err)
		}

		// Nicely print the plan results.
		fmt.Println("Plan:")
		if len(plan.Resources) == 0 {
			fmt.Println("No resources found.")
			return
		}
		for _, res := range plan.Resources {
			// Extract the resource name from metadata.
			name, ok := res.Metadata["name"].(string)
			if !ok {
				name = "unknown"
			}
			fmt.Printf("- Kind: %-20s API Version: %-10s Name: %s\n", res.Kind, res.APIVersion, name)
		}
	},
}

func init() {
	RootCmd.AddCommand(planCmd)
}
