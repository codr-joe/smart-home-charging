---
description: 'These instructions apply to all files in the infra/ folder and should be followed by all developers working on the application.'
applyTo: '**/*'
---

# Infrastructure Instructions

## Kubernetes

The application must be deployed on a bare metal Kubernetes cluster.

### Traefik

Traefik is our chosen ingress controller utilizing the Kubernetes Gateway API. Each application component (backend, frontend, database) should have its own Gateway and corresponding HTTPRoute resources defined in the Helm chart under the `/helm` directory. When making changes to the infrastructure, you should update the corresponding YAML files in this directory and ensure that Traefik can successfully route traffic to the application components.

### Cert-Manager

We use Cert-Manager to manage our TLS certificates. We have set up a ClusterIssuer called `letsencrypt-production` that uses Let's Encrypt to issue certificates for our application. When making changes to the infrastructure, you should ensure that the necessary annotations are added to the Gateway resources to request certificates from Cert-Manager and that the certificates are issued successfully.

### Helm Charts

To deploy our application to the Kubernetes cluster, we use Helm charts. Each component of the application (backend, frontend, database) has its own Helm chart located in the `/helm` directory. When making changes to the infrastructure, you should update the corresponding Helm chart and ensure that it can be deployed successfully to the cluster.

## Container Registry

We use Harbor as our container registry. The registry is hosted at `harbor.hooyberghs.eu`. All images must be pushed to this registry before they can be deployed to the Kubernetes cluster. You can use the `smart-charging` project in Harbor to organize your images.

## Argo CD

To deploy our application to the Kubernetes cluster, we use Argo CD. Argo CD is a declarative, GitOps continuous delivery tool for Kubernetes. We have set up Argo CD to monitor the `argo-cd/` directory in our GitHub repository. When you make changes to the infrastructure, you should update the corresponding YAML files in the `argo-cd/` directory and ensure that Argo CD can successfully deploy the changes to the cluster.
