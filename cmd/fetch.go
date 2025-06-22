package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/wycleffsean/nostos/pkg/kube"
)

// fetchCmd represents the "fetch" command
var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch Kubernetes resource specifications and store them in the registry",
	Long: `The fetch command retrieves Kubernetes type definitions from the cluster 
and stores them in the in-memory registry, making them available for the compiler.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger, _ := zap.NewDevelopmentConfig().Build()
		log := logger.Sugar()
		registry, err := kube.FetchAndFillRegistry()
		if err != nil {
			log.Fatalf("Failed to fetch kubernetes resources: %v\n", err)
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
