package controllers

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"

	gitifold "hyperspike.io/eng/gitifold/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"code.gitea.io/sdk/gitea"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func droneLabelNames(component string, cr *gitifold.VCS) (string, map[string]string) {
	labels := map[string]string{
		"app.kubernetes.io/name":       "drone",
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/deployment": "gitifold",
		"app.kubernetes.io/instance":   cr.Name,
	}

	name := strings.Join([]string{cr.Name, component, "gitifold", "drone"}, "-")

	return name, labels
}

func createDroneService(oauthApp *gitea.Oauth2, cr *gitifold.VCS, r *VCSReconciler) error {
	logger := r.Log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)

	droneService := newDroneServiceCr(cr)
	if err := controllerutil.SetControllerReference(cr, droneService, r.Scheme); err != nil {
		return err
	}
	foundService := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: droneService.Name, Namespace: droneService.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Drone Service")
		err = r.Client.Create(context.TODO(), droneService)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Drone Service already exists")
	}

	droneIngress := newDroneIngressCr(cr)
	if err = controllerutil.SetControllerReference(cr, droneIngress, r.Scheme); err != nil {
		return err
	}
	foundIngress := &netv1.Ingress{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: droneIngress.Name, Namespace: droneIngress.Namespace}, foundIngress)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Drone Ingress")
		err = r.Client.Create(context.TODO(), droneIngress)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Drone Ingress already exists")
	}

	droneSecret := newDroneSecretCr(oauthApp, cr)
	if err = controllerutil.SetControllerReference(cr, droneSecret, r.Scheme); err != nil {
		return err
	}
	foundSecret := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: droneSecret.Name, Namespace: droneSecret.Namespace}, foundSecret)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Drone Secret")
		err = r.Client.Create(context.TODO(), droneSecret)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Drone Secret already exists")
	}

	droneDeployment := newDroneDeploymentCr(cr)
	if err = controllerutil.SetControllerReference(cr, droneDeployment, r.Scheme); err != nil {
		return err
	}
	foundDeployment := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: droneDeployment.Name, Namespace: droneDeployment.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Drone Deployment")
		err = r.Client.Create(context.TODO(), droneDeployment)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Drone Deployment already exists")
	}

	droneRunnerService := newDroneRunnerServiceCr(cr)
	if err = controllerutil.SetControllerReference(cr, droneRunnerService, r.Scheme); err != nil {
		return err
	}
	foundRunnerService := &corev1.Service{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: droneRunnerService.Name, Namespace: droneRunnerService.Namespace}, foundRunnerService)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Drone Runner Service")
		err = r.Client.Create(context.TODO(), droneRunnerService)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Drone Runner Service already exists")
	}

	droneRunnerServiceAccount := newDroneRunnerServiceAccountCr(cr)
	if err = controllerutil.SetControllerReference(cr, droneRunnerServiceAccount, r.Scheme); err != nil {
		return err
	}
	foundRunnerServiceAccount := &corev1.ServiceAccount{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: droneRunnerServiceAccount.Name, Namespace: droneRunnerServiceAccount.Namespace}, foundRunnerServiceAccount)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Drone Runner Service Account")
		err = r.Client.Create(context.TODO(), droneRunnerServiceAccount)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Drone Runner Service Account already exists")
	}

	droneRunnerRole := newDroneRunnerRoleCr(cr)
	if err = controllerutil.SetControllerReference(cr, droneRunnerRole, r.Scheme); err != nil {
		return err
	}
	foundRunnerRole := &rbacv1.Role{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: droneRunnerRole.Name, Namespace: droneRunnerRole.Namespace}, foundRunnerRole)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Drone Runner Role")
		err = r.Client.Create(context.TODO(), droneRunnerRole)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Drone Runner Role already exists")
	}

	droneRunnerRoleBinding := newDroneRunnerRoleBindingCr(cr)
	if err = controllerutil.SetControllerReference(cr, droneRunnerRoleBinding, r.Scheme); err != nil {
		return err
	}
	foundRunnerRoleBinding := &rbacv1.RoleBinding{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: droneRunnerRoleBinding.Name, Namespace: droneRunnerRoleBinding.Namespace}, foundRunnerRoleBinding)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Drone Runner Role Binding")
		err = r.Client.Create(context.TODO(), droneRunnerRoleBinding)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Drone Runner Role Binding already exists")
	}

	droneRunnerDeployment := newDroneRunnerDeploymentCr(cr)
	if err = controllerutil.SetControllerReference(cr, droneRunnerDeployment, r.Scheme); err != nil {
		return err
	}
	foundRunnerDeployment := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: droneRunnerDeployment.Name, Namespace: droneRunnerDeployment.Namespace}, foundRunnerDeployment)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Drone Runner Deployment")
		err = r.Client.Create(context.TODO(), droneRunnerDeployment)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Drone Runner Deployment already exists")
	}

	return nil
}

func newDroneServiceCr(cr *gitifold.VCS) *corev1.Service {
	name, labels := droneLabelNames("app", cr)

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cr.Namespace,
			Annotations: make(map[string]string),
			Labels:      labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     "ClusterIP",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromString("http"),
				},
			},
		},
	}
}

func newDroneIngressCr(cr *gitifold.VCS) *netv1.Ingress {
	name, labels := droneLabelNames("app", cr)

	annotations := make(map[string]string)
	for key, value := range cr.Spec.CI.Annotations {
		annotations[key] = value
	}
	return &netv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cr.Namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{
				{
					Host: cr.Spec.CI.Hostname,
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Backend: netv1.IngressBackend{
										ServiceName: name,
										ServicePort: intstr.FromInt(80),
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
						cr.Spec.CI.Hostname,
					},
					SecretName: strings.Join([]string{cr.Name, "drone", "ingress", "tls"}, "-"),
				},
			},
		},
	}
}

func newDroneSecretCr(oauthApp *gitea.Oauth2, cr *gitifold.VCS) *corev1.Secret {
	name, labels := droneLabelNames("app", cr)
	secret, _ := GenerateRandomASCIIString(16)
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cr.Namespace,
			Annotations: make(map[string]string),
			Labels:      labels,
		},
		Data: map[string][]byte{
			"DRONE_DATABASE_DRIVER":     []byte("postgres"),
			"DRONE_GITEA_CLIENT_ID":     []byte(oauthApp.ClientID),
			"DRONE_GITEA_CLIENT_SECRET": []byte(oauthApp.ClientSecret),
			"DRONE_GITEA_SERVER":        []byte(strings.Join([]string{"https://", cr.Spec.Git.Hostname}, "")),
			"DRONE_RPC_SECRET":          []byte(secret),
			"DRONE_SERVER_HOST":         []byte(cr.Spec.CI.Hostname),
			"DRONE_SERVER_PROTO":        []byte("https"),
			"DRONE_USER_CREATE":         []byte("username:gitea,machine:false,admin:true"),
			"DRONE_RPC_HOST":            []byte(strings.Join([]string{name, ":80"}, "")),
			"DRONE_LOGS_DEBUG":          []byte("true"),
		},
	}
}

func newDroneDeploymentCr(cr *gitifold.VCS) *appsv1.Deployment {
	name, labels := droneLabelNames("app", cr)
	pgName := strings.Join([]string{cr.Name, "drone", "gitifold", "postgres"}, "-")

	rc := int32(1)
	fal := false

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
					AutomountServiceAccountToken: &fal,
					Volumes: []corev1.Volume{
						{
							Name: "storage-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "server",
							Image: "drone/drone:1.6.5",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
									Name:          "http",
									Protocol:      "TCP",
								},
								{
									ContainerPort: 9000,
									Name:          "grpc",
									Protocol:      "TCP",
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.IntOrString{Type: intstr.String, StrVal: "http"},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "storage-volume",
									MountPath: "/data",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "POSTGRES_DB",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: pgName,
											},
											Key: "db_name",
										},
									},
								},
								{
									Name: "POSTGRES_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: pgName,
											},
											Key: "db_user",
										},
									},
								},
								{
									Name: "POSTGRES_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: pgName,
											},
											Key: "db_pass",
										},
									},
								},
								{
									Name:  "POSTGRES_HOST",
									Value: pgName,
								},
								{
									Name:  "DRONE_DATABASE_DATASOURCE",
									Value: "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):5432/$(POSTGRES_DB)?sslmode=disable",
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: name,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func newDroneRunnerServiceAccountCr(cr *gitifold.VCS) *corev1.ServiceAccount {
	name, labels := droneLabelNames("runner", cr)
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cr.Namespace,
			Annotations: make(map[string]string),
			Labels:      labels,
		},
	}
}

func newDroneRunnerRoleCr(cr *gitifold.VCS) *rbacv1.Role {
	name, labels := droneLabelNames("runner", cr)
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cr.Namespace,
			Annotations: make(map[string]string),
			Labels:      labels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"secrets",
				},
				Verbs: []string{
					"create",
					"delete",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods",
					"pods/log",
				},
				Verbs: []string{
					"get",
					"create",
					"delete",
					"list",
					"watch",
					"update",
				},
			},
		},
	}
}

func newDroneRunnerRoleBindingCr(cr *gitifold.VCS) *rbacv1.RoleBinding {
	name, labels := droneLabelNames("runner", cr)

	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cr.Namespace,
			Annotations: make(map[string]string),
			Labels:      labels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: cr.Namespace,
			},
		},
	}
}

func newDroneRunnerServiceCr(cr *gitifold.VCS) *corev1.Service {
	name, labels := droneLabelNames("runner", cr)

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cr.Namespace,
			Annotations: make(map[string]string),
			Labels:      labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     "ClusterIP",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   "TCP",
					Port:       8080,
					TargetPort: intstr.FromString("http"),
				},
			},
		},
	}
}

func newDroneRunnerDeploymentCr(cr *gitifold.VCS) *appsv1.Deployment {
	name, labels := droneLabelNames("runner", cr)
	appName, labels := droneLabelNames("app", cr)

	rc := int32(1)

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
					ServiceAccountName: name,
					Containers: []corev1.Container{
						{
							Name:  "server",
							Image: "drone/drone-runner-kube:1.0.0-beta.1",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 3000,
									Name:          "http",
									Protocol:      "TCP",
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: appName,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
