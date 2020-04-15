# Debugging

Some useful commands:

### ProjectClaim

The `ProjectClaim` is getting deployed onto a namespace defined at the Resource.

```kube
name="example-projectclaim"
namespace="example-clusternamespace"

kubectl -n $namespace get projectclaim $name
NAME                   STATE   GCPPROJECTID           AGE
example-projectclaim   Ready   example-GCPprojectID   2m33s
```

If everything worked as expected the **STATE** should be `READY`.
Some useful information you can extract by passing the `-o yaml` is:

```
kubectl -n $namespace get projectclaim $name -o yaml
```

* The `availabilitiZones`:

```yaml
  availabilityZones:
  - us-east1-b
  - us-east1-c
  - us-east1-d
```

* The `gcpCredentialSecret`:

```yaml
  gcpCredentialSecret:
    name: gcp-secret
    namespace: example-clusternamespace
```

Having found the namespace, you can gather even more information:

```
kubectl -n example-clusternamespace get secrets gcp-secret -o yaml
```

If the `ProjectClaim` is not in READY state but in **PendingProject** it means the operator is still working on creating a project in GCP.

```kube
kubectl -n example-clusternamespace get projectclaim example-projectclaim
NAME                   STATE            GCPPROJECTID           AGE
example-projectclaim   PendingProject   example-GCPprojectID   82s
```

### ProjectReference

It gets created automatically by the Operator. You can find it by two ways:

1. Looking at the `projectReferenceCRLink` of the `ProjectClaim`. For example if at `ProjectClaim` spec has this spec:

```yaml
  projectReferenceCRLink:
    name: example-clusternamespace-example-projectclaim
    namespace: gcp-project-operator
```

then you can query for the `ProjectReference` like this:

```
kubectl get projectreferences example-clusternamespace-example-projectclaim -o yaml
```

2. Looking at all the `ProjectReferences` under the `gcp-project-operator` namespace.

```kube
$ kubectl -n gcp-project-operator get projectreferences
NAME                                            STATE   CLAIMNAME              CLAIMNAMESPACE             AGE
example-clusternamespace-example-projectclaim   Ready   example-projectclaim   example-clusternamespace   20m
```

Here you can find the real name of the GCP Project created at Google side, by adding the `-o yaml` at the end of the command:

```
kubectl -n gcp-project-operator get projectreferences example-clusternamespace-example-projectclaim -o yaml
```

* The `gcpProjectID`

```yaml
gcpProjectID: o-a68db2ad
```

* The `projectClaimCRLink`:

```yaml
  projectClaimCRLink:
    name: example-projectclaim
    namespace: example-clusternamespace
```

Notice that both `ProjectClaim` and `ProjectReference` are cross-referencing each other. That means if you know one of them, you can easily find the other.
