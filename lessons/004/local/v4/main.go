package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// --- 1. LEER ENV VARS (CONFIGURACIÓN) ---
	envVar := os.Getenv("TARGET_NAMESPACES")
	if envVar == "" {
		envVar = "default" // Fallback por si se te olvida el export
	}
	
	// Convertimos string "ns1,ns2" a slice ["ns1", "ns2"]
	targets := strings.Split(envVar, ",")
	for i := range targets {
		targets[i] = strings.TrimSpace(targets[i])
	}

	// --- 2. KUBERNETES SETUP ---
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		path := home + "/.kube/config"
		kubeconfig = flag.String("kubeconfig", path, "path")
	}
	flag.Parse()

	config, _ := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	clientset, _ := kubernetes.NewForConfig(config)

	fmt.Printf("🎯 Target Namespaces: %v\n", targets)
	fmt.Println("👀 Iniciando Watcher (Ctrl+C para salir)...")

	// --- 3. WATCHER GLOBAL ---
	// Vigilamos TODO ("") para enterarnos de cualquier cambio en el cluster
	watcher, _ := clientset.AppsV1().Deployments("").Watch(context.TODO(), metav1.ListOptions{})

	// --- 4. EVENT LOOP ---
	for event := range watcher.ResultChan() {
		// Usamos la variable event para que el compilador no chingue
		// y de paso vemos actividad en la terminal
		fmt.Printf("\n>> Evento detectado: %s\n", event.Type)
		
		// Regeneramos el JSON filtrado
		printFilteredJSON(clientset, targets)
	}
}

// Lógica de filtrado y JSON
func printFilteredJSON(clientset *kubernetes.Clientset, targets []string) {
	// Mapa: Namespace -> Lista de Apps
	worldState := make(map[string][]string)

	// Iteramos SOLO los namespaces que nos interesan (targets)
	for _, ns := range targets {
		// Listamos los deployments de ese namespace especifico
		list, _ := clientset.AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{})
		
		var apps []string
		for _, d := range list.Items {
			apps = append(apps, d.Name)
		}
		
		// Si encontramos apps, las guardamos en el mapa
		if len(apps) > 0 {
			worldState[ns] = apps
		}
	}

	// Imprimir JSON bonito
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(worldState)
}
