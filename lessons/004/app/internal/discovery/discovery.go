// Package discovery provides Kubernetes deployment discovery capabilities.
package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AppInfo represents a single application with its version information.
type AppInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// WorldState represents the complete state of deployments across namespaces.
// The key is the namespace name, the value is a slice of AppInfo.
type WorldState map[string][]AppInfo

// OnStateChange is a callback function invoked when deployment state changes.
type OnStateChange func(state WorldState) error

// Discovery watches Kubernetes deployments and reports state changes.
type Discovery struct {
	client            *kubernetes.Clientset
	logger            *slog.Logger
	ignoredNamespaces map[string]bool
}

// New creates a new Discovery instance configured for in-cluster operation.
func New(logger *slog.Logger, ignoredNamespaces []string) (*Discovery, error) {
	// Use in-cluster config when running inside Kubernetes
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("creating in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes clientset: %w", err)
	}

	// Build ignored namespaces lookup map for O(1) checks
	ignored := make(map[string]bool)
	for _, ns := range ignoredNamespaces {
		ignored[ns] = true
	}

	logger.Info("discovery client initialized",
		"ignored_namespaces", ignoredNamespaces,
	)

	return &Discovery{
		client:            clientset,
		logger:            logger,
		ignoredNamespaces: ignored,
	}, nil
}

// Watch starts watching deployment events and calls onChange when state changes.
// This function blocks until the context is cancelled.
func (d *Discovery) Watch(ctx context.Context, onChange OnStateChange) error {
	d.logger.Info("starting deployment watcher")

	// Initial state sync
	state, err := d.buildWorldState(ctx)
	if err != nil {
		return fmt.Errorf("building initial state: %w", err)
	}

	if err := onChange(state); err != nil {
		d.logger.Warn("initial state callback failed", "error", err)
	}

	// Watch for deployment changes across all namespaces
	watcher, err := d.client.AppsV1().Deployments("").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("creating deployment watcher: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("watcher context cancelled")
			return nil

		case event, ok := <-watcher.ResultChan():
			if !ok {
				d.logger.Warn("watcher channel closed, restarting")
				// Restart the watcher
				watcher, err = d.client.AppsV1().Deployments("").Watch(ctx, metav1.ListOptions{})
				if err != nil {
					return fmt.Errorf("restarting deployment watcher: %w", err)
				}
				continue
			}

			d.logger.Debug("deployment event received", "type", event.Type)

			// Rebuild the complete world state on any change
			state, err := d.buildWorldState(ctx)
			if err != nil {
				d.logger.Error("failed to build world state", "error", err)
				continue
			}

			if err := onChange(state); err != nil {
				d.logger.Warn("state change callback failed", "error", err)
			}
		}
	}
}

// buildWorldState queries all deployments and builds the current world state.
func (d *Discovery) buildWorldState(ctx context.Context) (WorldState, error) {
	deployments, err := d.client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing deployments: %w", err)
	}

	state := make(WorldState)

	for _, deploy := range deployments.Items {
		namespace := deploy.Namespace

		// Skip ignored namespaces
		if d.ignoredNamespaces[namespace] {
			continue
		}

		// Extract version from container image tag
		version := extractVersion(deploy.Spec.Template.Spec.Containers)

		app := AppInfo{
			Name:    deploy.Name,
			Version: version,
		}

		state[namespace] = append(state[namespace], app)
	}

	return state, nil
}

// extractVersion gets the image tag from the first container.
// Returns the full image reference if no tag separator is found.
func extractVersion(containers []corev1.Container) string {
	if len(containers) == 0 {
		return "unknown"
	}

	image := containers[0].Image

	// Try to extract the tag after the last colon
	// Handle both simple tags (nginx:1.19) and digests (nginx@sha256:...)
	if idx := strings.LastIndex(image, ":"); idx != -1 {
		// Check if this is after a potential port number in the registry
		possibleTag := image[idx+1:]
		if !strings.Contains(possibleTag, "/") {
			return possibleTag
		}
	}

	// Check for digest reference
	if idx := strings.LastIndex(image, "@"); idx != -1 {
		return image[idx+1:]
	}

	// Return full image if no tag/digest found
	return image
}
