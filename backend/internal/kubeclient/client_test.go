package kubeclient

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetDeploymentStatus(t *testing.T) {
	ns := "acme-prod"
	name := "user-service"
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  ns,
			Name:       name,
			Generation: 7,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            4,
			UpdatedReplicas:     3,
			ReadyReplicas:       3,
			AvailableReplicas:   3,
			UnavailableReplicas: 1,
			ObservedGeneration:  6,
		},
	})

	c := New(clientset)
	got, err := c.GetDeploymentStatus(context.Background(), ns, name)
	if err != nil {
		t.Fatalf("GetDeploymentStatus() err = %v", err)
	}
	if got.Namespace != ns || got.Name != name {
		t.Fatalf("unexpected deployment identity: %+v", got)
	}
	if got.Replicas != 4 || got.ReadyReplicas != 3 {
		t.Fatalf("unexpected deployment status: %+v", got)
	}
}

func TestGetDeploymentStatus_NotFound(t *testing.T) {
	c := New(fake.NewSimpleClientset())
	if _, err := c.GetDeploymentStatus(context.Background(), "acme-prod", "missing"); err == nil {
		t.Fatalf("expected not found error")
	}
}

func TestUpgradeRolling(t *testing.T) {
	ns := "acme-prod"
	name := "user-service"
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "repo/app:v1"},
					},
				},
			},
		},
	})

	c := New(clientset)
	if err := c.UpgradeRolling(context.Background(), ns, name, "app", "repo/app:v2"); err != nil {
		t.Fatalf("UpgradeRolling() err = %v", err)
	}
	got, err := clientset.AppsV1().Deployments(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get deployment err = %v", err)
	}
	if got.Spec.Template.Spec.Containers[0].Image != "repo/app:v2" {
		t.Fatalf("image = %s", got.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestUpgradeRolling_ContainerNotFound(t *testing.T) {
	ns := "acme-prod"
	name := "user-service"
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "repo/app:v1"},
					},
				},
			},
		},
	})
	c := New(clientset)
	if err := c.UpgradeRolling(context.Background(), ns, name, "api", "repo/app:v2"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpgradeRecreate(t *testing.T) {
	ns := "acme-prod"
	name := "user-service"
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "repo/app:v1"},
					},
				},
			},
		},
	})
	c := New(clientset)
	if err := c.UpgradeRecreate(context.Background(), ns, name, "app", "repo/app:v2"); err != nil {
		t.Fatalf("UpgradeRecreate() err = %v", err)
	}
	got, err := clientset.AppsV1().Deployments(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get deployment err = %v", err)
	}
	if got.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType {
		t.Fatalf("strategy = %s", got.Spec.Strategy.Type)
	}
	if got.Spec.Template.Spec.Containers[0].Image != "repo/app:v2" {
		t.Fatalf("image = %s", got.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestApplyStablePatch(t *testing.T) {
	ns := "acme-prod"
	name := "user-service"
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "repo/app:v1"}}},
			},
		},
	})
	c := New(clientset)
	if err := c.ApplyStablePatch(context.Background(), ns, name, "app", "repo/app:v3"); err != nil {
		t.Fatalf("ApplyStablePatch() err = %v", err)
	}
	got, _ := clientset.AppsV1().Deployments(ns).Get(context.Background(), name, metav1.GetOptions{})
	if got.Spec.Template.Spec.Containers[0].Image != "repo/app:v3" {
		t.Fatalf("image = %s", got.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestApplyCanaryPatch_Create(t *testing.T) {
	ns := "acme-prod"
	stable := "user-service"
	canary := "user-service-canary"
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: stable, Labels: map[string]string{"app": "user-service"}},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "user-service"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "repo/app:v1"}}},
			},
		},
	})
	c := New(clientset)
	if err := c.ApplyCanaryPatch(context.Background(), ns, stable, canary, "app", "repo/app:v2", 1); err != nil {
		t.Fatalf("ApplyCanaryPatch() err = %v", err)
	}
	got, err := clientset.AppsV1().Deployments(ns).Get(context.Background(), canary, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get canary err = %v", err)
	}
	if got.Spec.Template.Spec.Containers[0].Image != "repo/app:v2" {
		t.Fatalf("image = %s", got.Spec.Template.Spec.Containers[0].Image)
	}
	if got.Labels["track"] != "canary" {
		t.Fatalf("track label = %q", got.Labels["track"])
	}
}

func TestApplyCanaryPatch_Update(t *testing.T) {
	ns := "acme-prod"
	stable := "user-service"
	canary := "user-service-canary"
	replicas := int32(1)
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: stable},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "repo/app:v1"}}},
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: canary},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "repo/app:v-old"}}},
				},
			},
		},
	)
	c := New(clientset)
	if err := c.ApplyCanaryPatch(context.Background(), ns, stable, canary, "app", "repo/app:v2", 2); err != nil {
		t.Fatalf("ApplyCanaryPatch() err = %v", err)
	}
	got, _ := clientset.AppsV1().Deployments(ns).Get(context.Background(), canary, metav1.GetOptions{})
	if got.Spec.Template.Spec.Containers[0].Image != "repo/app:v2" {
		t.Fatalf("image = %s", got.Spec.Template.Spec.Containers[0].Image)
	}
	if got.Spec.Replicas == nil || *got.Spec.Replicas != 2 {
		t.Fatalf("replicas = %v", got.Spec.Replicas)
	}
}

func TestRollbackToTag(t *testing.T) {
	ns := "acme-prod"
	name := "user-service"
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "repo/app:v2"}}},
			},
		},
	})
	c := New(clientset)
	if err := c.RollbackToTag(context.Background(), ns, name, "app", "repo/app:v1"); err != nil {
		t.Fatalf("RollbackToTag() err = %v", err)
	}
	got, _ := clientset.AppsV1().Deployments(ns).Get(context.Background(), name, metav1.GetOptions{})
	if got.Spec.Template.Spec.Containers[0].Image != "repo/app:v1" {
		t.Fatalf("image = %s", got.Spec.Template.Spec.Containers[0].Image)
	}
}
