# Deployment Guide

This guide explains how the Smart Charging application is deployed to the Kubernetes cluster using Argo CD and a GitOps workflow.

## Overview

Deployments are fully declarative and GitOps-driven. Argo CD watches the `argo-cd/` directory of this repository and applies any changes automatically to the cluster. No manual `kubectl apply` or Helm commands are needed.

The Argo CD instance itself is managed in a separate repository. This repository only contains the Application manifests that Argo CD should sync.

## Directory Structure

```
argo-cd/
├── project.yaml        # AppProject scoping source repos and destinations
├── namespace.yaml      # Namespace for all smart-charging workloads
└── apps/
    ├── db.yaml         # Application — TimescaleDB
    ├── api.yaml        # Application — Go API
    └── web.yaml        # Application — SvelteKit frontend
```

## Components

Each component is backed by its own Helm chart located in `infra/helm/<component>`. Argo CD renders and applies these charts directly from the repository.

| Application | Helm chart path | Image |
|---|---|---|
| `smart-charging-db` | `infra/helm/db` | `timescale/timescaledb:latest-pg16` |
| `smart-charging-api` | `infra/helm/api` | `harbor.hooyberghs.eu/smart-charging/api` |
| `smart-charging-web` | `infra/helm/web` | `harbor.hooyberghs.eu/smart-charging/web` |

All workloads run in the `smart-charging` namespace.

## Sync Policy

All Applications are configured with automated sync, pruning, and self-healing:

- **Automated sync** — Argo CD watches `HEAD` of `main` and syncs on every push.
- **Prune** — Resources removed from the Helm chart are deleted from the cluster.
- **Self-heal** — Manual cluster mutations are reverted automatically.

## Bootstrapping in the Argo CD repo

In the Argo CD management repository, add a reference to this repo so Argo CD starts watching it. Apply the AppProject and Application manifests once:

```bash
kubectl apply -f https://raw.githubusercontent.com/codr-joe/smart-home-charging/main/argo-cd/project.yaml
kubectl apply -f https://raw.githubusercontent.com/codr-joe/smart-home-charging/main/argo-cd/namespace.yaml
kubectl apply -f https://raw.githubusercontent.com/codr-joe/smart-home-charging/main/argo-cd/apps/db.yaml
kubectl apply -f https://raw.githubusercontent.com/codr-joe/smart-home-charging/main/argo-cd/apps/api.yaml
kubectl apply -f https://raw.githubusercontent.com/codr-joe/smart-home-charging/main/argo-cd/apps/web.yaml
```

After that, all further changes are made through Git — push to `main` and Argo CD reconciles the cluster.

## Pre-deployment Secrets

The following secrets must be created manually in the `smart-home-charging` namespace **before** Argo CD syncs the Applications for the first time:

### TimescaleDB credentials

```bash
kubectl create secret generic timescaledb-secret \
  --namespace smart-home-charging \
  --from-literal=username=smartcharging \
  --from-literal=password='strong-password'
```

### API database URL

```bash
kubectl create secret generic api-secret \
  --namespace smart-home-charging \
  --from-literal=database-url='postgres://smartcharging:strong-password@db.smart-home-charging.svc.cluster.local:5432/smartcharging'
```

These secrets are referenced by name in the Helm values and are never stored in Git.
