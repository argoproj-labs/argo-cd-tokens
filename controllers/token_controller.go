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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	argoprojlabsv1 "github.com/dpadhiar/argo-cd-tokens/api/v1"
	"github.com/dpadhiar/argo-cd-tokens/utils/argocd"
	"github.com/dpadhiar/argo-cd-tokens/utils/jwt"
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

	//authTkn := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE1NjI5MDI3MjAsImlzcyI6ImFyZ29jZCIsIm5iZiI6MTU2MjkwMjcyMCwic3ViIjoiYWRtaW4ifQ.j0tOpDRSgHesKZw8Ghkzqa_yaRi5sDzqQw24a78AbPs"
	newTkn := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NjMxNTE0MTIsImlhdCI6MTU2MzE1MDgxMiwiaXNzIjoiYXJnb2NkIiwibmJmIjoxNTYzMTUwODEyLCJzdWIiOiJwcm9qOmRlZmF1bHQ6VGVzdFJvbGUifQ.A-RWmt0FBqDxjhTRAHvsFPFzxI15zr5ILsqPTD9qqrw"
	jwt.CheckParse(newTkn)
	tokenExpired := jwt.TokenExpired(newTkn)
	fmt.Println(tokenExpired)

	namespaceName := types.NamespacedName{
		Name:      "argocd-login",
		Namespace: "argocd",
	}

	var loginSecret corev1.Secret
	err = r.Get(ctx, namespaceName, &loginSecret)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	authTkn := string(loginSecret.Data["authTkn"])

	argoCDClient := argocd.NewArgoCDClient(authTkn, token)

	project, err := argoCDClient.GetProject()
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	tknString, err := argoCDClient.GenerateToken(project)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	secret, wasPatched, err := r.createSecret(ctx, tknString, logCtx, token)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	if wasPatched {
		secretMsg := fmt.Sprintf("Secret %s updated!", secret.ObjectMeta.Name)
		logCtx.Info(secretMsg)
	} else {
		secretMsg := fmt.Sprintf("Secret %s created!", secret.ObjectMeta.Name)
		logCtx.Info(secretMsg)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager s
func (r *TokenReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&argoprojlabsv1.Token{}).
		Watches(&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {

					ctx := context.Background()
					var tknList argoprojlabsv1.TokenList
					tknMatches := make([]argoprojlabsv1.Token, 0)

					err := r.List(ctx, &tknList)
					if err != nil {
						return []reconcile.Request{}
					}

					for _, token := range tknList.Items {
						if a.Meta.GetName() == token.Spec.SecretRef.Name {
							//fmt.Println(token.Name)
							tknMatches = append(tknMatches, token)
						}
					}

					requestArr := make([]reconcile.Request, 0)

					for _, token := range tknMatches {
						namespaceName := types.NamespacedName{
							Name:      token.Name,
							Namespace: token.Namespace,
						}
						requestArr = append(requestArr, reconcile.Request{NamespacedName: namespaceName})
					}

					return requestArr
				}),
			}).
		Complete(r)
}

// A helper function to create Secrets from strings
func (r *TokenReconciler) createSecret(ctx context.Context, tknStr string, logCtx logr.Logger, token argoprojlabsv1.Token) (*corev1.Secret, bool, error) {

	namespaceName := types.NamespacedName{
		Name:      token.Spec.SecretRef.Name,
		Namespace: token.ObjectMeta.Namespace,
	}

	var secret corev1.Secret

	err := r.Get(ctx, namespaceName, &secret)
	fmt.Println(string(secret.Data[token.Spec.SecretRef.Key]))
	if err == nil {
		logCtx.Info("Secret already exists and will be updated.")
		patch := &patchSecretKey{
			tknString: tknStr,
			tkn:       token,
		}
		err = r.Patch(ctx, &secret, patch)
		if err != nil {
			logCtx.Info(err.Error())
			return nil, false, err
		}
		return &secret, true, nil
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
		return nil, false, err
	}
	return &secret, false, nil
}
