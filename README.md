
   * [Info](#info)
      * [Workflow - ProjectClaim](#workflow---projectclaim)
         * [Example CR](#example-cr)
      * [Workflow - ClusterDeployment (deprecated)](#workflow---clusterdeployment-deprecated)
      * [Requirements](#requirements)
   * [Deployment](#deployment)
      * [Building](#building)
      * [Local Dev](#local-dev)
         * [Prerequisites](#prerequisites)
         * [Start operator locally](#start-operator-locally)
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
    * The field `spec.gcpProjectID` will be filled with the ID of the GCP project
    * A list of available zones in the input region is set in `spec.availabilityZones`.
4. When a `ProjectClaim` is removed, the GCP project and service accounts are deleted (WIP)
5. The operator removes the finalizer from the `ProjectClaim` (WIP)

### Example Input Custom Resource

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

# Deployment

## Building

Just run `make`.

## Local Dev

### Prerequisites

* Typically you'll want to use [CRC](https://github.com/code-ready/crc/), though it's fine if you're running OpenShift another way.
* You need to have [the operator-sdk binary](https://github.com/operator-framework/operator-sdk/releases) in your `$PATH`.

### Start operator locally

```
oc new-project gcp-project-operator
oc apply -f deploy/crds/gcp_v1alpha1_projectclaim_crd.yaml
oc apply -f deploy/crds/gcp_v1alpha1_projectreference_crd.yaml

operator-sdk run --local --namespace gcp-project-operator
```

If everything went ok, you should see some startup logs from the operator in your terminal window.

There are example CRs in `deploy/crds/` you might want to use to see how the operator reacts to their presence (and absence if you delete them).

### Running tests

You can run the tests using `make gotest` or `go test ./...`

## Configuration

For the operator to interact with GCP properly, it needs a bit of configuration first.

Note: unless you're running this against your very own GCP org, **someone likely already has this stuff prepared for you.**
**Ask around.**

### Auth Secret

1. Create a gcp service account with appropriate permissions to an empty folder ("(Project) Owner" and "Project Creator" should suffice).
2. Generate keys for the service account and download them.
3. Run `oc create -n gcp-project-operator secret generic gcp-project-operator-credentials --from-file=key.json=YOUR-KEYS-FILE-NAME.json`

### Configmap

The controller expects to find a `ConfigMap` with the name `gcp-project-operator` inside the `gcp-project-operator` namespace.
It will parse it and verify its contents, expecting to extract the values of two specific fields that should be already populated by you:

* `billingAccount`
* `parentFolderID`

To fulfill this prerequisite, please type:

```bash
export PARENTFOLDERID="123456789123"         # Google Cloud organization Parent Folder ID
export BILLINGACCOUNT="123456-ABCDEF-123456" # Google billing ID from https://console.cloud.google.com/billing

kubectl create -n gcp-project-operator configmap gcp-project-operator --from-literal parentFolderID=$PARENTFOLDERID --from-literal billingAccount=$BILLINGACCOUNT
```
