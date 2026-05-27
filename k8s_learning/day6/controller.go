package main

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	simpleappv1 "simple-operator/api/v1"
)

type SimpleAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *SimpleAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var app simpleappv1.SimpleApp
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Reconcile Deployment
	deploy := r.desiredDeployment(&app)
	if err := ctrl.SetControllerReference(&app, deploy, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	var foundDeploy appsv1.Deployment
	err := r.Get(ctx, client.ObjectKeyFromObject(deploy), &foundDeploy)
	if errors.IsNotFound(err) {
		logger.Info("Creating Deployment", "name", deploy.Name)
		if err := r.Create(ctx, deploy); err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		if *foundDeploy.Spec.Replicas != app.Spec.Replicas ||
			foundDeploy.Spec.Template.Spec.Containers[0].Image != app.Spec.Image {
			foundDeploy.Spec.Replicas = &app.Spec.Replicas
			foundDeploy.Spec.Template.Spec.Containers[0].Image = app.Spec.Image
			logger.Info("Updating Deployment", "name", deploy.Name)
			if err := r.Update(ctx, &foundDeploy); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Reconcile Service
	svc := r.desiredService(&app)
	if err := ctrl.SetControllerReference(&app, svc, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	var foundSvc corev1.Service
	err = r.Get(ctx, client.ObjectKeyFromObject(svc), &foundSvc)
	if errors.IsNotFound(err) {
		logger.Info("Creating Service", "name", svc.Name)
		if err := r.Create(ctx, svc); err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// Update status
	app.Status.ReadyReplicas = foundDeploy.Status.ReadyReplicas
	if app.Status.ReadyReplicas == app.Spec.Replicas {
		app.Status.Phase = "Running"
	} else {
		app.Status.Phase = "Scaling"
	}
	if err := r.Status().Update(ctx, &app); err != nil {
		logger.Error(err, "Failed to update status")
	}

	return ctrl.Result{}, nil
}

func (r *SimpleAppReconciler) desiredDeployment(app *simpleappv1.SimpleApp) *appsv1.Deployment {
	labels := map[string]string{"simpleapp": app.Name}
	replicas := app.Spec.Replicas
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "app",
						Image: app.Spec.Image,
						Ports: []corev1.ContainerPort{{ContainerPort: app.Spec.Port}},
					}},
				},
			},
		},
	}
}

func (r *SimpleAppReconciler) desiredService(app *simpleappv1.SimpleApp) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-svc", app.Name),
			Namespace: app.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"simpleapp": app.Name},
			Ports: []corev1.ServicePort{{
				Port:       app.Spec.Port,
				TargetPort: intstr.FromInt32(app.Spec.Port),
			}},
		},
	}
}

func (r *SimpleAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&simpleappv1.SimpleApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
