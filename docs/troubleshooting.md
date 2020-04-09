# Troubleshooting

This document provides useful information for common errors and their remedies:

## Run without CRD applied

### Command

```zsh
$ operator-sdk run --local --namespace gcp-project-operator
```

### Error Message

```golang
"kubebuilder.source","msg":"if kind is a CRD, it should be installed before calling Start"
panic: no matches for kind "ProjectClaim" in version "gcp.managed.openshift.io/v1alpha1"

goroutine 1 [running]:
main.main()
        FATA[0001] Failed to run operator locally: failed to run operator locally: failed to exec []string{"build/_output/bin/gcp-project-operator-local"}: exit status 2 
```

### Explanation

This error message means you have started running the the operator locally without having the ProjectClaim custom resource definition applied in the Kubernetes cluster.
As a result, kubernetes (and GCP Operator in extend) does not know what a ProjectClaim is.

### Solution

This is expected behavior.
According to the message from KubeBuilder (which is the one Operators SDK is based upon) the CRDs should be installed before starting the operator.
Please type:

```zsh
$ kubectl create -f deploy/crds/gcp_v1alpha1_projectclaim_crd.yaml
```


## Operator does nothing

### Command

```zsh
$ kubectl create -f deploy/crds/gcp_v1alpha1_projectclaim_crd.yaml
```

### Logs

```json
{"level":"info","ts":1583866815.154782,"logger":"kubebuilder.controller","msg":"Starting Controller","controller":"projectreference-controller"}
{"level":"info","ts":1583866815.154809,"logger":"kubebuilder.controller","msg":"Starting Controller","controller":"projectclaim-controller"}
{"level":"info","ts":1583866815.254899,"logger":"kubebuilder.controller","msg":"Starting workers","controller":"projectclaim-controller","worker count":1}
{"level":"info","ts":1583866815.254951,"logger":"kubebuilder.controller","msg":"Starting workers","controller":"projectreference-controller","worker count":1}
```

### Explanation

When the `SharedIndexInformer caches` are synced, then the `workers` are starting as well.
The workers are launching to process the resources.
The operator launches one worker per controller, thus `MaxConcurrentReconciles == 0` for each controller.
The operator is waiting for a ProjectClaim custom resource instance of an object to be created.
Until that happens, there will be no further logs, everything is quiet.

### Solution

This is expected behavior.
Please type:

```zsh
$ kubectl create -f deploy/crds/gcp_v1alpha1_projectclaim_cr.yaml
```


