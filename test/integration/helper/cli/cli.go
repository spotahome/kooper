package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" // Load oidc authentication when creating the kubernetes client.
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// GetK8sClient returns k8s client.
func GetK8sClient(kubehome string) (kubernetes.Interface, error) {
	// Try fallbacks.
	if kubehome == "" {
		if kubehome = os.Getenv("KUBECONFIG"); kubehome == "" {
			kubehome = filepath.Join(homedir.HomeDir(), ".kube", "config")
		}
	}

	// Load kubernetes local connection.
	config, err := clientcmd.BuildConfigFromFlags("", kubehome)
	if err != nil {
		return nil, fmt.Errorf("could not load configuration: %s", err)
	}

	// Get the client.
	k8sCli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return k8sCli, nil
}
