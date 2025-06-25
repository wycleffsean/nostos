package cmd

import (
	"flag"
	"fmt"
	"os"
	// "path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/wycleffsean/nostos/pkg/workspace"
)

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

var ignoreSystemNamespace bool
var ignoreClusterScoped bool

// Execute runs the root command.
func Execute() {
	// Bind the kubeconfig flag to Viper.
	if err := viper.BindPFlag("kubeconfig", RootCmd.PersistentFlags().Lookup("kubeconfig")); err != nil {
		cobra.CheckErr(err)
	}

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
	RootCmd.PersistentFlags().String("workspace-dir", "", "Nostos workspace directory")
	RootCmd.PersistentFlags().BoolVar(&ignoreSystemNamespace, "ignore-system-namespace", true, "Ignore resources in system namespaces")
	RootCmd.PersistentFlags().BoolVar(&ignoreClusterScoped, "ignore-cluster-scoped", true, "Ignore cluster-scoped resources")

	// Bind flags to Viper for centralized config handling
	cobra.CheckErr(viper.BindPFlag("kubeconfig", RootCmd.PersistentFlags().Lookup("kubeconfig")))
	cobra.CheckErr(viper.BindPFlag("context", RootCmd.PersistentFlags().Lookup("context")))
	cobra.CheckErr(viper.BindPFlag("workspace_dir", RootCmd.PersistentFlags().Lookup("workspace-dir")))
	cobra.CheckErr(viper.BindPFlag("ignore_cluster_scoped", RootCmd.PersistentFlags().Lookup("ignore-cluster-scoped")))

	RootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		workspace.Set(viper.GetString("workspace_dir"))
	}
}
