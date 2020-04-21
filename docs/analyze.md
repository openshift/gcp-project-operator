## Claim a new GCP Project

Kubernetes does not know what a `ProjectClaim` is.
To teach it, we need to create new Custom Resource Definition that is going to be the definition of the ProjectClaim's object specification.
Our operator is extending the Kubernetes API by aggregating adding new _REST_ instructions related to this new `ProjectClaim` type.
This allows us to create instances (called Custom Resources) of that type and put them into action.

### Create the ProjectClaim object

Regardless of how complex API extension might sound like, as a user you only have to create a `ProjectClaim` CRD.
You do not need to understand how API aggregation works behind the scenes.
Please type:

```zsh
$ kubectl create -f deploy/crds/gcp_v1alpha1_projectclaim_crd.yaml
```

### Create an instance of ProjectClaim type

Now that Kubernetes knows what a project claim is, it is about time to claim for project.
We can do that by creating a new `ProjectClaim` instance, configured with value of our own preference.

```yaml
apiVersion: gcp.managed.openshift.io/v1alpha1
kind: ProjectClaim
```

Change the name of this instance and the namespace it is going to be stored:

```yaml
metadata:
  name: example-projectclaim
  namespace: example-clusternamespace
```

Define the expected state by changing the values for your preference:

```yaml
spec:
  region: us-east1
  gcpProjectID: example-GCPprojectID
  projectReferenceCRLink:
    name: example-projectreference
    namespace: gcp-project-operator
  legalEntity:
    name: example-entity
    id: example-id
```

For example, you can apply our example by typing:

```zsh
$ kubectl create -f deploy/crds/gcp_v1alpha1_projectclaim_cr.yaml
```

After running this, verify everything is applied into Kubernetes:

```kubectl
$ kubectl get projectclaim --all-namespaces
NAMESPACE                  NAME                   STATE   GCPPROJECTID           AGE
example-clusternamespace   example-projectclaim           example-GCPprojectID   1m25s

$ kubectl get -n example-clusternamespace projectclaim example-projectclaim
NAME                   STATE   GCPPROJECTID           AGE
example-projectclaim           example-GCPprojectID   1m
```

A `ProjectClaim` instance named `example-projectclaim` has been created in our cluster inside the `example-clusternamespace` namespace.
We have successfully extended the Kubernetes API by adding the **Group** `gcp.managed.openshift.io`, the **Version** `v1alpha1` and a new **Kind** `projectclaims` with a **Resource** `example-projectclaim`.
To verify this, you can access directly the `selfLink` like this:

```
$ kubectl get --raw /apis/gcp.managed.openshift.io/v1alpha1/namespaces/example-clusternamespace/projectclaims/example-projectclaim
```

The API is not written in `yaml` but in `.go` files in the form a `struct` data type.
There are two files you need to be aware of:

* The ProjectClaim Go structs: `./pkg/apis/gcp/v1alpha1/projectclaim_types.go`
* The reconciler of the projectclaim controller: `./pkg/controller/projectclaim/projectclaim_controller.go`

## Kubernetes receives your request

The first log message says:

```json
{"logger":"controller_projectclaim","msg":"Reconciling ProjectClaim","Request.Namespace":"example-clusternamespace","Request.Name":"example-projectclaim"}
```

This means the `Reconcile()` method has been triggered and it's time for the `projectclaim_controller.go` to act upon the `example-projectclaim` projectclaim instance.

```go
func (r *ReconcileProjectClaim) Reconcile(request reconcile.Request) (reconcile.Result, error) {
    // This is the code
    // that is happening now
}
```

### Fetch the ProrectClaim instance

```go
instance := &gcpv1alpha1.ProjectClaim{}
err := r.client.Get(context.TODO(), request.NamespacedName, instance)
if err != nil {
    // If failed to found the example-projectclaim it stops and not reconcile again for this resource (like a 'break;' statement)
    // If failed for another reason it will restart the Reconcile function (like a 'continue;' statement)
}
```

If there is no `err` we assume the `example-projectclaim` instance has been fetched successfully.

What is happening first is a series of checks ensuring certain things are setup correctly as they should do. Such as `IsProjectClaimDeletion()`, `EnsureProjectClaimInitialized()` and `EnsureProjectClaimState()`. That is enough to make put `ProjectClaim` into action.

### Create the ProjectReference instance

Things are moving forward with the `EnsureProjectReferenceExists()` which the first time called creates the `ProjectReference` by calling `c.client.Create(context.TODO(), c.projectReference)`.

In contrast to ProjectClaim custom resource, the ProjectReference instance is created by the operator instead of the user.
The setup is implemented in a secondary file, called `customresourceadapter.go` that is still part of the projectclaim controller.
Since there is no user to apply a `*.yaml` file, the controller creates a `*gcpv1alpha1.ProjectReference` instance with the following fields and values:

* name: `<namespace of the projectclaim instance>-<name of the projectclaim instance>`
* namespace: `gcp-project-operator` that is hardcoded in `projectreference_types.go`
* ProjectClaimCRLink: {Name: `name of the projectclaim instance>`, Namespace: `namespace of the projectclaim instance`}
* LegalEntity: the same ones with the projectclaim instance
* ProjectID: it is empty for the moment (see: `omitempty` at `projectreference_types.go`)

So from now on, the `adapter` variable is a pointer to `*CustomResourceAdapter` so we can have shared access to the data.

This instance is created at the code side of things. It is not yet created on the cluster! For example it is like we have created `file.yaml` but we haven't run `oc create -f file.yaml` yet.
So the next step is to actually create and save the `example-clusternamespace-example-projectclaim` in the Kubernetes cluster by invoking `c.client.Create(context.TODO(), c.projectReference)`.
The result is returned back to the `err = adapter.EnsureProjectReferenceExists()` and there is not error.
At this point if the creation is successful our new `example-clusternamespace-example-projectclaim` will be created.

```kubectl
$ oc -n gcp-project-operator get projectreferences
NAME                                            STATE   CLAIMNAME              CLAIMNAMESPACE             AGE
example-clusternamespace-example-projectclaim           example-projectclaim   example-clusternamespace   54m
```

A `ProjectReference` instance named `example-clusternamespace-example-projectclaim` has been created in our cluster inside the `gcp-project-operator`.
The operator has successfully extended the Kubernetes API by adding the **Group** `gcp.managed.openshift.io`, the **Version** `v1alpha1` and a new **Kind** `projectreferences` with a **Resource** `example-clusternamespace-example-projectclaim`.
To verify this, you can access directly the `selfLink` like this:

```
kubectl get --raw /apis/gcp.managed.openshift.io/v1alpha1/namespaces/gcp-project-operator/projectreferences/example-clusternamespace-example-projectclaim
```

There are two files you need to be aware of:

* The projectReference Go structs: `gcp-project-operator/pkg/apis/gcp/v1alpha1/projectreference_types.go`
* The reconciler of the projectReference controller: `gcp-project-operator/pkg/controller/projectreference/projectreference_controller.go`

### Cross-reference between the Claim and Reference

The `ProjectClaim` creates the `ProjectReference` and these two are aware of each of other via `ProjectReferenceCRLink`.
That means the `ProjectClaim` is the _parent_ object and if it gets deleted it will also remove its children (aka ProjectReference) as well.

The code populates a field for the `example-projectclaim` ProjectClaim instance, that is the `projectReferenceCRLink`.
In that way both instances (projectclaim and projectreference) can cross-reference each other via this `{project,claim}ReferenceCRLink` respectively.
The operator couldn't create the `projectReferenceCRLink` before, because there was no `ProjectReference` object created.
So that field was empty (see: `ProjectReferenceCRLink NamespacedName json:"projectReferenceCRLink,omitempty"`) so far that means the condition `c.projectClaim.Spec.ProjectReferenceCRLink == expectedLink` will be `false`.
The code assigns the correct CR Link (see: `c.projectClaim.Spec.ProjectReferenceCRLink = expectedLink`) in the Go code, and then it updates the actual object in the Kubernetes cluster using the `client.Update` function (see: `c.client.Update(context.TODO(), c.projectClaim)`).
We can verify that:

```yaml
$ oc get -n example-clusternamespace projectclaim example-projectclaim -o yaml | grep -A 2 projectReferenceCRLink

  projectReferenceCRLink:
    name: example-clusternamespace-example-projectclaim
    namespace: gcp-project-operator
```

### Add a finalizer to ProjectClaim and wait

Before the end, `ProjectClaim` is doing is to add a finalizer to itself calling the `EnsureFinalizer()` function.
At first it checks if the finalizer already exists and then, if it doesn't, it adds it.
The addition of the finalizer is very important because it makes sure that the `ProjectClaim` will never get deleted while the children `ProjectReference` instance exists.

The last thing happening is setting the state of the `ProjectClaim`.
A _READY_ status it would be that our goal (the GCP Projection creation in Google cloud) has be completed.
As a result, while this is still on going, the state is set at `Pending` mode by calling `EnsureProjectClaimState(gcpv1alpha1.ClaimStatusPendingProject)`.

### Contact Google GCP using a key

Earlier, when `ProjectClaim` called `EnsureProjectReferenceExists()` a `ProjectReference` instance created and got picked-up by the Controller.

One of the first things the `ProjectReference` does is to setup a client by calling `getGcpClient()`.
The `ProjectReference` Controller expects to find a secret called `gcp-project-operator-credentials` inside `gcp-project-operator` namespace that contains the `ServiceAccount` and the `key` for accessing Google Cloud.

The secret looks like this: `kubectl -n gcp-project-operator get secret gcp-project-operator -o json`

```json
{
    "apiVersion": "v1",
    "data": {
        "key.json": "HERE YOU WILL FIND YOUR KEY"
      },
    "kind": "Secret",
    "metadata": {
        "creationTimestamp": "2020-03-11T14:26:49Z",
        "name": "gcp-project-operator-credentials",
        "namespace": "gcp-project-operator",
        "resourceVersion": "3650367",
        "selfLink": "/api/v1/namespaces/gcp-project-operator/secrets/gcp-project-operator",
        "uid": "8b2da99a-6504-466d-a9ea-6f02b1cc6ce5"
    },
    "type": "Opaque"
}
```

The code that _parses_ this secret can be found at `util.go` at the function `GetGCPCredentialsFromSecret()`

* If you don't have a key, you will see: `clusterdeployment.getGCPCredentialsFromSecret.Get Secret \"gcp-project-operator\" not found","`
* If you have an invalid secret, you will see: `"GCP credentials secret gcp-project-operator did not contain key {osServiceAccount,key}.json"`

In case you don't have a key, you need to configure one. To do that, type:

```kube
oc create -n gcp-project-operator secret generic gcp-project-operator-credentials --from-file=key.json=your-file.json
```

> Note: If you don't have a `key.json` follow [these](https://cloud.google.com/docs/authentication/getting-started) instructions.

So, up to this point the operator has the credentials to contact the Google GCP and exchange information with it.
We don't know though if those are valid or not before we actually try to communicate.
To communicate, we need a client. Let's build one in the next section.

#### Create the client for Google GCP

For the creation of a valid `gcpClient` instance we need to make sure that the credentials are valid for Google cloud.
To do that we use a function `CredentialsFromJSON` defined at `https://github.com/golang/oauth2`.
As soon as we get authenticated, we are creating all the required Google GCP environment using the [google-api-go-client](https://github.com/googleapis/google-api-go-client).

See the `/pkg/gpclient/client.go` for the creation of the client.
This is done via [Factory Pattern](https://www.sohamkamani.com/golang/2018-06-20-golang-factory-patterns/) using the `NewClient()` method:

```go
// NewClient creates our client wrapper object for interacting with GCP.
func NewClient(projectName string, authJSON []byte) (Client, error) {
        // code
}
```

#### Generate the GCP Project ID

One of the most important things, is the generation of the GCP Project identification which has to be unique.
This is getting done when `updateProjectID()` is called (which uses `GenerateProjectID()` internally).

#### Verify if the region is supported

Also, not all the regions are available to deploy an OpenShift cluster as they have limited hardware quota.
As a result, the Operator makes sure the chosen region propagated by the `ProjectClaim` complies with the requirements by calling `checkRequirements()`.

#### Add a finalizer to ProjectReference

Similarly with `ProjectClaim`, we add a finalizer for `ProjectReference` as well by calling the `EnsureFinalizerAdded()`.
The purpose is to make sure that `ProjectReference` will not get deleted while the actual GCP Project in Google cloud exists.

### Create the GCP Project

Having an established communication with Google GCP, the client is now creating the actual GCP Project at Google's infrastructure and then it configures it.

#### Get the configmap

The procedure starts by calling `EnsureProjectConfigured()`.
This is expecting to find a `configmap` with the name `gcp-project-operator` inside `gcp-project-operator` namespace.
This has to be created by the user. If you haven't done it already, then type:

```kubectl
export PARENTFOLDERID=your folder’s ID goes here
export BILLINGACCOUNT=your billing ID goes here

oc create -n gcp-project-operator configmap gcp-project-operator --from-literal parentFolderID=$PARENTFOLDERID --from-literal billingAccount=$BILLINGACCOUNT
```

The Operator should be able to read the ConfigMap by calling `getConfigMap()`.

#### Create the actual GCP Project

After having a client configured and authenticated with Google GCP, the Operator now creates a GCP Project by calling `createProject()`.

#### Enable the APIs

To bootstrap an OpenShift cluster in Google GCP the Operator needs to enable the following required APIs by calling `configureAPIS()`:

* [Service Usage](https://cloud.google.com/service-usage/docs/reference/rest)
* [Cloud Resource Manager](https://cloud.google.com/resource-manager/reference/rest)
* [Storage Component](https://cloud.google.com/storage/docs/json_api/v1)
* [Storage](https://cloud.google.com/storage/docs/json_api)
* [DNS](https://cloud.google.com/dns/docs/reference/v1)
* [IAM](https://cloud.google.com/iam/docs/reference/rest)
* [Compute](https://cloud.google.com/compute/docs/reference/rest/v1/)
* [Cloud](https://cloud.google.com/apis/docs/overview)
* [IAM Credentials](https://cloud.google.com/iam/docs/reference/credentials/rest)
* [Service Management](https://cloud.google.com/service-infrastructure/docs/service-management/reference/rest)

Next, the Operator is creating a Service Account and configures the Service Account Policies.
It creates the Credentials, such as the Service AccountKey and the `gcp-secret` in the namespace `example-clusternamespace`.
If everything goes well, it sets the status of ProjectReference to _READY_.

#### Configure a Service Account

The next step in the chain is creating a `ServiceAccount` (not in Kubernetes) in Google GCP.
A service account in Google GCP is a special kind of account used by an application or a virtual machine (VM) instance, _not a person_. Applications, like this Operator, use service accounts to make authorized API calls.
The creation of this ServiceAccount will be needed later by another Operator which will need to create virtual machine in order to install OpenShift.
As a result, this ServiceAcount gets created now with the permissions to create projects in the operator namespace. This is done by calling `configureServiceAccount()`.

You can find the service account by accessing the [Service Accounts](https://console.cloud.google.com/projectselector2/iam-admin/serviceaccounts?supportedpurview=project) page and clicking at the project Id (which has been created by the Operator).
The Operator creates a Service Account with an e-mail `osd-managed-admin@<PROJECTID>.iam.gserviceaccount.com` and a hardcoded name `osd-managed-admin`.

#### Pass the key to the ServiceAccount

For the ServiceAccount to be able to do real tasks in Google GCP, it needs to have the correct permissions to do it.
The creation of the key for the ServiceAccount is done by calling `createCredentials()` which uses `gcpClient.CreateServiceAccountKey()`.
After the successful creation of the key in Google GCP, the Operator is copying it over to the Kubernetes as well calling the `NewGCPSecretCRV2()`.
The new secret is called `gcp-secret` and can be found inside the `example-clusternamespace` namespace (it's the one where `ProjectClaim` lives).

The key can be found by issuing: `oc -n example-clusternamespace get secrets gcp-secret -o yaml`.
If you _decode_ with _base64_ the `data` section, it will reveal the private key that is connected to the Google GCP Service Account for the GCP Project.

### Setting status to READY

After the successful completion of all of these steps the status of `ProjectReference` sets to _READY_. At the same time, the `ProjectClaim` controller sees that and sets the `ProjectClaim` status to _READY_ as well. That is the last task done by the Operator. After this, it remains _idle_.