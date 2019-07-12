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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// Fills token object and catches error if not possible
	err := r.Get(ctx, req.NamespacedName, &token)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	projName := token.Spec.Project
	roleName := token.Spec.Role

	logCtx.Info(fmt.Sprintf("%s is the project name", projName))
	logCtx.Info(fmt.Sprintf("%s is the role name", roleName))

	// token, err := projIf.CreateToken(context.Background(), &projectpkg.ProjectTokenCreateRequest{Project: projName, Role: roleName, ExpiresIn: int64(duration.Seconds())})

	// conn := argocdclient.NewClientOrDie(clientOpts)

	// tokenString := "practice string"

	namespaceName := types.NamespacedName{
		Namespace: "argocd",
		Name:      "argocd-secret",
	}

	var secret corev1.Secret

	// Fills secret object and catches error if not possible
	err = r.Get(ctx, namespaceName, &secret)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	secretMsg := fmt.Sprintf("%s exists", secret.ObjectMeta.Name)
	dataMsg := fmt.Sprintf("%s exists", secret.Data)
	logCtx.Info(secretMsg)
	logCtx.Info(dataMsg)

	tokenStr := "this is a string"
	// var secret2 corev1.Secret
	secret2, _ := r.createSecret(ctx, tokenStr, logCtx, token)

	secretMsg2 := fmt.Sprintf("%s exists", secret2.ObjectMeta.Name)
	dataMsg2 := fmt.Sprintf("%s exists", secret2.Data)
	namespaceMsg2 := fmt.Sprintf("%s is the ns", secret2.ObjectMeta.Namespace)
	logCtx.Info(secretMsg2)
	logCtx.Info(dataMsg2)
	logCtx.Info(namespaceMsg2)

	return ctrl.Result{}, nil
}

// SetupWithManager s
func (r *TokenReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argoprojlabsv1.Token{}).
		Complete(r)
}

// A helper function to create Secrets from strings
func (r *TokenReconciler) createSecret(ctx context.Context, tknString string, logCtx logr.Logger, token argoprojlabsv1.Token) (*corev1.Secret, error) {

	namespaceName := types.NamespacedName{
		Name:      token.Spec.SecretRef.Name,
		Namespace: token.ObjectMeta.Namespace,
	}

	var secret corev1.Secret

	err := r.Get(ctx, namespaceName, &secret)
	if err == nil {
		logCtx.Info("Secret already exists and will be updated.")
		err = r.Update(ctx, &secret)
		if err != nil {
			logCtx.Info(err.Error())
			return nil, err
		}
		return &secret, nil
	}

	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      token.Spec.SecretRef.Name,
			Namespace: token.ObjectMeta.Namespace,
		},
	}
	err = r.Create(ctx, &secret)
	if err != nil {
		logCtx.Info(err.Error())
		return nil, err
	}
	return &secret, nil
}
