package controllers

import (
	"context"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	"strings"

	gitifold "hyperspike.io/eng/gitifold/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func getRegistryNames(cr *gitifold.VCS) (string, map[string]string) {
	labels := map[string]string{
		"app":        "registry",
		"component":  "containers",
		"deployment": "gitifold",
		"instance":   cr.Name,
	}
	name := strings.Join([]string{cr.Name, "-gitifold-registry"}, "")

	return name, labels
}

func createRegistryService(cr *gitifold.VCS, r *VCSReconciler) error {
	logger := r.Log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)

	service := newRegistryServiceCr(cr)
	if err := controllerutil.SetControllerReference(cr, service, r.Scheme); err != nil {
		return err
	}
	foundService := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Registry Service")
		err = r.Client.Create(context.TODO(), service)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Registry service already exists")
	}

	pvc := newRegistryPVCCr(cr)
	if err := controllerutil.SetControllerReference(cr, pvc, r.Scheme); err != nil {
		return err
	}
	foundPVC := &corev1.PersistentVolumeClaim{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, foundPVC)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Registry PVC", "PVC.Namespace")
		err = r.Client.Create(context.TODO(), pvc)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Registry PVC already exists")
	}

	deployment := newRegistryDeploymentCr(cr)
	if err = controllerutil.SetControllerReference(cr, deployment, r.Scheme); err != nil {
		return err
	}
	foundDeployment := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Registry Deployment")
		err = r.Client.Create(context.TODO(), deployment)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Registry Deployment already exists")
	}

	ingress := newRegistryIngressCr(cr)
	if err = controllerutil.SetControllerReference(cr, ingress, r.Scheme); err != nil {
		return err
	}
	foundIngress := &netv1.Ingress{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: ingress.Name, Namespace: ingress.Namespace}, foundIngress)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Registry Ingress")
		err = r.Client.Create(context.TODO(), ingress)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Registry Ingress already exists")
	}

	return nil
}

func newRegistryServiceCr(cr *gitifold.VCS) *corev1.Service {
	name, labels := getRegistryNames(cr)
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1", // corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cr.Namespace,
			Annotations: make(map[string]string),
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     "ClusterIP",
			Ports: []corev1.ServicePort{
				{
					Name:       "registry",
					Protocol:   "TCP",
					Port:       5000,
					TargetPort: intstr.FromInt(5000),
				},
			},
		},
	}
}
func newRegistryIngressCr(cr *gitifold.VCS) *netv1.Ingress {
	name, labels := getRegistryNames(cr)

	return &netv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{
				{
					Host: cr.Spec.Registry.Hostname,
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Backend: netv1.IngressBackend{
										ServiceName: name,
										ServicePort: intstr.FromInt(5000),
									},
									Path: "/",
								},
							},
						},
					},
				},
			},
			TLS: []netv1.IngressTLS{
				{
					Hosts: []string{
						cr.Spec.Registry.Hostname,
					},
					SecretName: "registry-ingress-tls",
				},
			},
		},
	}
}

func newRegistryPVCCr(cr *gitifold.VCS) *corev1.PersistentVolumeClaim {
	name, labels := getRegistryNames(cr)

	size, _ := resource.ParseQuantity("1Gi")
	//storageClass := "test"
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": size,
				},
			},
			// StorageClassName: &storageClass,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
		},
	}
}
func newRegistryDeploymentCr(cr *gitifold.VCS) *appsv1.Deployment {
	name, labels := getRegistryNames(cr)

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
					Volumes: []corev1.Volume{
						{
							Name: "registry",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: name,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "registry",
							Image: "registry:2.7.1",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "registry",
									MountPath: "/var/lib/registry",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "registry",
									ContainerPort: 5000,
									Protocol:      "TCP",
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(5000),
									},
								},
								InitialDelaySeconds: int32(8),
								PeriodSeconds:       int32(6),
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(5000),
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
