package controllers

import (
	"context"
	"crypto/rand"
	"math/big"
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

type DBSecret struct {
	Host string
	Name string
	Pass string
	User string
}

func pgLabelNames(component string, cr *gitifold.VCS) (string, map[string]string) {
	labels := map[string]string{
		"app.kubernetes.io/name":       "postgres",
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/deployment": "gitifold",
		"app.kubernetes.io/instance":   cr.Name,
	}

	name := strings.Join([]string{cr.Name, component, "gitifold", "postgres"}, "-")

	return name, labels
}

func createPgService(component string, cr *gitifold.VCS, r *VCSReconciler) (*DBSecret, error) {
	logger := r.Log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)

	svc := newPgServiceCr(component, cr)
	if err := controllerutil.SetControllerReference(cr, svc, r.Scheme); err != nil {
		return nil, err
	}
	found := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Postgres Service")
		err = r.Client.Create(context.TODO(), svc)
		if err != nil {
			return nil, err
		}
	} else {
		logger.Info("Skip reconcile: Postgres service already exists")
	}
	svc = newPgServiceHeadlessCr(component, cr)
	if err := controllerutil.SetControllerReference(cr, svc, r.Scheme); err != nil {
		return nil, err
	}
	found = &corev1.Service{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Postgres Headless Service")
		err = r.Client.Create(context.TODO(), svc)
		if err != nil {
			return nil, err
		}
	} else {
		logger.Info("Skip reconcile: Postgres Headless service already exists")
	}

	secret, dbSecrets := newPgSecretCr(component, cr)
	if err := controllerutil.SetControllerReference(cr, svc, r.Scheme); err != nil {
		return nil, err
	}
	foundSc := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, foundSc)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Postgres Secret")
		err = r.Client.Create(context.TODO(), secret)
		if err != nil {
			return nil, err
		}
	} else {
		logger.Info("Skip reconcile: Postgres Headless service already exists")
	}

	sts := newPgStatefulSetCr(component, cr)
	if err = controllerutil.SetControllerReference(cr, sts, r.Scheme); err != nil {
		return nil, err
	}
	foundSts := &appsv1.StatefulSet{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}, foundSts)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Postgres StatefulSet")
		err = r.Client.Create(context.TODO(), sts)
		if err != nil {
			return nil, err
		}
	} else {
		logger.Info("Skip reconcile: Postgres StatefulSet already exists")
	}

	return dbSecrets, nil
}

func newPgServiceCr(component string, cr *gitifold.VCS) *corev1.Service {

	name, labels := pgLabelNames(component, cr)

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
					Name:       "postgres",
					Protocol:   "TCP",
					Port:       5432,
					TargetPort: intstr.FromString("postgres"),
				},
			},
		},
	}
}

func newPgServiceHeadlessCr(component string, cr *gitifold.VCS) *corev1.Service {

	name, labels := pgLabelNames(component, cr)
	name = strings.Join([]string{name, "headless"}, "-")

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
			Selector:  labels,
			Type:      "ClusterIP",
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "postgres",
					Protocol:   "TCP",
					Port:       9817,
					TargetPort: intstr.FromString("metrics"),
				},
			},
		},
	}
}
func GenerateRandomASCIIString(length int) (string, error) {
	result := ""
	for {
		if len(result) >= length {
			return result, nil
		}
		num, err := rand.Int(rand.Reader, big.NewInt(int64(127)))
		if err != nil {
			return "", err
		}
		n := num.Int64()
		// Make sure that the number/byte/letter is inside
		// the range of printable ASCII characters (excluding space and DEL)
		if (n >= 48 && n <= 57) || (n >= 65 && n <= 90) || (n >= 97 && n <= 122) {
			result += string(n)
		}
	}
}

func newPgSecretCr(component string, cr *gitifold.VCS) (*corev1.Secret, *DBSecret) {

	pass, _ := GenerateRandomASCIIString(32)
	name, labels := pgLabelNames(component, cr)
	return &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   cr.Namespace,
				Annotations: make(map[string]string),
				Labels:      labels,
			},
			Data: map[string][]byte{
				"db_user": []byte(component),
				"db_name": []byte(component),
				"db_pass": []byte(pass),
				"db_host": []byte(name),
			},
		}, &DBSecret{
			Name: component,
			User: component,
			Pass: pass,
			Host: name,
		}
}

func newPgStatefulSetCr(component string, cr *gitifold.VCS) *appsv1.StatefulSet {

	name, labels := pgLabelNames(component, cr)

	limitCpu, _ := resource.ParseQuantity("1000m")
	limitMemory, _ := resource.ParseQuantity("2048Mi")
	requestCpu, _ := resource.ParseQuantity("100m")
	requestMemory, _ := resource.ParseQuantity("384Mi")

	rc := int32(1)
	gracePeriod := int64(90)
	size, _ := resource.ParseQuantity("2Gi")

	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: make(map[string]string),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &rc,
			ServiceName: name,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
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
						//StorageClassName: &storageClass,
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
					},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:                 "Always",
					TerminationGracePeriodSeconds: &gracePeriod,
					Volumes: []corev1.Volume{
						{
							Name: "runner",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "exporter",
							Env: []corev1.EnvVar{
								{
									Name: "POSTGRES_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: name,
											},
											Key: "db_user",
										},
									},
								},
								{
									Name:  "DATA_SOURCE_NAME",
									Value: "user=$(POSTGRES_USER) host=/run/postgresql/ sslmode=disable",
								},
							},
							Image: "wrouesnel/postgres_exporter:v0.8.0",
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.IntOrString{Type: intstr.String, StrVal: "metrics"},
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.IntOrString{Type: intstr.String, StrVal: "metrics"},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9187,
									Protocol:      "TCP",
									Name:          "metrics",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "runner",
									MountPath: "/run",
								},
							},
						},
						{
							Name:  "postgres",
							Image: "postgres:12.2-alpine",
							Env: []corev1.EnvVar{
								{
									Name:  "PGDATA",
									Value: "/var/lib/postgresql/data",
								},
								{
									Name: "POSTGRES_DB",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: name,
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
												Name: name,
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
												Name: name,
											},
											Key: "db_pass",
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5432,
									Protocol:      "TCP",
									Name:          "postgres",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "runner",
									MountPath: "/run",
								},
								{
									Name:      name,
									MountPath: "/var/lib/postgresql",
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"sh",
											"-i",
											"-c",
											"psql -U $POSTGRES_USER -q  -c 'SELECT 1'",
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
											"-i",
											"-c",
											"psql -U $POSTGRES_USER -q  -c 'SELECT 1'",
										},
									},
								},
								InitialDelaySeconds: int32(4),
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
