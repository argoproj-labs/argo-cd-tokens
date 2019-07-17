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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	argoprojlabsv1 "github.com/dpadhiar/argo-cd-tokens/api/v1"
)

const (
	updateTokenPatch = `{
	"stringData": {
			"%s": "%s"
	}
}`
)

// TokenReconciler reconciles a Token object
type TokenReconciler struct {
	client.Client
	Log logr.Logger
}

// Defines our Patch object we use for updating Secrets
type patchSecretKey struct {
	tknString string
	tkn       argoprojlabsv1.Token
}

func (p *patchSecretKey) Type() types.PatchType {
	return types.MergePatchType
}

func (p *patchSecretKey) Data(obj runtime.Object) ([]byte, error) {
	patch := fmt.Sprintf(updateTokenPatch, p.tkn.Spec.SecretRef.Key, p.tknString)
	return []byte(patch), nil
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

	// conn := argocdclient.NewClientOrDie(clientOpts)
	// TODO: projIf, err :=

	// token, err := projIf.CreateToken(context.Background(), &projectpkg.ProjectTokenCreateRequest{Project: projName, Role: roleName, ExpiresIn: int64(duration.Seconds())})

	// tokenString := "practice string"

	tokenStr := "let's try updating now"
	secret, _ := r.createSecret(ctx, tokenStr, logCtx, token)

	secretMsg := fmt.Sprintf("%s exists", secret.ObjectMeta.Name)
	dataMsg := fmt.Sprintf("%s exists", secret.Data)
	namespaceMsg := fmt.Sprintf("%s is the ns", secret.ObjectMeta.Namespace)
	logCtx.Info(secretMsg)
	logCtx.Info(dataMsg)
	logCtx.Info(namespaceMsg)

	return ctrl.Result{}, nil
}

var (
	secretOwnerKey = ".metadata.controller"
)

// SetupWithManager s
func (r *TokenReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(&corev1.Secret{}, "metadata.name", func(rawObj runtime.Object) []string {
		// grab the object
		ctx := context.Background()
		var tknList argoprojlabsv1.TokenList
		fmt.Println("Function was entered")

		secret := rawObj.(*corev1.Secret)
		tknNames := make([]string, 0)

		err := r.List(ctx, &tknList)
		fmt.Println(tknList.Items[0].Name)
		if err != nil {
			return tknNames
		}

		for _, token := range tknList.Items {
			if secret.Name == token.Spec.SecretRef.Name {
				//tknNames = append(tknNames, token.Name)
				s := fmt.Sprintf("%s/%s", token.Namespace, token.Name)
				tknNames = append(tknNames, s)
			}
		}

		return tknNames
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&argoprojlabsv1.Token{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// A helper function to create Secrets from strings
func (r *TokenReconciler) createSecret(ctx context.Context, tknStr string, logCtx logr.Logger, token argoprojlabsv1.Token) (*corev1.Secret, error) {

	namespaceName := types.NamespacedName{
		Name:      token.Spec.SecretRef.Name,
		Namespace: token.ObjectMeta.Namespace,
	}

	var secret corev1.Secret

	err := r.Get(ctx, namespaceName, &secret)
	if err == nil {
		logCtx.Info("Secret already exists and will be updated.")
		patch := &patchSecretKey{
			tknString: tknStr,
			tkn:       token,
		}
		err = r.Patch(ctx, &secret, patch)
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
		StringData: map[string]string{
			token.Spec.SecretRef.Key: tknStr,
		},
	}
	err = r.Create(ctx, &secret)
	if err != nil {
		logCtx.Info(err.Error())
		return nil, err
	}
	return &secret, nil
}
