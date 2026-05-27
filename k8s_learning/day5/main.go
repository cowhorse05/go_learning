package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func getClientset() (*kubernetes.Clientset, error) {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if envKC := os.Getenv("KUBECONFIG"); envKC != "" {
		kubeconfig = envKC
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("build config: %w", err)
	}
	return kubernetes.NewForConfig(config)
}

func listPods(clientset kubernetes.Interface, namespace string) error {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("%-40s %-12s %-25s\n", "NAME", "STATUS", "NODE")
	for _, pod := range pods.Items {
		fmt.Printf("%-40s %-12s %-25s\n", pod.Name, pod.Status.Phase, pod.Spec.NodeName)
	}
	return nil
}

func listDeployments(clientset kubernetes.Interface, namespace string) error {
	deploys, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("%-30s %-10s %-10s\n", "NAME", "READY", "REPLICAS")
	for _, d := range deploys.Items {
		fmt.Printf("%-30s %d/%-8d %-10d\n", d.Name, d.Status.ReadyReplicas, *d.Spec.Replicas, *d.Spec.Replicas)
	}
	return nil
}

func scaleDeployment(clientset kubernetes.Interface, namespace, name string, replicas int32) error {
	deploy, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get deployment: %w", err)
	}
	deploy.Spec.Replicas = &replicas
	_, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deploy, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update deployment: %w", err)
	}
	fmt.Printf("Deployment %s scaled to %d replicas\n", name, replicas)
	return nil
}

func restartDeployment(clientset kubernetes.Interface, namespace, name string) error {
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err := clientset.AppsV1().Deployments(namespace).Patch(
		context.TODO(), name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	fmt.Printf("Deployment %s restarted\n", name)
	return nil
}

func usage() {
	fmt.Println(`Usage:
  k8s-tool pods                    List all pods
  k8s-tool list                    List all deployments
  k8s-tool scale <name> <replicas> Scale a deployment
  k8s-tool restart <name>          Rolling restart a deployment`)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	clientset, err := getClientset()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	namespace := "default"
	cmd := os.Args[1]

	switch cmd {
	case "pods":
		if err := listPods(clientset, namespace); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "list":
		if err := listDeployments(clientset, namespace); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "scale":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: k8s-tool scale <name> <replicas>")
			os.Exit(1)
		}
		replicas, err := strconv.ParseInt(os.Args[3], 10, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid replicas: %v\n", err)
			os.Exit(1)
		}
		if err := scaleDeployment(clientset, namespace, os.Args[2], int32(replicas)); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "restart":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: k8s-tool restart <name>")
			os.Exit(1)
		}
		if err := restartDeployment(clientset, namespace, os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
	}
}
