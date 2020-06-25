#!/bin/bash

source gcp_resources.env
ERR_MISSINGENV=1

if [ -z $PARENTFOLDERID ]; then
	echo "Parent  folder’s ID required in environment PARENTFOLDERID"
	exit $ERR_MISSINGENV
fi
if [ -z $BILLINGACCOUNT ]; then
	echo "Billing account required in environment BILLINGACCOUNT"
	exit $ERR_MISSINGENV
fi
if [ -z $GCP_PRIVATE_KEY_LOCATION ]; then
	echo "Location of the private gcp credentials, required environment GCP_PRIVATE_KEY_LOCATION "
	exit $ERR_MISSINGENV
fi
PROJECT_NAMESPACE=${PROJECT_NAMESPACE:-gcp-project-operator}

oc project $PROJECT_NAMESPACE 2>/dev/null || oc new-project $PROJECT_NAMESPACE

oc create -n $PROJECT_NAMESPACE configmap gcp-project-operator --from-literal parentFolderID=$PARENTFOLDERID --from-literal billingAccount=$BILLINGACCOUNT

oc create -n gcp-project-operator secret generic gcp-project-operator-credentials --from-file=key.json=$GCP_PRIVATE_KEY_LOCATION 
