package kubeclient

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type DeploymentStatus struct {
	Namespace          string
	Name               string
	Replicas           int32
	UpdatedReplicas    int32
	ReadyReplicas      int32
	AvailableReplicas  int32
	UnavailableReplicas int32
	ObservedGeneration int64
	Generation         int64
}

type Client struct {
	kube kubernetes.Interface
}

func New(kube kubernetes.Interface) *Client {
	return &Client{kube: kube}
}

func (c *Client) GetDeploymentStatus(ctx context.Context, namespace, name string) (DeploymentStatus, error) {
	if c == nil || c.kube == nil {
		return DeploymentStatus{}, fmt.Errorf("kubernetes client is not configured")
	}
	dep, err := c.kube.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return DeploymentStatus{}, err
	}

	return toDeploymentStatus(dep), nil
}

func (c *Client) UpgradeRolling(ctx context.Context, namespace, name, container, image string) error {
	if c == nil || c.kube == nil {
		return fmt.Errorf("kubernetes client is not configured")
	}
	dep, err := c.kube.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if len(dep.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("deployment has no containers")
	}
	updated := false
	for i := range dep.Spec.Template.Spec.Containers {
		if dep.Spec.Template.Spec.Containers[i].Name != container {
			continue
		}
		dep.Spec.Template.Spec.Containers[i].Image = image
		updated = true
		break
	}
	if !updated {
		return fmt.Errorf("container %q not found", container)
	}
	if dep.Spec.Strategy.Type == "" {
		dep.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
	}
	_, err = c.kube.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
	return err
}

func toDeploymentStatus(dep *appsv1.Deployment) DeploymentStatus {
	return DeploymentStatus{
		Namespace:           dep.Namespace,
		Name:                dep.Name,
		Replicas:            dep.Status.Replicas,
		UpdatedReplicas:     dep.Status.UpdatedReplicas,
		ReadyReplicas:       dep.Status.ReadyReplicas,
		AvailableReplicas:   dep.Status.AvailableReplicas,
		UnavailableReplicas: dep.Status.UnavailableReplicas,
		ObservedGeneration:  dep.Status.ObservedGeneration,
		Generation:          dep.Generation,
	}
}
