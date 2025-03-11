package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/wycleffsean/nostos/pkg/cluster"
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

// initConfig initializes configuration and loads the kubeconfig file.
func initConfig() {
	var cfgFile string
	if kubeconfig != "" {
		cfgFile = kubeconfig
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Unable to determine home directory:", err)
			return
		}
		cfgFile = filepath.Join(home, ".kube", "config")
	}
	// Bind the flag value explicitly to Viper so that viper.GetString("kubeconfig") works.
	viper.Set("kubeconfig", cfgFile)
	viper.SetConfigFile(cfgFile)
	// If there's no extension, set config type to YAML.
	if filepath.Ext(cfgFile) == "" {
		viper.SetConfigType("yaml")
	}

	// Attempt to read the kubeconfig file; do not fail if it doesn't exist.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Kubeconfig file not found, continuing without it.")
		} else {
			fmt.Println("Error reading kubeconfig file:", err)
		}
	} else {
		fmt.Println("Using kubeconfig file:", viper.ConfigFileUsed())
	}
}

// GetClientConfig builds a Kubernetes REST configuration using the kubeconfig path
// provided via Viper.
func GetClientConfig() (*cluster.ClientConfig, error) {
	// Retrieve kubeconfig from Viper.
	kubeconfigPath := viper.GetString("kubeconfig")
	// Build the config from flags. The empty master URL tells it to use the kubeconfig.
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}
	return config, nil
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
	// Ensure configuration is initialized on command execution.
	cobra.OnInitialize(initConfig)
	// Add persistent flag for kubeconfig.
	RootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to the kubeconfig file (default is $HOME/.kube/config)")
	// Also parse standard flags (required for klog/client-go logging adjustments).
	flag.Parse()
}
