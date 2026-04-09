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

## Github Actions Runner Controller (ARC)

All our infrastructure is not reachable from public internet, therefore to allow Github Actions to deploy to our Kubernetes cluster or to push images to the Container Registry, we use the Github Actions Runner Controller (ARC). This allows us to run self-hosted runners in our Kubernetes cluster that can execute our deployment workflows. The ARC is configured to automatically scale the number of runners based on the workload, ensuring that we have enough capacity to handle our deployment needs while minimizing costs. You can use the runner called `smart-charging-runner` for your deployment workflows.
