# gcp-service-account-maintenance

## Configuration

Create a local config file

```
# cp config.source.sample config.source
```

Edit your local config file, internal comments should explain the purpose

## build-service-account.bash

```
# ./build-service-account.bash

checking connection:
connected to [api-crc-testing:6443]

checking gcloud --project example-project:
connected to gcloud --project example-project with [sa@redhat.com]

checking existing service account [sa-test]
sa-test                  sa-test@example-project.iam.gserviceaccount.com                  False

enable service account [ sa-test ]
Enabled service account [sa-test@example-project.iam.gserviceaccount.com].

set service account roles [sa-test]
Updated IAM policy for project [example-project].
bindings:
- members:
  - serviceAccount:sa-test@example-project.iam.gserviceaccount.com
  role: roles/compute.admin

check existing service account keys [sa-test]
KEY_ID                                    CREATED_AT            EXPIRES_AT
1234567890123456789012345678901234567890  2020-06-01T00:00:00Z  2020-07-01T00:00:00Z

downloading service account keys [sa-test]
created key [1234567890123456789012345678901234567890] of type [json] as [./key.json] for [sa-test@example-project.iam.gserviceaccount.com]

Note: After you create a key, you might need to wait for 60 seconds or more before you use the key. If you try to use a key immediately after you create it, and you receive an error, wait at least 60 seconds and try again.
https://cloud.google.com/iam/docs/creating-managing-service-account-keys#creating_service_account_keys

uploading service account keys [sa-test] to [gcp-project-operator:secret/gcp-project-operator-credentials]
secret/gcp-project-operator-credentials created
```

