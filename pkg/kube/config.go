package kube

import (
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	// "k8s.io/client-go/tools/clientcmd/api"
	"github.com/spf13/viper"
)

// LoadKubeConfig loads Kubernetes client configuration with kubectl-like behavior.
func LoadKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Check if a kubeconfig file is explicitly provided via --kubeconfig or env var
	kubeconfigPath := viper.GetString("kubeconfig")
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}

	// Load kubeconfig file if provided, otherwise fallback to default location
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}

	// Load Kubernetes context from --context flag or default config
	overrides := &clientcmd.ConfigOverrides{}
	if context := viper.GetString("context"); context != "" {
		overrides.CurrentContext = context
	}

	// Build config using kubeconfig file and overrides
	config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not load Kubernetes client config: %w", err)
	}

	return config, nil
}

// CurrentContext returns the name of the currently selected Kubernetes context.
func CurrentContext() (string, error) {
	kubeconfigPath := viper.GetString("kubeconfig")
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}

	overrides := &clientcmd.ConfigOverrides{}
	if context := viper.GetString("context"); context != "" {
		overrides.CurrentContext = context
	}

	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).RawConfig()
	if err != nil {
		return "", fmt.Errorf("could not load Kubernetes client config: %w", err)
	}

	if overrides.CurrentContext != "" {
		return overrides.CurrentContext, nil
	}
	return rawConfig.CurrentContext, nil
}
