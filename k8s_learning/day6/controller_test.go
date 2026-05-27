package main

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	simpleappv1 "simple-operator/api/v1"
)

func testScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = simpleappv1.AddToScheme(s)
	return s
}

func TestReconcile_CreatesDeploymentAndService(t *testing.T) {
	s := testScheme()
	app := &simpleappv1.SimpleApp{
		ObjectMeta: metav1.ObjectMeta{Name: "my-web", Namespace: "default"},
		Spec:       simpleappv1.SimpleAppSpec{Image: "nginx:alpine", Replicas: 3, Port: 80},
	}

	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(app).WithStatusSubresource(app).Build()
	r := &SimpleAppReconciler{Client: cl, Scheme: s}

	_, err := r.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-web", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile error: %v", err)
	}

	// Verify Deployment was created
	var deploy appsv1.Deployment
	err = cl.Get(context.TODO(), types.NamespacedName{Name: "my-web", Namespace: "default"}, &deploy)
	if err != nil {
		t.Fatalf("Expected Deployment to be created, got: %v", err)
	}
	if *deploy.Spec.Replicas != 3 {
		t.Errorf("Expected 3 replicas, got %d", *deploy.Spec.Replicas)
	}
	if deploy.Spec.Template.Spec.Containers[0].Image != "nginx:alpine" {
		t.Errorf("Expected image nginx:alpine, got %s", deploy.Spec.Template.Spec.Containers[0].Image)
	}

	// Verify Service was created
	var svc corev1.Service
	err = cl.Get(context.TODO(), types.NamespacedName{Name: "my-web-svc", Namespace: "default"}, &svc)
	if err != nil {
		t.Fatalf("Expected Service to be created, got: %v", err)
	}
	if svc.Spec.Ports[0].Port != 80 {
		t.Errorf("Expected port 80, got %d", svc.Spec.Ports[0].Port)
	}
}

func TestReconcile_UpdatesDeploymentOnSpecChange(t *testing.T) {
	s := testScheme()
	app := &simpleappv1.SimpleApp{
		ObjectMeta: metav1.ObjectMeta{Name: "my-web", Namespace: "default"},
		Spec:       simpleappv1.SimpleAppSpec{Image: "nginx:alpine", Replicas: 3, Port: 80},
	}

	replicas := int32(3)
	existingDeploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "my-web", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"simpleapp": "my-web"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"simpleapp": "my-web"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "nginx:1.24"}},
				},
			},
		},
	}

	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(app, existingDeploy).WithStatusSubresource(app).Build()
	r := &SimpleAppReconciler{Client: cl, Scheme: s}

	_, err := r.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-web", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile error: %v", err)
	}

	// Verify Deployment was updated
	var deploy appsv1.Deployment
	_ = cl.Get(context.TODO(), types.NamespacedName{Name: "my-web", Namespace: "default"}, &deploy)
	if deploy.Spec.Template.Spec.Containers[0].Image != "nginx:alpine" {
		t.Errorf("Expected image to be updated to nginx:alpine, got %s", deploy.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestReconcile_IgnoresNotFound(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	r := &SimpleAppReconciler{Client: cl, Scheme: s}

	_, err := r.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Expected no error for NotFound, got: %v", err)
	}
}
