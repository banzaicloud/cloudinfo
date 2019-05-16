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

| Parameter                                 | Description                                | Default                 |
|-------------------------------------------|--------------------------------------------|-------------------------|
| `frontend.image.repository`               | Frontend container image repository        | `banzaicloud/cloudinfo` |
| `frontend.image.tag`                      | Frontend container image tag               | `0.6.0`                 |
| `frontend.image.pullPolicy`               | Frontend container pull policy             | `IfNotPresent`          |
| `frontend.service.type`                   | Frontend Kubernetes service type to use    | `ClusterIP`             |
| `frontend.service.port`                   | Frontend Kubernetes service port to use    | `80`                    |
| `frontend.deployment.labels`              | Additional frontend deployment labels      | `{}`                    |
| `frontend.deployment.annotations`         | Additional frontend deployment annotations | `{}`                    |
| `scraper.image.repository`                | Scraper container image repository         | `banzaicloud/cloudinfo` |
| `scraper.image.tag`                       | Scraper container image tag                | `0.6.0`                 |
| `scraper.image.pullPolicy`                | Scraper container pull policy              | `IfNotPresent`          |
| `scraper.deployment.labels`               | Additional scraper deployment labels       | `{}`                    |
| `scraper.deployment.annotations`          | Additional scraper deployment annotations  | `{}`                    |
| `app.logLevel`                            | Log level                                  | `info`                  |
| `app.basePath`                            | Application base path                      | `/`                     |
| `app.vault`                               | Vault configuration                        | `{}`                    |
| `providers.[provider].enabled`            | Enable a provider                          | `false`                 |
| `providers.[provider].*`                  | Provider specific configuration            | `{}`                    |
| `metrics.enabled`                         | Enable application metrics                 | `true`                  |
| `metrics.port`                            | Metrics service type port                  | `9900`                  |
| `metrics.serviceMonitor.enabled`          | Enable serviceMonitor                      | `true`                  |
| `metrics.serviceMonitor.additionalLabels` | ServiceMonitor additional labels           | `{}`                    |

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example:

```bash
$ helm install --name my-release -f values.yaml banzaicloud-stable/cloudinfo
```

> **Tip**: You can use the default [values.yaml](values.yaml)


### Configuring a provider

```yaml
providers:
    providerName:
        enabled: true
        
        # Provider specific configuration
        # accessKey: ""
        # secretKey: ""
```

### Redis configuration

The application requires a Redis store at the moment.
The chart can install it for you:

```yaml
store:
    redis:
        enabled: true
        
redis:
    enabled: true
```

Alternatively, you can provide connection information for your own Redis instance:

```yaml
store:
    redis:
        enabled: true
        host: "my-redis"
        port: 6379
```

**Note:** The application does not support Redis application at the moment.


### Getting secrets from Vault

The application supports getting cloud provider secrets from [Vault](http://vaultproject.io/):

```yaml
app:
    vault:
        address: "http://my-vault:8200"
        token: "my-token"
        secretPath: "secret/data/app/cloudinfo"

providers:
    providerName:
        enabled: true
```
