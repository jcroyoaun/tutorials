package main

import (
	"context"       // <--- NUEVO
	"encoding/json" // <--- NUEVO
	"flag"
	"fmt"
	"os"            // <--- NUEVO

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // <--- NUEVO
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// 1. Setup básico (SE QUEDA IGUAL)
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		path := home + "/.kube/config"
		kubeconfig = flag.String("kubeconfig", path, "kubeconfig file")
	}
	flag.Parse()

	// 2. Conexión (SE QUEDA IGUAL)
	config, _ := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	clientset, _ := kubernetes.NewForConfig(config)

	// --- AQUI EMPIEZA LO NUEVO (Borraste lo de la versión) ---

	fmt.Println("👀 Vigilando Namespaces... (Crea o borra uno para ver magia)")

	// 3. Crear el Watcher
	// Le decimos a K8s: "Avísame de CUALQUIER cambio en namespaces"
	watcher, _ := clientset.CoreV1().Namespaces().Watch(context.TODO(), metav1.ListOptions{})

	// 4. Loop Infinito (Reactivo)
	// Este canal se desbloquea cada vez que pasa algo en el cluster
	for event := range watcher.ResultChan() {
		
		// Opcional: Mostrar qué pasó
		fmt.Printf(">> Evento detectado: %s\n", event.Type)

		// 5. Generar el JSON del estado actual
		printNamespaceJSON(clientset)
	}
}

// Función auxiliar para no ensuciar el main
func printNamespaceJSON(clientset *kubernetes.Clientset) {
	// a) Listamos todo lo que existe AHORITA
	list, _ := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})

	// b) Sacamos solo los nombres (string array)
	var names []string
	for _, item := range list.Items {
		names = append(names, item.Name)
	}

	// c) Imprimimos JSON bonito a STDOUT
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(names)
}
