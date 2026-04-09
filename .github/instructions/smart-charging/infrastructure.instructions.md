---
description: 'These instructions apply to all files in the infra/ folder and should be followed by all developers working on the application.'
applyTo: '**/*'
---

# Infrastructure Instructions

## Kubernetes

The application must be deployed on a bare metal Kubernetes cluster.

### Ingress

For ingress, we must use Traefik with Kubernetes Gateway API. We must enforce HTTPS using Let's Encrypt certificates by making use of Cert-Manager ClusterIssuer called `letsencrypt-production`.

### Helm Charts

To deploy our application to the Kubernetes cluster, we use Helm charts. Each component of the application (backend, frontend, database) has its own Helm chart located in the `infra/helm/` directory. When making changes to the infrastructure, you should update the corresponding Helm chart and ensure that it can be deployed successfully to the cluster.

## Container Registry

We use Harbor as our container registry. The registry is hosted at `harbor.hooyberghs.eu`. All images must be pushed to this registry before they can be deployed to the Kubernetes cluster. You can use the `smart-charging` project in Harbor to organize your images.

## Argo CD

To deploy our application to the Kubernetes cluster, we use Argo CD. Argo CD is a declarative, GitOps continuous delivery tool for Kubernetes. We have set up Argo CD to monitor the `argo-cd/` directory in our GitHub repository. When you make changes to the infrastructure, you should update the corresponding YAML files in the `argo-cd/` directory and ensure that Argo CD can successfully deploy the changes to the cluster.
