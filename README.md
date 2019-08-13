# argo-cd-tokens - Generate tokens for Argo projects declaratively

## What is Argo CD Tokens?

Argo CD Tokens is a controller that will create a Kubernetes Secret to hold a Token for a role of an Argo CD project.

## Why use Argo CD Tokens?

This CRD allows users to forego the process of using the CLI or UI in generating a token. It will also generate a new
token when the current one expires. Event triggers when the secret is updated or deleted and when the token expires.

