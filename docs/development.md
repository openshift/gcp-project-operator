# Development

## Development Environment Setup

* A recent Go distribution (>=1.14) with enabled Go modules

```
export OPERATOR_NAME=gcp-project-operator
export GO111MODULE=on
```

* Code-Ready Containers

Red Hat CodeReady Containers brings a minimal OpenShift 4 cluster to your local computer.
This cluster provides a minimal environment for development and testing purposes.
It’s mainly targeted at running on developers' desktops.
Installation and configuration of CRC is beyond the scope of this document.
Alternatively you can use `minikube` instead.

* kubectl client (>= version with your kubernetes server)

Make sure kubectl is pointing to your kubernetes cluster you want to test the Operator against.

* Operators SDK

The Operator is being developed based on the [Operators SDK](https://github.com/operator-framework/operator-sdk).
Make sure you have it installed correctly.

* Docker or Podman

## Run the operator

The operator can run either:

* locally - without building a container and pushing it to your kubernetes cluster. This is the most convenient way.
* remotely - building a container with podman/docker and pushing to a registry and installing to your k8s cluster along with some RBAC configuration

No matter which option you choose, before running the Operator you have to create the following Custom Resource Definitions on the cluster:

```shell
kubectl create -f deploy/crds/gcp_v1alpha1_projectclaim_crd.yaml
kubectl create -f deploy/crds/gcp_v1alpha1_projectreference_crd.yaml
```

### Locally

Make sure you have the [operator-sdk](https://github.com/operator-framework/operator-sdk/releases) binary in your `$PATH` and run it locally:

```shell
$ operator-sdk run --local --namespace gcp-project-operator
```

You will see some initialization logs.
The Operator will remain _idle_ after that, waiting for `ProjectClaim` resources to be present in the cluster.

You can change the verbosity of the logs, by passing the `--zap-level` flag as part of the `--operator-flags`:

```shell
run --local --namespace gcp-project-operator --operator-flags --zap-level=99
```

* Level 1: ProjectReference
* Level 2: ProjectClaim
* Level 3: gcpclient
* Level 4: operator-sdk framrwork logs (golang version, etc)

### Remotely

#### Pushing Image a container registry

Push the image to your container registry of your choice. For example:

```shell
username="razevedo"
podman build . -f build/Dockerfile -t "quay.io/$username/gcp-project-operator"
podman push "quay.io/$username/gcp-project-operator"
```

#### Deploying code

Update deploy/operator.yaml with image you would like deployed.

```kube
kubectl apply -f deploy/cluster_role_binding.yaml
kubectl apply -f deploy/cluster_role.yaml
kubectl apply -f deploy/service_account.yaml
kubectl apply -f deploy/operator.yaml
```

If you need to update to lastest image pushed to quay repo.

```kube
kubectl scale deployment gcp-project-operator -n gcp-project-operator --replicas=0
kubectl scale deployment gcp-project-operator -n gcp-project-operator --replicas=1
```

Otherwise, you can directly upload the image to your kubernetes cluster by hand

```shell
# Export the image locally
docker save $image-name > image-name.tar

# Copy the image to your CRC/Minikube remote cluster. Copy one of the following:
scp image-name.tar core@`minikube ip`:   # minikube
scp image-name.tar core@`crc ip`:        # CRC

# SSH into the k8s node and load the image
minikube ssh # For minikube

# SSH into k8s node (for CRC devenv)
export CRCIP=$(crc ip)
alias sshcrc="ssh -o ConnectionAttempts=3 -o ConnectTimeout=10 -o ControlMaster=no -o ControlPath=none -o LogLevel=quiet -o PasswordAuthentication=no -o ServerAliveInterval=60 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null core@192.168.64.2 -o IdentitiesOnly=yes -i /Users/drpaneas/.crc/machines/crc/id_rsa -p 22"

# Load the image to CRI-O
cat image-name.tar | podman load
```

## Configure GCP Cloud

The Operator requires some GCP–related configs to be present on your cluster.

Follow [these instructions](./gcpconfig.md) to create both the `ConfigMap` and the `secret` required.

## Use the Operator

No matter if the operator is running locally or remotely, you can request it to create Google GCP Project for you. So, if you want to actually test the operator, create a `ProjectClaim` resource or apply the example:

```
kubectl create -f deploy/crds/gcp_v1alpha1_projectclaim_cr.yaml
```

This will trigger the Operator to start reconciling.
