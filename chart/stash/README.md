# Stash
[Stash](https://github.com/appscode/stash) provides support to backup your Kubernetes Volumes
## TL;DR;

```bash
$ helm install chart/stash
```

## Introduction

This chart bootstraps a [Stash controller](https://github.com/appscode/stash) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.5+ 

## Installing the Chart
To install the chart with the release name `my-release`:
```bash
$ helm install --name my-release chart/stash
```
The command deploys Stash Controller on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release`:

```bash
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following tables lists the configurable parameters of the Stash chart and their default values.


| Parameter                  | Description                | Default                                                    |
| -----------------------    | ----------------------     | ------------------- |
| `image`                    |  Container image to run    | `appscode/stash`   |
| `tag`                 |  Image tag of container    | `latest`            |
