
   * [Info](#info)
      * [Workflow ProjectClaim](#workflow---pjrojectclaim)
      * [Workflow ClusterDeployment (deprecated)](#workflow---clusterdeployment-deprecated)
      * [Requirements](#requirements)
   * [Deployment](#deployment)
      * [Building](#building)
      * [Local Dev](#local-dev)
         * [Prerequisites](#prerequisites)
         * [Load Hive CRDS](#load-hive-crds)
         * [Start operator locally](#start-operator-locally)
      * [Remote](#remote)
         * [Pushing Image To quay](#pushing-image-to-quay)
         * [Deploying code](#deploying-code)
      * [Configuration](#configuration)
         * [Auth Secret](#auth-secret)
         * [Configmap](#configmap)

# Info

The gcp project operator is reponsible for creating projects and service accounts in GCP and storing the credentials in a secret.

## Workflow - ProjectClaim

1. The operator watches all namespaces for `ProjectClaim` resources
2. When a `ProjectClaim` is found (see example below) the operator triggers the creation of a project in GCP
3. After successful project creation
    * the field `State` will be set to Ready
    * A secret is created in the cluster namespace, as defined in the `ProjectClaim`
    * The field `spec.gcpProjectID` will be filled with the ID of the GCP project (WIP)
4. When a `ProjectClaim` is removed, the GCP project and service accounts are deleted (WIP)
5. The operator removes the finalizer from the `ProjectClaim` (WIP)

### Example

```yaml
apiVersion: gcp.managed.openshift.io/v1alpha1
kind: ProjectClaim
metadata:
  name: example-projectclaim
  namespace: example-clusternamespace
spec:
  region: us-east1
  gcpCredentialSecret:
    name: gcp-secret
    namespace: example-clusternamespace
  legalEntity:
    name: example-legal-entity
    id: example-legal-entity-id
```

## Workflow - ClusterDeployment (deprecated)

- The operator would watch clusterdeployments in all namespaces.
- Operator would check that the clusterdeployment’s labels “api.openshift.com/managed = true” and “hive.openshift.io/cluster-platform = gcp"
  - If both labels are as expected and clusterdeployment field “Spec.Installed = false”
    - The operator will create a project for the cluster from the name provided in
“Spec.Platform.GCP.ProjectID” in the region provided by "Spec.Platform.GCP.Region"
    - The operator will then enable the required APIs.
    - The operator will set quotas as required
    - The operator will then create a service account for the operator and key
    - The operator will create a secret called gcp
    - The operator will add a finalizer to the clusterdeployment
  - If both labels are as expected and the clusterdeployment field “Spec.Installed = true” and the clusterdeployment is attempted to be deleted.
    - The operator will delete the service account and project
    - The operator will remove the finalizer

## Requirements

- OCM will provide a clusterdeployment with the following
  - An agreed upon secret name in "Spec.PlatformSecrets.GCP.Credentials.Name" now it is using name **_gcp_**
  - Unique name in projectID “Spec.Platform.GCP.ProjectID”
  - Supported region in “Spec.Platform.GCP.Region”
  - Ssh Key in the clusterdeployment namesapcew with the clusterdeployment  "Spec.SshKey.Name" filled out
-- Service account credentials with permissions to create projects, service accounts, and service account keys  in the operator namespace _**gcp-project-operator**_

# Deployment

## Building

Just run `make`.

## Local Dev

### Prerequisites

* Typically you'll want to use [CRC](https://github.com/code-ready/crc/), though it's fine if you're running OpenShift another way.
* You need to have [the operator-sdk binary](https://github.com/operator-framework/operator-sdk/releases) in your `$PATH`.

### Load Hive CRDS

For gcp-project-operator to work, the cluster needs to have CRDs from [Hive](https://github.com/openshift/hive) present in the system.

Note: the `git clone` below *isn't* using the `master` branch.

```
git clone --branch v1alpha1 https://github.com/openshift/hive
for crd in hive/config/crds/hive_v1alpha1_*.yaml; do (set -x; oc apply -f $crd); done
```

### Start operator locally

```
oc new-project gcp-project-operator
oc apply -f deploy/crds/gcp_v1alpha1_projectclaim_crd.yaml
oc apply -f deploy/crds/gcp_v1alpha1_projectreference_crd.yaml

operator-sdk run --local --namespace gcp-project-operator
```

If everything went ok, you should see some startup logs from the operator in your terminal window.

There are example CRs in `deploy/crds/` you might want to use to see how the operator reacts to their presence (and absence if you delete them).


## Remote

### Pushing Image To quay

If you have permissions to push to quay.io/razevedo/gcp-project-operator. You can use the following commands to push the latest code

```
podman build . -f build/Dockerfile -t quay.io/razevedo/gcp-project-operator

podman push quay.io/razevedo/gcp-project-operator
```

### Deploying code

Currently it is being deployed using image 'quay.io/razevedo/gcp-project-operator' Update deploy/operator.yaml with image you would like deployed.

```
oc apply -f deploy/cluster_role_binding.yaml
oc apply -f deploy/cluster_role.yaml
oc apply -f deploy/service_account.yaml
oc apply -f deploy/operator.yaml
```

If you need to update to lastest image pushed to quay repo

```
oc scale deployment gcp-project-operator -n gcp-project-operator --replicas=0

oc scale deployment gcp-project-operator -n gcp-project-operator --replicas=1
```

## Configuration

For the operator to interact with GCP properly, it needs a bit of configuration first.

Note: unless you're running this against your very own GCP org, **someone likely already has this stuff prepared for you.**
**Ask around.**

### Auth Secret

TODO: short info on how to set up a service account with the right perms.

### Configmap

```bash
export ORGPARENTFOLDERID="12345678" # Google Cloud organization Parent Folder ID
export BILLINGACCOUNT="" # obtain billing ID from https://console.cloud.google.com/billing

kubectl create configmap gcp-project-operator --from-literal orgParentFolderID=$ORGPARENTFOLDERID --from-literal billingaccount=$BILLINGACCOUNT -n gcp-project-operator

```
