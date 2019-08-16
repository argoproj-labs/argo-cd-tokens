package argocd

import (
	"bytes"
	"crypto/tls"
	"fmt"

	"encoding/json"
	"io/ioutil"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	argoprojlabsv1 "github.com/argoproj-labs/argo-cd-tokens/api/v1"
	jwt "github.com/argoproj-labs/argo-cd-tokens/utils/jwt"
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

// Client holds our client, the cookie used to login and a token object
type Client struct {
	client      http.Client
	loginCookie http.Cookie
	token       argoprojlabsv1.Token
}

// NewArgoCDClient constructs a Client object
func NewArgoCDClient(authTkn string, token argoprojlabsv1.Token) Client {

	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	loginCookie := http.Cookie{
		Name:     "argocd.token",
		Value:    authTkn,
		Path:     "/",
		MaxAge:   60 * 60,
		HttpOnly: true,
	}

	argoCDClient := Client{
		client:      http.Client{Transport: transCfg},
		loginCookie: loginCookie,
		token:       token,
	}

	return argoCDClient
}

// GetProject pings ArgoCD for the project which we will create a token for
func (a *Client) GetProject() (AppProject, error) {

	argoCDEndpt := fmt.Sprintf("%s/api/v1/projects/%s", a.token.Spec.ArgoCDEndpt, a.token.Spec.Project)

	request, err := http.NewRequest("GET", argoCDEndpt, nil)

	request.AddCookie(&a.loginCookie)

	var project AppProject

	response, err := a.client.Do(request)
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

// GenerateToken uses a project to create a token pertaining to a specified role
func (a *Client) GenerateToken(project AppProject) (string, error) {

	roleExistence := roleExists(a.token.Spec.Role, project)
	if !roleExistence {
		return "", fmt.Errorf("The role does not exist")
	}

	argoCDEndpt := fmt.Sprintf("%s/api/v1/projects/%s/roles/%s/token", a.token.Spec.ArgoCDEndpt, a.token.Spec.Project, a.token.Spec.Role)

	postReq := PostRequest{
		ExpiresIn: a.token.Spec.ExpiresIn,
		Project:   a.token.Spec.Project,
		Role:      a.token.Spec.Role,
	}

	bytePostReq, err := json.Marshal(postReq)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest("POST", argoCDEndpt, bytes.NewBuffer(bytePostReq))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&a.loginCookie)

	response, err := a.client.Do(request)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var tkn Token
	err = json.Unmarshal(body, &tkn)
	if err != nil {
		return "", err
	}

	return tkn.Token, nil
}

// DeleteToken removes expired tokens from ArgoCD
func (a *Client) DeleteToken(token string) error {

	tokenIAT := jwt.ReturnIAT(token)

	argoCDEndpt := fmt.Sprintf("%s/api/v1/projects/%s/roles/%s/token/%d", a.token.Spec.ArgoCDEndpt, a.token.Spec.Project, a.token.Spec.Role, tokenIAT)

	request, err := http.NewRequest("DELETE", argoCDEndpt, nil)

	request.AddCookie(&a.loginCookie)

	response, err := a.client.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	return err
}

// check if the role exists within the given project
func roleExists(roleName string, project AppProject) bool {

	for i := range project.Spec.Roles {
		if project.Spec.Roles[i].Name == roleName {
			return true
		}
	}

	return false
}
