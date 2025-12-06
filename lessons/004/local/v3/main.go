package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// 1. Setup (SE QUEDA IGUAL)
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		path := home + "/.kube/config"
		kubeconfig = flag.String("kubeconfig", path, "kubeconfig file")
	}
	flag.Parse()

	config, _ := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	clientset, _ := kubernetes.NewForConfig(config)

	fmt.Println("👀 Vigilando Deployments en TODO el cluster...")

	// 2. EL CAMBIO: Watch Deployments (Namespace "" significa ALL)
	// Antes era: CoreV1().Namespaces()
	watcher, _ := clientset.AppsV1().Deployments("").Watch(context.TODO(), metav1.ListOptions{})

	// 3. Loop (SE QUEDA IGUAL)
	for event := range watcher.ResultChan() {
		// Opcional: Ver qué está pasando
		fmt.Printf(">> Cambio detectado (%s). Actualizando mapa...\n", event.Type)

		printDeploymentsJSON(clientset)
	}
}

// 4. EL CAMBIO: Lógica para agrupar por Namespace
func printDeploymentsJSON(clientset *kubernetes.Clientset) {
	// a) Pedimos TODOS los deployments del cluster
	list, _ := clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})

	// b) Creamos un Mapa: Llave=Namespace, Valor=ListaDeApps
	worldState := make(map[string][]string)

	for _, d := range list.Items {
		// Agrupamos: "En este namespace, agrega este deployment"
		worldState[d.Namespace] = append(worldState[d.Namespace], d.Name)
	}

	// c) Imprimimos JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(worldState)
}
