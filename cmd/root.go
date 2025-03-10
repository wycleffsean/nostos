package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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

	// Tell Viper which file to read.
	viper.SetConfigFile(cfgFile)
	// If there's no file extension, explicitly set the config type to YAML.
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
// provided via Viper (either default or overridden by the --kubeconfig flag).
func GetClientConfig() (*rest.Config, error) {
	// Get the kubeconfig path from Viper.
	kubeconfigPath := viper.GetString("kubeconfig")
	// Build the config from flags. Passing an empty master URL will let the function use the kubeconfig.
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}
	return config, nil
}

// Execute runs the root command.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Initialize configuration when any command is executed.
	cobra.OnInitialize(initConfig)

	// Add a persistent flag for specifying kubeconfig.
	RootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to the kubeconfig file (default is $HOME/.kube/config)")
}
