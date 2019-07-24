package controllers

import (
	"bytes"
	"crypto/tls"
	"fmt"

	"encoding/json"
	"io/ioutil"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	argoprojlabsv1 "github.com/dpadhiar/argo-cd-tokens/api/v1"
)

// PostRequest used for RequestPayload
type PostRequest struct {
	ExpiresIn int
	Project   string
	Role      string
}

// Token used to store token string generated from ArgoCD
type Token struct {
	Token string
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

// GetProject TODO
func GetProject(token argoprojlabsv1.Token) (AppProject, error) {

	// will probably be moved to a configuration
	// https://www.socketloop.com/tutorials/golang-disable-security-check-for-http-ssl-with-bad-or-expired-certificate
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore expired SSL certificates
	}
	client := &http.Client{Transport: transCfg}

	argoCDEndpt := token.Spec.ArgoCDEndpt
	argoCDEndpt = fmt.Sprint(argoCDEndpt, token.Spec.Project)

	request, err := http.NewRequest("GET", argoCDEndpt, nil)

	loginCookie := http.Cookie{
		Name:     "argocd.token",
		Value:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE1NjI5MDI3MjAsImlzcyI6ImFyZ29jZCIsIm5iZiI6MTU2MjkwMjcyMCwic3ViIjoiYWRtaW4ifQ.j0tOpDRSgHesKZw8Ghkzqa_yaRi5sDzqQw24a78AbPs",
		Path:     "/",
		MaxAge:   60 * 60,
		HttpOnly: true,
	}

	request.AddCookie(&loginCookie)

	var project AppProject

	response, err := client.Do(request)
	if err != nil {
		return project, err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return project, err
	}

	err = json.Unmarshal(body, &project)
	if err != nil {
		return project, err
	}

	return project, nil
}

// GenerateToken TODO
func GenerateToken(project AppProject, token argoprojlabsv1.Token) (string, error) {

	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transCfg}

	loginCookie := http.Cookie{
		Name:     "argocd.token",
		Value:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE1NjI5MDI3MjAsImlzcyI6ImFyZ29jZCIsIm5iZiI6MTU2MjkwMjcyMCwic3ViIjoiYWRtaW4ifQ.j0tOpDRSgHesKZw8Ghkzqa_yaRi5sDzqQw24a78AbPs",
		Path:     "/",
		MaxAge:   60 * 60,
		HttpOnly: true,
	}

	argoCDEndpt := token.Spec.ArgoCDEndpt
	argoCDEndpt = fmt.Sprint(argoCDEndpt, token.Spec.Project, "/roles/", token.Spec.Role, "/token")

	postReq := PostRequest{
		ExpiresIn: token.Spec.ExpiresIn,
		Project:   token.Spec.Project,
		Role:      token.Spec.Role,
	}

	bytePostReq, err := json.Marshal(postReq)
	if err != nil {
		return "", nil
	}

	request, err := http.NewRequest("POST", argoCDEndpt, bytes.NewBuffer(bytePostReq))
	if err != nil {
		return "", nil
	}
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&loginCookie)

	response, err := client.Do(request)
	if err != nil {
		return "", nil
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", nil
	}

	var tkn Token
	err = json.Unmarshal(body, &tkn)
	if err != nil {
		return "", nil
	}

	return tkn.Token, nil
}
