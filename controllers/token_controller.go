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
	"bytes"
	"context"
	"crypto/tls"
	"fmt"

	"encoding/json"
	"io/ioutil"
	"net/http"

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

// PostRequest used for RequestPayload
type PostRequest struct {
	ExpiresIn int
	Project   string
	Role      string
}

// AppProject provides a logical grouping of applications, providing controls for:
// * where the apps may deploy to (cluster whitelist)
// * what may be deployed (repository whitelist, resource whitelist/blacklist)
// * who can access these applications (roles, OIDC group claims bindings)
// * and what they can do (RBAC policies)
// * automation access to these roles (JWT tokens)
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:path=appprojects,shortName=appproj;appprojs
type AppProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              AppProjectSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// AppProjectSpec is the specification of an AppProject
type AppProjectSpec struct {
	// SourceRepos contains list of git repository URLs which can be used for deployment
	SourceRepos []string `json:"sourceRepos,omitempty" protobuf:"bytes,1,name=sourceRepos"`
	// Destinations contains list of destinations available for deployment
	Destinations []ApplicationDestination `json:"destinations,omitempty" protobuf:"bytes,2,name=destination"`
	// Description contains optional project description
	Description string `json:"description,omitempty" protobuf:"bytes,3,opt,name=description"`
	// Roles are user defined RBAC roles associated with this project
	Roles []ProjectRole `json:"roles,omitempty" protobuf:"bytes,4,rep,name=roles"`
	// ClusterResourceWhitelist contains list of whitelisted cluster level resources
	ClusterResourceWhitelist []metav1.GroupKind `json:"clusterResourceWhitelist,omitempty" protobuf:"bytes,5,opt,name=clusterResourceWhitelist"`
	// NamespaceResourceBlacklist contains list of blacklisted namespace level resources
	NamespaceResourceBlacklist []metav1.GroupKind `json:"namespaceResourceBlacklist,omitempty" protobuf:"bytes,6,opt,name=namespaceResourceBlacklist"`
}

// ApplicationDestination contains deployment destination information
type ApplicationDestination struct {
	// Server overrides the environment server value in the ksonnet app.yaml
	Server string `json:"server,omitempty" protobuf:"bytes,1,opt,name=server"`
	// Namespace overrides the environment namespace value in the ksonnet app.yaml
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,2,opt,name=namespace"`
}

// ProjectRole represents a role that has access to a project
type ProjectRole struct {
	// Name is a name for this role
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Description is a description of the role
	Description string `json:"description,omitempty" protobuf:"bytes,2,opt,name=description"`
	// Policies Stores a list of casbin formated strings that define access policies for the role in the project
	Policies []string `json:"policies,omitempty" protobuf:"bytes,3,rep,name=policies"`
	// JWTTokens are a list of generated JWT tokens bound to this role
	JWTTokens []JWTToken `json:"jwtTokens,omitempty" protobuf:"bytes,4,rep,name=jwtTokens"`
	// Groups are a list of OIDC group claims bound to this role
	Groups []string `json:"groups,omitempty" protobuf:"bytes,5,rep,name=groups"`
}

// JWTToken holds the issuedAt and expiresAt values of a token
type JWTToken struct {
	IssuedAt  int64 `json:"iat" protobuf:"int64,1,opt,name=iat"`
	ExpiresAt int64 `json:"exp,omitempty" protobuf:"int64,2,opt,name=exp"`
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

	// https://www.socketloop.com/tutorials/golang-disable-security-check-for-http-ssl-with-bad-or-expired-certificate
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore expired SSL certificates
	}
	client := &http.Client{Transport: transCfg}

	request, err := http.NewRequest("GET", "https://10.107.242.190/api/v1/projects/default", nil)

	loginCookie := http.Cookie{
		Name:     "argocd.token",
		Value:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE1NjI5MDI3MjAsImlzcyI6ImFyZ29jZCIsIm5iZiI6MTU2MjkwMjcyMCwic3ViIjoiYWRtaW4ifQ.j0tOpDRSgHesKZw8Ghkzqa_yaRi5sDzqQw24a78AbPs",
		Path:     "/",
		MaxAge:   60 * 60,
		HttpOnly: true,
	}

	request.AddCookie(&loginCookie)

	response, err := client.Do(request)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	var project AppProject

	err = json.Unmarshal(body, &project)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	postReq := PostRequest{
		ExpiresIn: 10,
		Project:   token.Spec.Project,
		Role:      token.Spec.Role,
	}

	bytePostReq, err := json.Marshal(postReq)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	request, err = http.NewRequest("POST", "https://10.107.242.190/api/v1/projects/default/roles/TestRole/token", bytes.NewBuffer(bytePostReq))
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&loginCookie)

	response, err = client.Do(request)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	defer response.Body.Close()

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		logCtx.Info(err.Error())
		return ctrl.Result{}, nil
	}

	fmt.Println(string(body))

	tokenStr := "this will come from ArgoCD eventually"
	secret, wasPatched, err := r.createSecret(ctx, tokenStr, logCtx, token)
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
							fmt.Println(token.Name)
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

/* func fetchToken(url, loginToken string) (tokenStr string, err error) {

	resp, err := http.Get("http://example.com/")
} */
