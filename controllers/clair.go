package controllers

// service

// ingress

// secret

// deployment

func newClairDeploymentCr(cr *gitifold.VCS) *appsv1.Deployment {
	name, labels := clairLabelNames("app", cr)
	pgName := strings.Join([]string{cr.Name, "clair", "gitifold", "postgres"}, "-")

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
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: name,
									Items: []corev1.KeyToPath{
										{
											Key:  "config.yaml",
											Path: "config.yaml",
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "server",
							Image: "coreos/clair:v2.12",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 6060,
									Name:          "api",
									Protocol:      "TCP",
								},
								{
									ContainerPort: 6061,
									Name:          "metrics",
									Protocol:      "TCP",
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.IntOrString{Type: intstr.String, StrVal: "metrics"},
									},
								},
							},
							RedinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.IntOrString{Type: intstr.String, StrVal: "metrics"},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/clair",
								},
							},
						},
					},
				},
			},
		},
	}
}
