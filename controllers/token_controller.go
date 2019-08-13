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
	"os"
	"time"

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
	Log     logr.Logger
	authTkn string
}

// Defines our Patch object we use for updating Secrets
type patchSecretKey struct {
	jwtTkn string
	tkn    argoprojlabsv1.Token
}

func (p *patchSecretKey) Type() types.PatchType {
	return types.MergePatchType
}

func (p *patchSecretKey) Data(obj runtime.Object) ([]byte, error) {
	patch := fmt.Sprintf(updateTokenPatch, p.tkn.Spec.SecretRef.Key, p.jwtTkn)
	return []byte(patch), nil
}

// Reconcile checks if our Secret exists and generates a new Secret or updates a current one
// +kubebuilder:rbac:groups=argoprojlabs.argoproj-labs.io,resources=tokens,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoprojlabs.argoproj-labs.io,resources=tokens/status,verbs=get;update;patch
// +kubebuilder:rbac:resources=secrets,verbs=get;patch;create;list;watch
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

	argoCDClient := argocd.NewArgoCDClient(r.authTkn, token)

	project, err := argoCDClient.GetProject()
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	namespaceName := types.NamespacedName{
		Name:      token.Spec.SecretRef.Name,
		Namespace: token.ObjectMeta.Namespace,
	}

	var tknSecret corev1.Secret

	err = r.Get(ctx, namespaceName, &tknSecret)
	if err == nil {
		jwtTkn := string(tknSecret.Data[token.Spec.SecretRef.Key])
		isTokenExpired, err := jwt.TokenExpired(jwtTkn)
		if err != nil {
			logCtx.Info(err.Error())
			return ctrl.Result{}, nil
		}
		if isTokenExpired {
			err = argoCDClient.DeleteToken(jwtTkn)
			if err != nil {
				logCtx.Info(err.Error())
				return ctrl.Result{}, nil
			}
			jwtTkn, err = argoCDClient.GenerateToken(project)
			if err != nil {
				logCtx.Info(err.Error())
				return ctrl.Result{}, nil
			}
			err = r.patchSecret(ctx, &tknSecret, jwtTkn, logCtx, token)
			if err != nil {
				logCtx.Info(err.Error())
				return ctrl.Result{}, nil
			}
			logCtx.Info("Secret successfully updated!")
			scheduleReconcile := ctrl.Result{RequeueAfter: time.Duration(jwt.TimeTillExpire(jwtTkn)) * time.Second}
			return scheduleReconcile, nil
		}

		logCtx.Info("Secret was not updated, token still valid")
		return ctrl.Result{}, nil
	}

	jwtTkn, err := argoCDClient.GenerateToken(project)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	secret, err := r.createSecret(ctx, jwtTkn, logCtx, token)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	secretMsg := fmt.Sprintf("Secret %s created!", secret.ObjectMeta.Name)
	logCtx.Info(secretMsg)

	scheduleReconcile := ctrl.Result{RequeueAfter: time.Duration(jwt.TimeTillExpire(jwtTkn)) * time.Second}
	return scheduleReconcile, nil
}

// SetupWithManager sets up secrets to be watched and gets auth tkn to login to argocd
func (r *TokenReconciler) SetupWithManager(mgr ctrl.Manager) error {

	r.authTkn = os.Getenv("AUTH_TKN")
	fmt.Println(r.authTkn)

	return ctrl.NewControllerManagedBy(mgr).
		For(&argoprojlabsv1.Token{}).
		Watches(&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {

					ctx := context.Background()
					var allTkns argoprojlabsv1.TokenList
					tknMatches := make([]argoprojlabsv1.Token, 0)

					err := r.List(ctx, &allTkns)
					if err != nil {
						return []reconcile.Request{}
					}

					for _, token := range allTkns.Items {
						if a.Meta.GetName() == token.Spec.SecretRef.Name {
							tknMatches = append(tknMatches, token)
						}
					}

					requests := make([]reconcile.Request, 0)

					for _, token := range tknMatches {
						namespaceName := types.NamespacedName{
							Name:      token.Name,
							Namespace: token.Namespace,
						}
						requests = append(requests, reconcile.Request{NamespacedName: namespaceName})
					}

					return requests
				}),
			}).
		Complete(r)
}

// A helper function to create Secrets from strings
func (r *TokenReconciler) createSecret(ctx context.Context, tknStr string, logCtx logr.Logger, token argoprojlabsv1.Token) (*corev1.Secret, error) {

	var secret corev1.Secret

	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      token.Spec.SecretRef.Name,
			Namespace: token.ObjectMeta.Namespace,
		},
		StringData: map[string]string{
			token.Spec.SecretRef.Key: tknStr,
		},
	}
	err := r.Create(ctx, &secret)
	if err != nil {
		logCtx.Info(err.Error())
		return nil, err
	}
	return &secret, nil
}

// patchSecret updates an expired token within a Secret with a new oen
func (r *TokenReconciler) patchSecret(ctx context.Context, tknSecret *corev1.Secret, tknStr string, logCtx logr.Logger, token argoprojlabsv1.Token) error {

	logCtx.Info("Secret already exists and will be updated.")

	patch := &patchSecretKey{
		jwtTkn: tknStr,
		tkn:    token,
	}
	err := r.Patch(ctx, tknSecret, patch)
	if err != nil {
		logCtx.Info(err.Error())
		return err
	}

	return nil
}
