**GCP-PROJECT-OPERATOR**

The gcp project operator is reponsible for creating projects and service accounts in GCP and storing the credentials in a secret. 

**workflow**:
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

**Requirements**:
- OCM will provide a clusterdeployment with the following
  - An agreed upon secret name in "Spec.PlatformSecrets.GCP.Credentials.Name" now it is using name **_gcp_**
  - Unique name in projectID “Spec.Platform.GCP.ProjectID” 
  - Supported region in “Spec.Platform.GCP.Region”
  - Ssh Key in the clusterdeployment namesapcew with the clusterdeployment  "Spec.SshKey.Name" filled out
-- Service account credentials with permissions to create projects, service accounts, and service account keys  in the operator namespace _**gcp-project-operator**_

**Building code**

`make`

**Pushing Image To quay**

If you have permissions to push to quay.io/razevedo/gcp-project-operator. You can use the following commands to push the latest code 

```
docker build . -f build/Dockerfile -t quay.io/razevedo/gcp-project-operator

docker push quay.io/razevedo/gcp-project-operator
```

**Deploying code**

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

**GCP secret creation**

```bash
export GCPSA_NAME=gcp-account-operator
export GCP_ORG_NAME=osd-management
# ServiceDelivery org ID
export GCP_ROOT_ORG_NAME=240634451310 

gcloud beta iam service-accounts create $GCPSA_NAME \
    --description "$GCPSA_NAME" \
    --display-name "$GCPSA_NAME"

# TODO: this does not work due perm error
gcloud projects add-iam-policy-binding $GCP_ROOT_ORG_NAME \
  --member serviceAccount:$GCPSA_NAME@$GCP_ORG_NAME.iam.gserviceaccount.com \
  --role roles/owner --role roles/resourcemanager.projectCreator \
  --role roles/resourcemanager.folderAdmin

gcloud iam service-accounts keys create key.json \
  --iam-account $GCPSA_NAME@$GCP_ROOT_ORG_NAME.iam.gserviceaccount.com


# billindacount.txt contains billing ID from https://console.cloud.google.com/billing
kubectl create secret generic gcp-project-operator --from-file=key.json=secrets/key.json --from-file=billingaccount=secrets/billingaccount.txt -n gcp-project-operator

```

**TODO**:
-  Creation of project for the cluster
  - Some of this code is mocked out but not tested since we do not have a test org yet.
- Enabling the required APIs.
    - Compute Engine API (`compute.googleapis.com`)
    - Google Cloud APIs (`cloudapis.googleapis.com`)
    - Cloud Resource Manager API (`cloudresourcemanager.googleapis.com`)
    - Google DNS API (`dns.googleapis.com`)
    - Identity and Access Management (IAM) API (`iam.googleapis.com`)
    - IAM Service Account Credentials API (`iamcredentials.googleapis.com`)
    - Service Management API (`servicemanagement.googleapis.com`)
    - Service Usage API (`serviceusage.googleapis.com`)
    - Google Cloud Storage JSON API (`storage-api.googleapis.com`)
    - Cloud Storage (`storage-component.googleapis.com`)
- Setting required quotas
- Enabling Billing
- Adding finalizer to the clusterdeployment
- Cleaning up when clusterdeployment is removed
- Credential Rotation