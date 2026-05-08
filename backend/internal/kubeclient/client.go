package kubeclient

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	kube              kubernetes.Interface
	dockerRegistry   string
	dockerUsername    string
	dockerPassword    string
}

type DockerCredentials struct {
	Registry   string
	Username   string
	Password   string
}

func New(kube kubernetes.Interface) *Client {
	return &Client{kube: kube}
}

func NewWithDocker(kube kubernetes.Interface, registry, username, password string) *Client {
	return &Client{kube: kube, dockerRegistry: registry, dockerUsername: username, dockerPassword: password}
}

func (c *Client) ensureImagePullSecret(ctx context.Context, namespace, secretName string) error {
	if c.dockerUsername == "" || c.dockerPassword == "" {
		return nil
	}

	_, err := c.kube.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	registry := c.dockerRegistry
	if registry == "" {
		registry = "docker.io"
	}

	data := map[string][]byte{
		".dockerconfigjson": []byte(fmt.Sprintf(
			`{"auths":{"%s":{"username":"%s","password":"%s","email":"","auth":"%s"}}}`,
			registry,
			c.dockerUsername,
			c.dockerPassword,
			fmt.Sprintf("%s:%s", c.dockerUsername, c.dockerPassword),
		)),
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: data,
	}

	_, err = c.kube.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
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
	if err := c.ensureNamespace(ctx, namespace); err != nil {
		return err
	}
	dep, err := c.kube.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return c.createDeployment(ctx, namespace, name, container, image, appsv1.RollingUpdateDeploymentStrategyType)
		}
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

func (c *Client) UpgradeRecreate(ctx context.Context, namespace, name, container, image string) error {
	if c == nil || c.kube == nil {
		return fmt.Errorf("kubernetes client is not configured")
	}
	if err := c.ensureNamespace(ctx, namespace); err != nil {
		return err
	}
	dep, err := c.kube.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return c.createDeployment(ctx, namespace, name, container, image, appsv1.RecreateDeploymentStrategyType)
		}
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
	dep.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
	_, err = c.kube.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
	return err
}

func (c *Client) ApplyStablePatch(ctx context.Context, namespace, name, container, image string) error {
	return c.UpgradeRolling(ctx, namespace, name, container, image)
}

func (c *Client) ApplyCanaryPatch(ctx context.Context, namespace, name, canaryName, container, image string, replicas int32) error {
	if c == nil || c.kube == nil {
		return fmt.Errorf("kubernetes client is not configured")
	}
	if err := c.ensureNamespace(ctx, namespace); err != nil {
		return err
	}
	stable, err := c.kube.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	canary, err := c.kube.AppsV1().Deployments(namespace).Get(ctx, canaryName, metav1.GetOptions{})
	if err == nil {
		if len(canary.Spec.Template.Spec.Containers) == 0 {
			return fmt.Errorf("canary deployment has no containers")
		}
		updated := false
		for i := range canary.Spec.Template.Spec.Containers {
			if canary.Spec.Template.Spec.Containers[i].Name != container {
				continue
			}
			canary.Spec.Template.Spec.Containers[i].Image = image
			updated = true
			break
		}
		if !updated {
			return fmt.Errorf("container %q not found", container)
		}
		canary.Spec.Replicas = &replicas
		_, err = c.kube.AppsV1().Deployments(namespace).Update(ctx, canary, metav1.UpdateOptions{})
		return err
	}

	newDep := stable.DeepCopy()
	newDep.ResourceVersion = ""
	newDep.UID = ""
	newDep.Name = canaryName
	if newDep.Labels == nil {
		newDep.Labels = map[string]string{}
	}
	newDep.Labels["track"] = "canary"
	if newDep.Spec.Template.Labels == nil {
		newDep.Spec.Template.Labels = map[string]string{}
	}
	newDep.Spec.Template.Labels["track"] = "canary"
	newDep.Spec.Replicas = &replicas
	if len(newDep.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("stable deployment has no containers")
	}
	updated := false
	for i := range newDep.Spec.Template.Spec.Containers {
		if newDep.Spec.Template.Spec.Containers[i].Name != container {
			continue
		}
		newDep.Spec.Template.Spec.Containers[i].Image = image
		updated = true
		break
	}
	if !updated {
		return fmt.Errorf("container %q not found", container)
	}
	_, err = c.kube.AppsV1().Deployments(namespace).Create(ctx, newDep, metav1.CreateOptions{})
	return err
}

func (c *Client) RollbackToTag(ctx context.Context, namespace, name, container, image string) error {
	return c.UpgradeRolling(ctx, namespace, name, container, image)
}

func (c *Client) ensureNamespace(ctx context.Context, namespace string) error {
	_, err := c.kube.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	_, err = c.kube.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}, metav1.CreateOptions{})
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

func (c *Client) createDeployment(ctx context.Context, namespace, name, container, image string, strategy appsv1.DeploymentStrategyType) error {
	replicas := int32(1)

	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:  container,
			Image: image,
			Ports: []corev1.ContainerPort{{ContainerPort: 80}},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{Path: "/", Port: intstr.FromInt(80)},
				},
			},
		}},
	}

	if c.dockerUsername != "" {
		if err := c.ensureImagePullSecret(ctx, namespace, "dockerhub-creds"); err != nil {
			return fmt.Errorf("ensure image pull secret: %w", err)
		}
		podSpec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: "dockerhub-creds"}}
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": name},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Strategy: appsv1.DeploymentStrategy{Type: strategy},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: podSpec,
			},
		},
	}
	_, err := c.kube.AppsV1().Deployments(namespace).Create(ctx, dep, metav1.CreateOptions{})
	return err
}
