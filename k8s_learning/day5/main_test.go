package main

import (
	"bytes"
	"context"
	"os"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func int32Ptr(i int32) *int32 { return &i }

func TestListPods(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "default"},
			Spec:       corev1.PodSpec{NodeName: "node-1"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-2", Namespace: "default"},
			Spec:       corev1.PodSpec{NodeName: "node-2"},
			Status:     corev1.PodStatus{Phase: corev1.PodPending},
		},
	)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listPods(clientset, "default")

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("listPods returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("pod-1")) {
		t.Errorf("expected pod-1 in output, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("pod-2")) {
		t.Errorf("expected pod-2 in output, got: %s", output)
	}
}

func TestListDeployments(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "go-app", Namespace: "default"},
			Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(3)},
			Status:     appsv1.DeploymentStatus{ReadyReplicas: 3},
		},
	)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listDeployments(clientset, "default")

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("listDeployments returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("go-app")) {
		t.Errorf("expected go-app in output, got: %s", output)
	}
}

func TestScaleDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "go-app", Namespace: "default"},
			Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(3)},
		},
	)

	err := scaleDeployment(clientset, "default", "go-app", 5)
	if err != nil {
		t.Fatalf("scaleDeployment returned error: %v", err)
	}

	deploy, _ := clientset.AppsV1().Deployments("default").Get(context.TODO(), "go-app", metav1.GetOptions{})
	if *deploy.Spec.Replicas != 5 {
		t.Errorf("expected 5 replicas, got %d", *deploy.Spec.Replicas)
	}
}

func TestRestartDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "go-app", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(3),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "go-app"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "go-app", Image: "myapp:v1"}},
					},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "go-app"},
				},
			},
		},
	)

	err := restartDeployment(clientset, "default", "go-app")
	if err != nil {
		t.Fatalf("restartDeployment returned error: %v", err)
	}

	deploy, _ := clientset.AppsV1().Deployments("default").Get(context.TODO(), "go-app", metav1.GetOptions{})
	annotations := deploy.Spec.Template.ObjectMeta.Annotations
	if _, ok := annotations["kubectl.kubernetes.io/restartedAt"]; !ok {
		t.Error("expected restartedAt annotation to be set")
	}
}
