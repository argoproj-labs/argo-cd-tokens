# Argo CD Tokens Demo

## Prerequisites 
- A running Argo CD instance (Check out [getting started guide](https://argoproj.github.io/argo-cd/getting_started/))
- kubectl 
- Argo CD auth token stored in environment variable `ARGOCD_AUTH_TOKEN`

## Create Argo CD project declaratively

```bash
kubectl apply -f https://raw.githubusercontent.com/dpadhiar/argo-cd-tokens/master/demo/project.yaml
```

## Create Secret with Argo CD auth token for token controller to consume 

```bash
kubectl create namespace argo-cd-tokens-system
kubectl create secret generic argocd-auth-token -n argo-cd-tokens-system --from-literal=authTkn=$ARGOCD_AUTH_TOKEN
```

## Create Token Controller through Argo CD

```bash
argocd app create token-controller --dest-namespace argo-cd-tokens-system --dest-server https://kubernetes.default.svc --repo https://github.com/dpadhiar/argo-cd-tokens --path config/default --project token-controller
argocd app sync token-controller
```

## Create Token CRD instance

```bash
kubectl apply -f https://raw.githubusercontent.com/dpadhiar/argo-cd-tokens/master/demo/token.yaml
```

## Create Deployment to mount the generated token `kubectl apply`

```bash
kubectl apply -f https://raw.githubusercontent.com/dpadhiar/argo-cd-tokens/master/demo/deployment_with_secret.yaml
kubectl exec -it <POD_NAME> /bin/bash
```

The first two commands will succeed but the delete command will fail. The role for that JWT token specificed by AUTH_TKN does not have the permissions to delete the token-controller application.
```bash
argocd app --server cd.apps.argoproj.io:443 --grpc-web --auth-token $AUTH_TKN get token-controller
argocd app --server cd.apps.argoproj.io:443 --grpc-web --auth-token $AUTH_TKN delete token-controller
```
