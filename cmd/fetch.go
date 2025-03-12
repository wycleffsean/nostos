package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/wycleffsean/nostos/pkg/kube"
	"github.com/wycleffsean/nostos/pkg/types"
)

// fetchCmd represents the "fetch" command
var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch Kubernetes resource specifications and store them in the registry",
	Long: `The fetch command retrieves Kubernetes type definitions from the cluster 
and stores them in the in-memory registry, making them available for the compiler.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create type registry
		registry := types.NewRegistry()

		// Fetch Kubernetes specifications and populate the registry
		fmt.Println("Fetching Kubernetes resource specifications...")
		if err := kube.FetchSpecifications(registry); err != nil {
			log.Fatalf("Failed to fetch Kubernetes specifications: %v", err)
		}

		// Display a summary of fetched types
		types := registry.ListTypes()
		fmt.Printf("Fetched %d Kubernetes types.\n", len(types))
		for _, typeDef := range types {
			if typeDef.Kind == "Pod" {
				fmt.Printf("%+v\n", typeDef)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(fetchCmd)
}
