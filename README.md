# k8s-resourcequota-autoscaler
A Kubernetes addon to autoscale resource quota for Kubernetes namespaces

## Disclaimer
Do not run this in production. This is a proof of concept. No support. No maintenance.

## Problem statement

Kubernetes Namespaces can limit resource usage via ResourceQuota objects for CPU and memory.
Resource Quota objects are currently limited to static values and cannot easily scale, e.g. if there is a horizontal scale on the node level.
This becomes an issue if a cluster is running a number of DaemonSets that scale with the number of nodes.
DaemonSets are often used for services that need to run on a host level, e.g. a log forwarder, runtime security tooling, networking providers etc.

## Solution

ResourceQuota objects will be adjusted through an Operator that uses a CRD to define extended logic.
The extended logic can be implemented as golang templates using [sprig](https://github.com/Masterminds/sprig) and currently only supports the number of nodes in a cluster.

### Example

Existing static ResourceQuota objects:

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-resources
spec:
  hard:
    requests.cpu: "1"
    requests.memory: 1Gi
    limits.cpu: "2"
    limits.memory: 2Gi
```

A ManagedResourceQuota object will be defined in the namespace and detected by the operator:

```yaml
apiVersion: k8s-resourcequota-autoscaler.m21r.de/v1beta1
kind: ManagedResourceQuota
metadata:
  name: compute-resources
spec:
  template:
    hard:
      requests.cpu: "{{ mul .Nodes 1 }}"
      requests.memory: "{{ max (mul .Nodes 1) 24 }}Gi"
      limits.cpu: "{{ mul .Nodes 2 }}"
      limits.memory: "{{ mul .Nodes 2 }}Gi"
```

The operator will watch the nodes getting added or removed and generates a ResourceQuota object that scales per Node added/removed from the cluster.

For 16 nodes it will render:
```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-resources
spec:
  hard:
    requests.cpu: "16"
    requests.memory: 24Gi
    limits.cpu: "32"
    limits.memory: 32Gi
```
