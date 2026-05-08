package kubeclient

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
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
