# Helm Starter

This is a helm chart from Helm Starter.

## Prerequisites

- Kubernetes 1.14+
- [helm secret](https://github.com/zendesk/helm-secrets)

## Install

To install this chart, you simply run

```
helm -n my_namespace secrets install my_release . -f secrets/secrets.yaml
```

## Configuration

The following table lists the configurable parameters of the chart and their default values.

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `image.repository` | Image repository | `busybox` |
| `image.tag` | Image tag | `""` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `imagePullSecrets` | Reference to one or more secrets to be used when pulling images | `[]` |
| `replicaCount` | Number of replicas | `1` |
| `ingress.enable` | Create Ingress | `false` |
| `ingress.enternal_dns` | Create domain name | `true` |
| `ingress.hosts` | Associate hosts with the Ingress | `{}` |
| `service.type` | Type of service to create | `ClusterIP` |
| `service.port` | Service port to expose | `80` |
| `certificate.enable` | Issue certificate | `false` |
| `certificate.issuerRef.name` | Issuer name for issuing certificate | `letsencrypt-staging-dns` |
| `certificate.issuerRef.kind` | Issuer kind | `ClusterIssuer` |
| `resources` | CPU/memory resource requests/limits | `{}` |
| `podAnnotations` | Annotations for pod | `{}` |
| `podSecurityContext` | Pod securityContext | `{}` |
| `securityContext` | container securityContext | `{}` |
| `nodeSelector` | Node labels for pod assignment | `{}` |
| `tolerations` | Node tolerations for pod assignment | `[]` |
| `affinity` | Node affinity for pod assignment | `{}` |

