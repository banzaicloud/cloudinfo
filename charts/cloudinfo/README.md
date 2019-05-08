# Cloud Info Helm Chart

[Cloud Info](https://github.com/banzaicloud/cloudinfo) provides resource and pricing information about products available on supported cloud providers. 

## tl;dr:

```bash
$ helm repo add banzaicloud-stable https://kubernetes-charts.banzaicloud.com
$ helm repo update
$ helm install banzaicloud-stable/cloudinfo
```

## Introduction

This chart bootstraps a [Cloud Info](https://hub.helm.sh/charts/banzaicloud-stable/cloudinfo) deployment on a
[Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.10+ with Beta APIs enabled

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
$ helm install --name my-release banzaicloud-stable/cloudinfo
```

The command deploys Cloud Info on the Kubernetes cluster with the default configuration.
The [configuration](#configuration) section lists the parameters that can be configured during installation.

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following tables lists the configurable parameters of the cloudinfo chart and their default values.

| Parameter                                  | Description                                           | Default                 |
|--------------------------------------------|-------------------------------------------------------|-------------------------|
| `image.repository`                         | Container image repository                            | `banzaicloud/cloudinfo` |
| `image.tag`                                | Container image tag                                   | `latest`                |
| `image.pullPolicy`                         | Container pull policy                                 | `IfNotPresent`          |
| `service.type`                             | The Kubernetes service type to use                    | `ClusterIP`             |
| `service.port`                             | The Kubernetes service port to use                    | `80`                    |
| `app.logLevel`                             | Log level                                             | `info`                  |
| `app.basePath`                             | Application base path                                 | `/`                     |
| `providers.amazon.enabled`                 | Enable Amazon provider                                | `false`                 |
| `providers.amazon.awsAccessKeyId`          | Amazon Access Key ID                                  | `""`                    |
| `providers.amazon.awsSecretAccessKey`      | Amazon Secret Access Key                              | `""`                    |
| `providers.google.enabled`                 | Enable Google provider                                | `false`                 |
| `providers.google.gceApiKey`               | GCE API Key                                           | `""`                    |
| `providers.google.gceCredentials`          | GCE Credential file (encoded by base64)               | `""`                    |
| `providers.alibaba.enabled`                | Enable Alibaba provider                               | `false`                 |
| `providers.alibaba.alibabaAccessKeyId`     | Alibaba Access Key ID                                 | `""`                    |
| `providers.alibaba.alibabaAccessKeySecret` | Alibaba Access Key Secret                             | `""`                    |
| `providers.alibaba.alibabaRegionId`        | Alibaba Region ID                                     | `""`                    |
| `providers.oracle.enabled`                 | Enable Oracle provider                                | `false`                 |
| `providers.oracle.ociUser`                 | The OCID of the user                                  | `""`                    |
| `providers.oracle.ociTenancy`              | The OCID of the tenancy                               | `""`                    |
| `providers.oracle.ociRegion`               | Specific region for OCI                               | `""`                    |
| `providers.oracle.ociKey`                  | The key pair must be in PEM format (encode by base64) | `""`                    |
| `providers.oracle.ociFingerprint`          | Fingerprint for the key pair being used               | `""`                    |
| `providers.azure.enabled`                  | Enable Azure provider                                 | `false`                 |
| `providers.azure.azureCredentials`         | Azure Credential file (encoded by base64)             | `""`                    |
| `deploymentLabels`                         | Additional deployment labels                          | `{}`                    |
| `deploymentAnnotations`                    | Additional deployment annotations                     | `{}`                    |
| `metrics.enabled`                          | Enable application metrics                            | `true`                  |
| `metrics.port`                             | Metrics service type port                             | `9900`                  |
| `metrics.serviceMonitor.enabled`           | Enable serviceMonitor                                 | `true`                  |
| `metrics.serviceMonitor.additionalLabels`  | ServiceMonitor additional labels                      | `{}`                    |

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example:

```bash
$ helm install --name my-release -f values.yaml banzaicloud-stable/cloudinfo
```

> **Tip**: You can use the default [values.yaml](values.yaml)
