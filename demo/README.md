# Argo CD Tokens Demo

## Prerequisites 
- A running Argo CD instance (Check out [getting started guide](https://argoproj.github.io/argo-cd/getting_started/))
- kubectl 
- Argo CD auth token stored in environment variable `ARGOCD_AUTH_TOKEN`

## Create Secret with Argo CD auth token for token controller to consume 

```bash
kubectl create namespace argo-cd-tokens-system
kubectl create secret generic argocd-auth-token -n argo-cd-tokens-system --from-literal=authTkn=$ARGOCD_AUTH_TOKEN
```

## Create Token Controller through Argo CD

```bash
argocd app create token-controller --dest-namespace argo-cd-tokens-system --dest-server https://kubernetes.default.svc --repo github.com/dpadhiar/argo-cd-tokens --path config/default
argocd app sync token-controller
```

## Create Deployment without token secret mounted

```bash
kubectl apply -f https://github.com/dpadhiar/argo-cd-tokens/blob/master/demo/deployment-without-secret.yaml
kubectl exec -it <POD_NAME> /bin/sh
argocd get token-controller
```

## Create Token CRD instance

```bash
kubectl apply -f https://github.com/dpadhiar/argo-cd-tokens/blob/master/demo/token.yaml
```

## Modify Deployment to mount the generated token `kubectl apply`

```bash
kubectl apply -f https://github.com/dpadhiar/argo-cd-tokens/blob/master/demo/deployment-with-secret.yaml
kubectl exec -it <POD_NAME> /bin/sh
argocd get token-controller
```