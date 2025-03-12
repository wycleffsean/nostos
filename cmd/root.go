package cmd

import (
	"flag"
	"fmt"
	"os"
	// "path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var kubeconfig string

// RootCmd is the base command for Nostos.
var RootCmd = &cobra.Command{
	Use:   "nostos",
	Short: "Nostos is a Helm replacement for Kubernetes with its own DSL.",
	Long: `Nostos is a programming language designed for Kubernetes configuration,
offering a plan/apply workflow similar to Terraform, as well as an integrated language server.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Nostos CLI. Use --help for more information.")
	},
}

// Execute runs the root command.
func Execute() {
	// Bind the kubeconfig flag to Viper.
	viper.BindPFlag("kubeconfig", RootCmd.PersistentFlags().Lookup("kubeconfig"))

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// TODO: is this needed?
	// parse standard flags (required for klog/client-go logging adjustments).
	flag.Parse()

	// Global flags available for all commands
	RootCmd.PersistentFlags().String("kubeconfig", "", "Path to kubeconfig file")
	RootCmd.PersistentFlags().String("context", "", "Kubernetes context to use")

	// Bind flags to Viper for centralized config handling
	viper.BindPFlag("kubeconfig", RootCmd.PersistentFlags().Lookup("kubeconfig"))
	viper.BindPFlag("context", RootCmd.PersistentFlags().Lookup("context"))
}
