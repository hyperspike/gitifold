package controllers

import (
	"bytes"
	"context"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/util/intstr"

	gitifold "hyperspike.io/eng/gitifold/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func clairLabelNames(cr *gitifold.VCS) (string, map[string]string) {
	labels := map[string]string{
		"app.kubernetes.io/name":       "clair",
		"app.kubernetes.io/component":  "security",
		"app.kubernetes.io/deployment": "gitifold",
		"app.kubernetes.io/instance":   cr.Name,
	}

	name := strings.Join([]string{cr.Name, "security", "gitifold", "clair"}, "-")

	return name, labels
}

func createClairService(dbConfig *DBSecret, cr *gitifold.VCS, r *VCSReconciler) error {
	logger := r.Log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)

	clairService := newClairServiceCr(cr)
	if err := controllerutil.SetControllerReference(cr, clairService, r.Scheme); err != nil {
		return err
	}
	foundService := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: clairService.Name, Namespace: clairService.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Clair Service")
		err = r.Client.Create(context.TODO(), clairService)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Clair Service already exists")
	}

	clairIngress := newClairIngressCr(cr)
	if err = controllerutil.SetControllerReference(cr, clairIngress, r.Scheme); err != nil {
		return err
	}
	foundIngress := &netv1.Ingress{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: clairIngress.Name, Namespace: clairIngress.Namespace}, foundIngress)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Clair Ingress")
		err = r.Client.Create(context.TODO(), clairIngress)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Clair Ingress already exists")
	}

	clairSecret, err := newClairSecretCr(dbConfig, cr)
	if err = controllerutil.SetControllerReference(cr, clairSecret, r.Scheme); err != nil {
		return err
	}
	foundSecret := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: clairSecret.Name, Namespace: clairSecret.Namespace}, foundSecret)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Clair Secret")
		err = r.Client.Create(context.TODO(), clairSecret)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Clair Secret already exists")
	}

	clairDeployment := newClairDeploymentCr(cr)
	if err = controllerutil.SetControllerReference(cr, clairDeployment, r.Scheme); err != nil {
		return err
	}
	foundDeployment := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: clairDeployment.Name, Namespace: clairDeployment.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Clair Deployment")
		err = r.Client.Create(context.TODO(), clairDeployment)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Clair Deployment already exists")
	}

	return nil
}

func newClairServiceCr(cr *gitifold.VCS) *corev1.Service {
	name, labels := clairLabelNames(cr)

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
					Name:       "api",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromString("api"),
				},
			},
		},
	}
}

func newClairIngressCr(cr *gitifold.VCS) *netv1.Ingress {
	name, labels := clairLabelNames(cr)

	annotations := make(map[string]string)
	for key, value := range cr.Spec.Clair.Annotations {
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
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{
				{
					Host: cr.Spec.Git.Hostname,
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
						cr.Spec.Git.Hostname,
					},
					SecretName: strings.Join([]string{cr.Name, "clair", "ingress", "tls"}, "-"),
				},
			},
		},
	}
}

type ClairData struct {
	Key     string
	Ingress string
	DB      *DBSecret
}

func newClairSecretCr(dbSecret *DBSecret, cr *gitifold.VCS) (*corev1.Secret, error) {
	name, labels := clairLabelNames(cr)

	secret, err := GenerateRandomBase64String(32)
	if err != nil {
		return nil, err
	}
	data := ClairData{
		Key:     secret,
		DB:      dbSecret,
		Ingress: cr.Spec.Clair.Hostname,
	}

	config := template.New("config")
	config, err = config.Parse(`clair:
  database:
    # Database driver
    type: pgsql
    options:
      # PostgreSQL Connection string
      # https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
      # This should be done using secrets or Vault, but for now this will also work
      source: "postgres://{{ .DB.User -}}:{{ .DB.Pass -}}@{{ .DB.Host -}}:5432/{{ .DB.Name -}}?sslmode=disable"

      # Number of elements kept in the cache
      # Values unlikely to change (e.g. namespaces) are cached in order to save prevent needless roundtrips to the database.
      cachesize: 16384

      # 32-bit URL-safe base64 key used to encrypt pagination tokens
      # If one is not provided, it will be generated.
      # Multiple clair instances in the same cluster need the same value.
      paginationkey: "{{ .Key -}}"
  api:
    # v3 grpc/RESTful API server address
    addr: "0.0.0.0:6060"

    # Health server address
    # This is an unencrypted endpoint useful for load balancers to check to healthiness of the clair server.
    healthaddr: "0.0.0.0:6061"

    # Deadline before an API request will respond with a 503
    timeout: 900s

    # Optional PKI configuration
    # If you want to easily generate client certificates and CAs, try the following projects:
    # https://github.com/coreos/etcd-ca
    # https://github.com/cloudflare/cfssl
    servername:
    cafile:
    keyfile:
    certfile:

  worker:
    namespace_detectors:
    - os-release
    - lsb-release
    - apt-sources
    - alpine-release
    - redhat-release

    feature_listers:
    - apk
    - dpkg
    - rpm

  updater:
    # Frequency the database will be updated with vulnerabilities from the default data sources
    # The value 0 disables the updater entirely.
    interval: "2h"
    enabledupdaters:
    - debian
    - ubuntu
    - rhel
    - oracle
    - alpine

  notifier:
    # Number of attempts before the notification is marked as failed to be sent
    attempts: 3

    # Duration before a failed notification is retried
    renotifyinterval: 2h

    http:
      # Optional endpoint that will receive notifications via POST requests
      endpoint: "https://{{ .Ingress -}}/notify/me"

      # Optional PKI configuration
      # If you want to easily generate client certificates and CAs, try the following projects:
      # https://github.com/cloudflare/cfssl
      # https://github.com/coreos/etcd-ca
      servername:
      cafile:
      keyfile:
      certfile:

      # Optional HTTP Proxy: must be a valid URL (including the scheme).
      proxy:`)

	if err != nil {
		return nil, err
	}
	var str bytes.Buffer
	if err = config.Execute(&str, data); err != nil {
		return nil, err
	}

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
			"config.yaml": str.Bytes(),
		},
	}, nil
}

func newClairDeploymentCr(cr *gitifold.VCS) *appsv1.Deployment {
	name, labels := clairLabelNames(cr)
	// pgName := strings.Join([]string{cr.Name, "clair", "gitifold", "postgres"}, "-")

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
							ReadinessProbe: &corev1.Probe{
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
