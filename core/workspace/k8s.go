package workspace

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"sync"
	"time"
)

// Compile-time check.
var _ WorkspacePort = (*K8sWorkspace)(nil)

// K8sClient abstracts the Kubernetes API operations needed by K8sWorkspace.
// Implementations can use the real Kubernetes API or a test double.
type K8sClient interface {
	// CreatePod creates a pod and returns when it is ready.
	CreatePod(ctx context.Context, name, image, namespace string, env map[string]string, command []string) error

	// DeletePod removes a pod.
	DeletePod(ctx context.Context, name, namespace string) error

	// PodStatus returns the current phase of a pod ("Running", "Pending", etc.).
	PodStatus(ctx context.Context, name, namespace string) (string, error)

	// ExecInPod runs a command in an existing pod and returns the result.
	ExecInPod(ctx context.Context, name, namespace, container string, cmd []string) (string, string, int, error)

	// PodIP returns the IP address of a running pod.
	PodIP(ctx context.Context, name, namespace string) (string, error)
}

// K8sConfig configures the K8sWorkspace adapter.
type K8sConfig struct {
	// Namespace is the Kubernetes namespace for workspaces.
	Namespace string

	// DefaultImage is the container image to use when WorkspaceSpec.Stack is empty.
	DefaultImage string

	// DefaultImageMap maps WorkspaceSpec.Stack values to container images.
	// If empty, DefaultImage is used for all stacks.
	DefaultImageMap map[string]string

	// PodTimeout is how long to wait for a pod to become Ready.
	PodTimeout time.Duration
}

// DefaultK8sConfig returns a sensible default configuration.
func DefaultK8sConfig() K8sConfig {
	return K8sConfig{
		Namespace:    "devos",
		DefaultImage: "ubuntu:22.04",
		PodTimeout:   60 * time.Second,
	}
}

// K8sOption configures the K8sWorkspace adapter.
type K8sOption func(*K8sConfig)

func WithK8sNamespace(ns string) K8sOption {
	return func(c *K8sConfig) { c.Namespace = ns }
}

func WithK8sDefaultImage(img string) K8sOption {
	return func(c *K8sConfig) { c.DefaultImage = img }
}

func WithK8sImageMap(m map[string]string) K8sOption {
	return func(c *K8sConfig) { c.DefaultImageMap = m }
}

func WithK8sPodTimeout(d time.Duration) K8sOption {
	return func(c *K8sConfig) { c.PodTimeout = d }
}

// K8sWorkspace implements WorkspacePort by provisioning pods in Kubernetes.
// It depends on the K8sClient interface so the Kubernetes API can be
// replaced with a test double.
type K8sWorkspace struct {
	client K8sClient
	config K8sConfig

	mu   sync.Mutex
	pods map[WorkspaceID]Workspace
}

// NewK8sWorkspace creates a new K8sWorkspace with the given client and options.
func NewK8sWorkspace(client K8sClient, opts ...K8sOption) *K8sWorkspace {
	cfg := DefaultK8sConfig()
	for _, fn := range opts {
		fn(&cfg)
	}
	return &K8sWorkspace{
		client: client,
		config: cfg,
		pods:   make(map[WorkspaceID]Workspace),
	}
}

// Provision creates a new workspace as a Kubernetes pod.
func (k *K8sWorkspace) Provision(ctx context.Context, spec WorkspaceSpec) (Workspace, error) {
	id, err := newK8sWorkspaceID()
	if err != nil {
		return Workspace{}, fmt.Errorf("k8s: generate id: %w", err)
	}

	image := k.imageFor(spec.Stack)
	log.Printf("k8s: provisioning pod=%s image=%s stack=%s", id, image, spec.Stack)

	if err := k.client.CreatePod(ctx, string(id), image, k.config.Namespace, spec.Env, nil); err != nil {
		return Workspace{}, fmt.Errorf("k8s: create pod %s: %w", id, err)
	}

	ws := Workspace{
		ID:      id,
		Spec:    spec,
		Status:  WorkspaceStatusReady,
		RootDir: "/workspace",
		Created: time.Now(),
	}

	k.mu.Lock()
	k.pods[id] = ws
	k.mu.Unlock()

	return ws, nil
}

// Exec runs a command inside the workspace pod.
func (k *K8sWorkspace) Exec(ctx context.Context, id WorkspaceID, req ExecRequest) (ExecResponse, error) {
	k.mu.Lock()
	_, ok := k.pods[id]
	k.mu.Unlock()

	if !ok {
		return ExecResponse{}, fmt.Errorf("k8s: %w: id=%s", ErrNotFound, id)
	}

	cmd := []string{"/bin/sh", "-c", req.Command}
	if req.WorkDir != "" {
		cmd = []string{"/bin/sh", "-c", fmt.Sprintf("cd %s && %s", req.WorkDir, req.Command)}
	}

	stdout, stderr, exitCode, err := k.client.ExecInPod(ctx, string(id), k.config.Namespace, "workspace", cmd)
	if err != nil {
		return ExecResponse{
			Stdout:   stdout,
			Stderr:   stderr,
			ExitCode: exitCode,
		}, fmt.Errorf("k8s: exec: %w", err)
	}

	return ExecResponse{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}, nil
}

// Status returns the current status of the workspace pod.
func (k *K8sWorkspace) Status(ctx context.Context, id WorkspaceID) (WorkspaceStatus, error) {
	k.mu.Lock()
	_, ok := k.pods[id]
	k.mu.Unlock()

	if !ok {
		return WorkspaceStatusUnknown, fmt.Errorf("k8s: %w: id=%s", ErrNotFound, id)
	}

	phase, err := k.client.PodStatus(ctx, string(id), k.config.Namespace)
	if err != nil {
		return WorkspaceStatusUnknown, fmt.Errorf("k8s: status: %w", err)
	}

	switch phase {
	case "Running":
		return WorkspaceStatusReady, nil
	case "Pending":
		return WorkspaceStatusProvisioning, nil
	case "Succeeded", "Failed":
		return WorkspaceStatusFailed, nil
	default:
		return WorkspaceStatusUnknown, nil
	}
}

// Recycle deletes the workspace pod.
func (k *K8sWorkspace) Recycle(ctx context.Context, id WorkspaceID) error {
	k.mu.Lock()
	_, ok := k.pods[id]
	if !ok {
		k.mu.Unlock()
		return fmt.Errorf("k8s: %w: id=%s", ErrNotFound, id)
	}
	delete(k.pods, id)
	k.mu.Unlock()

	log.Printf("k8s: recycling pod=%s", id)
	return k.client.DeletePod(ctx, string(id), k.config.Namespace)
}

func (k *K8sWorkspace) imageFor(stack string) string {
	if stack == "" {
		return k.config.DefaultImage
	}
	if k.config.DefaultImageMap != nil {
		if img, ok := k.config.DefaultImageMap[stack]; ok {
			return img
		}
	}
	return k.config.DefaultImage
}

func newK8sWorkspaceID() (WorkspaceID, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return WorkspaceID(fmt.Sprintf("k8s-%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])), nil
}
