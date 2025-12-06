package main

import (
	"flag"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// 1. Setup básico
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		path := home + "/.kube/config"
		kubeconfig = flag.String("kubeconfig", path, "kubeconfig file")
	}
	flag.Parse()

	// 2. Conexión
	config, _ := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	clientset, _ := kubernetes.NewForConfig(config)

	// 3. Prueba de vida
	version, _ := clientset.Discovery().ServerVersion()
	fmt.Printf("✅ Conectado a K8s versión: %s\n", version)
}
