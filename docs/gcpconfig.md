# GCP Configuration

The GCP Project Operator expects some pre-existing configuration into your Kubernetes cluster that is related to your account in Google GCP cloud.

> Note: Unless you're running this against your very own personal GCP org account, someone likely already has this stuff prepared for you in your company/team. Ask around.

### Configmap

The Operator needs to be aware of your Google Billing account. If you don't have one, please [create](https://cloud.google.com/billing/docs/how-to/manage-billing-account) one and note its number down. For parent folder you can use any folder you like. If you don't have one, feel free to [create](https://cloud.google.com/resource-manager/docs/creating-managing-folders) one.

Forward this information into your Kubernetes cluster by creating a `configmap`:

```zsh
$ export PARENTFOLDERID=your folder’s ID goes here
$ export BILLINGACCOUNT=your billing account number goes here
$ kubectl create -n gcp-project-operator configmap gcp-project-operator --from-literal parentFolderID=$PARENTFOLDERID --from-literal billingAccount=$BILLINGACCOUNT
```

### Secret

The Operator needs a ServiceAccount to authenticate its client against Google GCP.
Find your Google GCP ServiceAccount by going [here](https://console.cloud.google.com/projectselector2/iam-admin/serviceaccounts?supportedpurview=project) or [create](https://cloud.google.com/iam/docs/creating-managing-service-accounts) one. This downloads a json file with your key.

Forward this information into your Kubernetes cluster by creating a `secret`:

```zsh
$ kubectl create -n gcp-project-operator secret generic gcp-project-operator-credentials --from-file=key.json=your-file.json
```

Now your kubernetes cluster has everything it needs to build a client and communicate with Google GCP using your billing account and a ServiceAccount that has the permissions to create projects and other resources (such as virtual-machines).