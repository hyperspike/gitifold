package controllers

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	gitifold "hyperspike.io/eng/gitifold/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func keydbLabelNames(component string, cr *gitifold.VCS) (string, map[string]string) {
	labels := map[string]string{
		"app.kubernetes.io/name":       "keydb",
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/deployment": "gitifold",
		"app.kubernetes.io/instance":   cr.Name,
	}

	name := strings.Join([]string{cr.Name, component, "gitifold", "keydb"}, "-")

	return name, labels
}

func createKeyDBService(component string, cr *gitifold.VCS, r *VCSReconciler) error {
	logger := r.Log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	svc := newKeyDBServiceCr(component, cr)
	if err := controllerutil.SetControllerReference(cr, svc, r.Scheme); err != nil {
		return err
	}
	foundSvc := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, foundSvc)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new KeyDB Service")
		err = r.Client.Create(context.TODO(), svc)
		if err != nil {
			return err
		}
	}
	logger.Info("Skip reconcile: KeyDB service already exists")

	sts := newKeyDBDeploymentCr(component, cr)
	if err := controllerutil.SetControllerReference(cr, sts, r.Scheme); err != nil {
		return err
	}
	found := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Redis Deployment")
		err = r.Client.Create(context.TODO(), sts)
		if err != nil {
			return err
		}
	}
	logger.Info("Skip reconcile: KeyDB Deployment already exists")

	return nil
}

func newKeyDBServiceCr(component string, cr *gitifold.VCS) *corev1.Service {

	name, labels := keydbLabelNames(component, cr)

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: make(map[string]string),
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     "ClusterIP",
			Ports: []corev1.ServicePort{
				{
					Name:       "redis",
					Protocol:   "TCP",
					Port:       6379,
					TargetPort: intstr.FromString("redis"),
				},
			},
		},
	}
}

func newKeyDBDeploymentCr(component string, cr *gitifold.VCS) *appsv1.Deployment {

	name, labels := keydbLabelNames(component, cr)

	limitCpu, _ := resource.ParseQuantity("250m")
	limitMemory, _ := resource.ParseQuantity("1024Mi")
	requestCpu, _ := resource.ParseQuantity("10m")
	requestMemory, _ := resource.ParseQuantity("15Mi")

	var rc int32
	rc = 1
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &rc,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "keydb",
							Image:           "eqalpha/keydb:latest",
							ImagePullPolicy: "Always",
							Ports: []corev1.ContainerPort{
								{
									Name:          "redis",
									ContainerPort: 6379,
									Protocol:      "TCP",
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"sh",
											"-c",
											"keydb-cli -h $(hostname) ping",
										},
									},
								},
								InitialDelaySeconds: int32(8),
								PeriodSeconds:       int32(6),
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"sh",
											"-c",
											"keydb-cli -h $(hostname) ping",
										},
									},
								},
								InitialDelaySeconds: int32(5),
								PeriodSeconds:       int32(6),
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    limitCpu,
									"memory": limitMemory,
								},
								Requests: corev1.ResourceList{
									"cpu":    requestCpu,
									"memory": requestMemory,
								},
							},
						},
					},
				},
			},
		},
	}
}
