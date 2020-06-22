#!/bin/bash
set -euo pipefail

# Confirm execution from correct directory
if [ ! -f ./config.source ]; then
    echo "Not in tool root";
    exit 1;
fi
source ./config.source

DATE=$(date +%Y%m%d-%H%M%S)

# Used in referencing for delete/enable commands
GCLOUD_SA_EMAIL="${GCLOUD_SA}@${GCLOUD_PROJECT}.iam.gserviceaccount.com"

GCLOUD="gcloud --project ${GCLOUD_PROJECT}"

echo -e "\nchecking connection:"
CURRENT_OC_CONTEXT=$(oc config view -o json | jq -r '.["current-context"]')
CURRENT_OC_SERVER=$(oc config view -o json | jq -r ".contexts[] | select(.name == \"${CURRENT_OC_CONTEXT}\") | .context.cluster")
if [ "${CURRENT_OC_SERVER}" != "${TARGET_OC_SERVER}" ]; then
    echo "oc cli configured to incorrect context, expected [${TARGET_OC_SERVER}] got [${CURRENT_OC_SERVER}]";
    echo "please log into your expected cluster and try again, or reconfigure ./config.source";
    exit 1;
fi
echo "connected to [${CURRENT_OC_SERVER}]"

echo -e "\nchecking ${GCLOUD}:"
GCLOUD_USER=$(${GCLOUD} config get-value core/account)
if [ "${GCLOUD_USER}" != "${TARGET_GCLOUD_USER}" ]; then
    echo "oc cli configured to incorrect context, expected [${TARGET_GCLOUD_USER}] got [${GCLOUD_USER}]";
    echo "please log into your expected gcloud account and try again, or reconfigure ./config.source";
    exit 1;
fi
echo "connected to ${GCLOUD} with [${GCLOUD_USER}]"

echo -e "\nchecking existing service account [${GCLOUD_SA}]"
if ! ${GCLOUD} iam service-accounts list --quiet --filter name:"${GCLOUD_SA}" | grep "${GCLOUD_SA}"; then
    echo "creating service account [${GCLOUD_SA}]"
    ## TODO: Change URL to being where this script is located
    ${GCLOUD} iam service-accounts create "${GCLOUD_SA}" \
        --description="date: ${DATE}; docs: https://github.com/openshift/gcp-project-operator/blob/master/docs/gcpconfig.md" \
        --display-name="${GCLOUD_SA}"
fi

echo -e "\nenable service account [${GCLOUD_SA}]"
${GCLOUD} iam service-accounts enable "${GCLOUD_SA_EMAIL}"

echo -e "\nset service account roles [${GCLOUD_SA}]"

# Required Roles are defined here:
# https://github.com/openshift/gcp-project-operator/blob/master/pkg/controller/projectreference/projectreference_adapter.go#L58-L68
for required_role in \
    roles/storage.admin \
    roles/iam.serviceAccountUser \
    roles/iam.serviceAccountKeyAdmin \
    roles/iam.serviceAccountAdmin \
    roles/iam.securityAdmin \
    roles/dns.admin \
    roles/compute.admin \
; do
    ${GCLOUD} projects add-iam-policy-binding "${GCLOUD_PROJECT}" \
        --member="serviceAccount:${GCLOUD_SA_EMAIL}" --role=${required_role}
done

echo -e "\ncheck existing service account keys [${GCLOUD_SA}]"
${GCLOUD} iam service-accounts keys list --iam-account "${GCLOUD_SA_EMAIL}"

echo -e "\ndownloading service account keys [${GCLOUD_SA}]"
${GCLOUD} iam service-accounts keys create ./key.json --iam-account "${GCLOUD_SA_EMAIL}"

echo -e "\nNote: After you create a key, you might need to wait for 60 seconds or more before you use the key. If you try to use a key immediately after you create it, and you receive an error, wait at least 60 seconds and try again."
echo "https://cloud.google.com/iam/docs/creating-managing-service-account-keys#creating_service_account_keys"

echo -e "\nuploading service account keys [${GCLOUD_SA}] to [${GPO_NAMESPACE}:secret/${GPO_SECRET}]"
oc delete -n "${GPO_NAMESPACE}" secret "${GPO_SECRET}"
oc create -n "${GPO_NAMESPACE}" secret generic "${GPO_SECRET}" --from-file=key.json=./key.json
