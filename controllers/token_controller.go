/*

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
	"fmt"

	argocdclient "github.com/argoproj/argo-cd/pkg/apiclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	argoprojlabsv1 "github.com/dpadhiar/argo-cd-tokens/api/v1"
)

// TokenReconciler reconciles a Token object
type TokenReconciler struct {
	client.Client
	Log logr.Logger
}

// Reconcile s
// +kubebuilder:rbac:groups=argoprojlabs.argoproj-labs.io,resources=tokens,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoprojlabs.argoproj-labs.io,resources=tokens/status,verbs=get;update;patch
func (r *TokenReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logCtx := r.Log.WithValues("token", req.NamespacedName)

	var token argoprojlabsv1.Token

	err := r.Get(ctx, req.NamespacedName, &token)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	projName := token.Spec.Project
	roleName := token.Spec.Role

	conn := argocdclient.NewClientOrDie(clientOpts)

	namespaceName := types.NamespacedName{
		Namespace: "argocd",
		Name:      "argocd-secret",
	}
	var secret corev1.Secret
	err = r.Get(ctx, namespaceName, &secret)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	secretMsg := fmt.Sprintf("%s exists", secret.ObjectMeta.Name)
	logCtx.Info(secretMsg)
	// your logic here

	// var

	return ctrl.Result{}, nil
}

// SetupWithManager s
func (r *TokenReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argoprojlabsv1.Token{}).
		Complete(r)
}
