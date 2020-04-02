package controllers

import (
	"context"

	"bytes"
	"crypto/rand"
	"encoding/base64"
	"io"
	"math/big"
	"strings"
	"text/template"
	"time"

	"github.com/dgrijalva/jwt-go"

	"k8s.io/apimachinery/pkg/api/resource"
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

func createGiteaService(dbSecret *DBSecret, cr *gitifold.VCS, r *VCSReconciler) error {
	logger := r.Log.WithValues("Request.Namespace", cr.Namespace, "Request.Name", cr.Name)
	cm, err := newGiteaSecret(dbSecret, cr)
	if err != nil {
		return err
	}
	if err = controllerutil.SetControllerReference(cr, cm, r.Scheme); err != nil {
		return err
	}
	foundCM := &corev1.Secret{}

	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, foundCM)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Gitea Secret")
		err = r.Client.Create(context.TODO(), cm)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Gitea Secret already exists")
	}

	svc := newGiteaServiceCr(cr)
	if err := controllerutil.SetControllerReference(cr, svc, r.Scheme); err != nil {
		return err
	}
	found := &corev1.Service{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Gitea Service")
		err = r.Client.Create(context.TODO(), svc)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Gitea service already exists")
	}

	pvc := newGiteaPVCCr(cr)
	if err := controllerutil.SetControllerReference(cr, pvc, r.Scheme); err != nil {
		return err
	}
	foundPVC := &corev1.PersistentVolumeClaim{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, foundPVC)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Gitea PVC")
		err = r.Client.Create(context.TODO(), pvc)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Gitea PVC already exists")
	}

	ing := newGiteaIngressCr(cr)
	if err := controllerutil.SetControllerReference(cr, ing, r.Scheme); err != nil {
		return err
	}
	foundIng := &netv1.Ingress{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: ing.Name, Namespace: ing.Namespace}, foundIng)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Gitea Ingress")
		err = r.Client.Create(context.TODO(), ing)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Gitea Ingress already exists")
	}

	dep := newGiteaDeploymentCr(cr)
	if err := controllerutil.SetControllerReference(cr, dep, r.Scheme); err != nil {
		return err
	}
	foundDep := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, foundDep)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Gitea Deployment")
		err = r.Client.Create(context.TODO(), dep)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Skip reconcile: Gitea Deployment already exists")
	}
	return nil
}

type GitConfig struct {
	Name         string
	Namespace    string
	Domain       string
	NoReplyEmail string
	SecretKey    string
	Token        string
	DBConf       *DBSecret
	LFSSecret    string
	OauthSecret  string
	OathSecret   string
}

func randomInt(max *big.Int) (int, error) {
	rand, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}

	return int(rand.Int64()), nil
}

func getRandomString(n int) (string, error) {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	buffer := make([]byte, n)
	max := big.NewInt(int64(len(alphanum)))

	for i := 0; i < n; i++ {
		index, err := randomInt(max)
		if err != nil {
			return "", err
		}

		buffer[i] = alphanum[index]
	}

	return string(buffer), nil
}

func genJWTSecret() string {
	JWTSecretBytes := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, JWTSecretBytes)
	if err != nil {
		return ""
	}
	JWTSecretBase64 := base64.RawURLEncoding.EncodeToString(JWTSecretBytes)

	return JWTSecretBase64
}

func newGiteaSecret(dbSecret *DBSecret, cr *gitifold.VCS) (*corev1.Secret, error) {
	config := template.New("config")
	labels := map[string]string{
		"app":        "gitea",
		"component":  "vcs",
		"deployment": "gitifold",
		"instance":   cr.Name,
	}
	name := strings.Join([]string{cr.Name, "-gitifold-gitea"}, "")

	secretBytes := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, secretBytes)
	if err != nil {
		return nil, err
	}

	secretKey := base64.RawURLEncoding.EncodeToString(secretBytes)
	now := time.Now()
	var internalToken string
	internalToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"nbf": now.Unix(),
	}).SignedString([]byte(secretKey))

	secret, err := getRandomString(64)
	if err != nil {
		return nil, err
	}

	data := GitConfig{
		Name:         cr.Name,
		Domain:       cr.Spec.Git.Hostname,
		LFSSecret:    genJWTSecret(),
		OauthSecret:  genJWTSecret(),
		Token:        internalToken,
		SecretKey:    secret,
		NoReplyEmail: cr.Spec.Git.Hostname,
		DBConf:       dbSecret,
		Namespace:    cr.Namespace,
	}
	config, err = config.Parse(`APP_NAME = {{ .Name -}} Git
RUN_MODE = prod
RUN_USER = git

[repository]
ROOT = /data/git/repositories

[repository.local]
LOCAL_COPY_PATH = /data/gitea/tmp/local-repo

[repository.upload]
TEMP_PATH = /data/gitea/uploads

[server]
APP_DATA_PATH    = /data/gitea
SSH_DOMAIN       = {{ .Domain }}
HTTP_PORT        = 3000
ROOT_URL         = https://{{ .Domain -}}/
DISABLE_SSH      = false
SSH_PORT         = 22
LFS_CONTENT_PATH = /data/git/lfs
DOMAIN           = {{ .Domain }}
LFS_START_SERVER = true
LFS_JWT_SECRET   = {{ .LFSSecret }}
OFFLINE_MODE     = true

[database]
DB_TYPE  = postgres
HOST     = {{ .DBConf.Host -}}.{{ .Namespace -}}.svc:5432
NAME     = {{ .DBConf.Name }}
USER     = {{ .DBConf.User }}
PASSWD   = {{ .DBConf.Pass }}
SSL_MODE = disable

[indexer]
ISSUE_INDEXER_PATH = /data/gitea/indexers/issues.bleve

[session]
PROVIDER_CONFIG = network=tcp,addr={{ .Name -}}-gitea-gitifold-keydb.{{ .Namespace -}}.svc:6379,db=0,pool_size=100,idle_timeout=180
PROVIDER        = redis

[cache]
ADAPTER = redis
HOST = network=tcp,addr={{ .Name -}}-gitea-gitifold-keydb.{{ .Namespace -}}.svc:6379,db=1,pool_size=100,idle_timeout=180

[picture]
AVATAR_UPLOAD_PATH      = /data/gitea/avatars
DISABLE_GRAVATAR        = true
ENABLE_FEDERATED_AVATAR = false

[attachment]
PATH = /data/gitea/attachments

[log]
ROOT_PATH = /data/gitea/log
MODE      = console
LEVEL     = Info

[security]
INSTALL_LOCK   = true
SECRET_KEY     = {{ .SecretKey }}
INTERNAL_TOKEN = {{ .Token }}

[service]
DISABLE_REGISTRATION              = false
REQUIRE_SIGNIN_VIEW               = true
REGISTER_EMAIL_CONFIRM            = false
ENABLE_NOTIFY_MAIL                = false
ALLOW_ONLY_EXTERNAL_REGISTRATION  = false
ENABLE_CAPTCHA                    = true
DEFAULT_KEEP_EMAIL_PRIVATE        = false
DEFAULT_ALLOW_CREATE_ORGANIZATION = true
DEFAULT_ENABLE_TIMETRACKING       = true
NO_REPLY_ADDRESS                  = git@{{ .NoReplyEmail }}

[mailer]
ENABLED = false

[oauth2]
ENABLED = true
JWT_SECRET={{ .OauthSecret }}

[metrics]
ENABLED = true

[openid]
ENABLE_OPENID_SIGNIN = true
ENABLE_OPENID_SIGNUP = true`)

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
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"app.ini": str.Bytes(),
		},
	}, nil
}

func newGiteaServiceCr(cr *gitifold.VCS) *corev1.Service {
	labels := map[string]string{
		"app":        "gitea",
		"component":  "vcs",
		"deployment": "gitifold",
		"instance":   cr.Name,
	}
	name := strings.Join([]string{cr.Name, "-gitifold-gitea"}, "")

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
			Type:     corev1.ServiceType("ClusterIP"),
			Ports: []corev1.ServicePort{
				{
					Name:       "gitea-http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(3000),
				},
				{
					Name:       "gitea-ssh",
					Protocol:   "TCP",
					Port:       22,
					TargetPort: intstr.FromInt(22),
				},
			},
		},
	}
}

func newGiteaIngressCr(cr *gitifold.VCS) *netv1.Ingress {
	labels := map[string]string{
		"app":        "gitea",
		"component":  "vcs",
		"deployment": "gitifold",
		"instance":   cr.Name,
	}
	name := strings.Join([]string{cr.Name, "-gitifold-gitea"}, "")

	annotations := make(map[string]string)
	for key, value := range cr.Spec.Git.Annotations {
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
					SecretName: strings.Join([]string{cr.Name, "gitea", "ingress", "tls"}, "-"),
				},
			},
		},
	}
}

func newGiteaPVCCr(cr *gitifold.VCS) *corev1.PersistentVolumeClaim {
	labels := map[string]string{
		"app":        "gitea",
		"component":  "vcs",
		"deployment": "gitifold",
		"instance":   cr.Name,
	}
	name := strings.Join([]string{cr.Name, "-gitifold-gitea"}, "")

	size, _ := resource.ParseQuantity("5Gi")
	/*if cr.Spec.Git.Storage.StorageSize != "" {
		size, _ = resource.ParseQuantity(cr.Spec.Git.Storage.StorageSize)
	}
	*/
	ret := &corev1.PersistentVolumeClaim{
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
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
		},
	}
	/*
		if cr.Spec.Git.Storage.StorageClass != "" {
			ret.Spec.StorageClassName = &cr.Spec.Git.Storage.StorageClass
		}
	*/
	return ret
}

func newGiteaDeploymentCr(cr *gitifold.VCS) *appsv1.Deployment {
	labels := map[string]string{
		"app":        "gitea",
		"component":  "vcs",
		"deployment": "gitifold",
		"instance":   cr.Name,
	}
	name := strings.Join([]string{cr.Name, "-gitifold-gitea"}, "")

	limitCpu, _ := resource.ParseQuantity("250m")
	limitMemory, _ := resource.ParseQuantity("250Mi")
	requestCpu, _ := resource.ParseQuantity("10m")
	requestMemory, _ := resource.ParseQuantity("50Mi")

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
					Volumes: []corev1.Volume{
						{
							Name: "git",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: name,
								},
							},
						},
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: name,
									Items: []corev1.KeyToPath{
										{
											Key:  "app.ini",
											Path: "app.ini",
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "gitea",
							Image: "gitea/gitea:1.11.3",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "git",
									MountPath: "/data",
								},
								{
									Name:      "config",
									MountPath: "/data/gitea/conf/app.ini",
									SubPath:   "app.ini",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 3000,
									Name:          "http",
									Protocol:      "TCP",
								},
								{
									ContainerPort: 22,
									Name:          "ssh",
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
								InitialDelaySeconds: int32(20),
								PeriodSeconds:       int32(6),
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.IntOrString{Type: intstr.String, StrVal: "http"},
									},
								},
								InitialDelaySeconds: int32(10),
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
