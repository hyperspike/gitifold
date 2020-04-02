/*
Copyright 2020 Dan Molik <dan@hyperspike.io>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gitifold "hyperspike.io/eng/gitifold/api/v1beta1"
)

// VCSReconciler reconciles a VCS object
type VCSReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gitifold.hyperspike.io,resources=vcs,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=gitifold.hyperspike.io,resources=vcs/status,verbs=get;update;patch

// +kubebuilder:rbac:groups="";networking.k8s.io;apps;rbac.authorization.k8s.io,resources=statefulesets;services;secrets;configmaps;deployments;ingresses;persistentvolumeclaims;serviceaccounts;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *VCSReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	logger := r.Log.WithValues("pipeline", req.NamespacedName)
	logger.Info("reconciling")

	instance := &gitifold.VCS{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Gitea Components
	dbSecret, err := createPgService("gitea", instance, r)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err = createKeyDBService("gitea", instance, r); err != nil {
		return ctrl.Result{}, err
	}
	if err = createGiteaService(dbSecret, instance, r); err != nil {
		return ctrl.Result{}, err
	}

	if _, err = createPgService("drone", instance, r); err != nil {
		return ctrl.Result{}, err
	}
	if err = createDroneService(instance, r); err != nil {
		return ctrl.Result{}, err
	}
	if err = createRegistryService(instance, r); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *VCSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gitifold.VCS{}).
		Complete(r)
}
