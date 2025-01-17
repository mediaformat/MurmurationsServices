# How to Install Nginx Controller

## Installation

Follow [Official Installation Guide](https://kubernetes.github.io/ingress-nginx/deploy/) for more information.

**Development**

```
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm upgrade --install ingress-nginx ingress-nginx \
  --repo https://kubernetes.github.io/ingress-nginx \
  --namespace ingress-nginx --create-namespace
```
Note: k8s version v1.24.0 is not compatible with the current latest ingress-nginx v4.1.4. If the above command is not working, you can add `--version 4.0.19` after the command.

**Production / Staging**

```
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
kubectl create namespace ingress-nginx
helm upgrade ingress-nginx ingress-nginx/ingress-nginx -f docs/ingress-nginx/values-ingress-nginx.yaml -n ingress-nginx --install
```
