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

// --- NUEVO EN V5: Definimos la estructura del objeto ---
type AppInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"` // Aquí va la imagen
}

func main() {
	// --- 1. LEER ENV VARS (IGUAL QUE V4) ---
	envVar := os.Getenv("TARGET_NAMESPACES")
	if envVar == "" {
		envVar = "default"
	}

	targets := strings.Split(envVar, ",")
	for i := range targets {
		targets[i] = strings.TrimSpace(targets[i])
	}

	// --- 2. KUBERNETES SETUP (IGUAL QUE V4) ---
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		path := home + "/.kube/config"
		kubeconfig = flag.String("kubeconfig", path, "path")
	}
	flag.Parse()

	config, _ := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	clientset, _ := kubernetes.NewForConfig(config)

	fmt.Printf("🎯 Target Namespaces: %v\n", targets)
	fmt.Println("👀 V5: Extrayendo versiones de imágenes...")

	// --- 3. WATCHER GLOBAL (IGUAL QUE V4) ---
	watcher, _ := clientset.AppsV1().Deployments("").Watch(context.TODO(), metav1.ListOptions{})

	// --- 4. EVENT LOOP (IGUAL QUE V4) ---
	for event := range watcher.ResultChan() {
		fmt.Printf("\n>> Evento detectado: %s\n", event.Type)
		printFilteredJSON(clientset, targets)
	}
}

// --- LÓGICA V5: AHORA SACAMOS LA IMAGEN ---
func printFilteredJSON(clientset *kubernetes.Clientset, targets []string) {
	
	// CAMBIO 1: El mapa ahora guarda una lista de OBJETOS (AppInfo), no strings
	worldState := make(map[string][]AppInfo)

	for _, ns := range targets {
		list, _ := clientset.AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{})

		var apps []AppInfo // Lista de objetos
		
		for _, d := range list.Items {
			// CAMBIO 2: Llenamos el struct sacando la imagen del container 0
			apps = append(apps, AppInfo{
				Name:    d.Name,
				Version: d.Spec.Template.Spec.Containers[0].Image,
			})
		}

		if len(apps) > 0 {
			worldState[ns] = apps
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(worldState)
}
