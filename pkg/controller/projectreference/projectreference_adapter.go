package projectreference

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/openshift/cluster-api/pkg/util"
	clusterapi "github.com/openshift/cluster-api/pkg/util"
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	condition "github.com/openshift/gcp-project-operator/pkg/condition"
	"github.com/openshift/gcp-project-operator/pkg/configmap"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	gcputil "github.com/openshift/gcp-project-operator/pkg/util"
	operrors "github.com/openshift/gcp-project-operator/pkg/util/errors"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iam/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectState bool

const (
	ObjectModified  ObjectState = true
	ObjectUnchanged ObjectState = false
)

const (
	osdServiceAccountName = "osd-managed-admin"
	FinalizerName         = "finalizer.gcp.managed.openshift.io"
)

// OSDRequiredAPIS is list of API's, required to setup
// OpenShift cluster. Order is important.
var OSDRequiredAPIS = []string{
	"serviceusage.googleapis.com",
	"cloudresourcemanager.googleapis.com",
	"storage-component.googleapis.com",
	"storage-api.googleapis.com",
	"dns.googleapis.com",
	"iam.googleapis.com",
	"compute.googleapis.com",
	"cloudapis.googleapis.com",
	"iamcredentials.googleapis.com",
	"servicemanagement.googleapis.com",
}

// OSDRequiredRoles is a list of Roles that a service account
// required to setup Openshift cluster
var OSDRequiredRoles = []string{
	"roles/storage.admin",
	"roles/iam.serviceAccountUser",
	"roles/iam.serviceAccountKeyAdmin",
	"roles/iam.serviceAccountAdmin",
	"roles/iam.securityAdmin",
	"roles/dns.admin",
	"roles/compute.admin",
}

// OSDSREConsoleAccessRoles is a list of Roles that a service account
// required to get console access.
var OSDSREConsoleAccessRoles = []string{
	"roles/compute.admin",
	//"roles/iam.organizationRoleAdmin",
	"roles/editor",
	//"roles/resourcemanager.organizationViewer",
	"roles/resourcemanager.projectIamAdmin",
	"roles/servicemanagement.quotaAdmin",
	"roles/iam.serviceAccountAdmin",
	"roles/serviceusage.serviceUsageAdmin",
	"roles/orgpolicy.policyViewer",
}

//ReferenceAdapter is used to do all the processing of the ProjectReference type inside the reconcile loop
type ReferenceAdapter struct {
	ProjectClaim     *gcpv1alpha1.ProjectClaim
	ProjectReference *gcpv1alpha1.ProjectReference
	logger           logr.Logger
	kubeClient       client.Client
	gcpClient        gcpclient.Client
	conditionManager condition.Conditions
	OperatorConfig   configmap.OperatorConfigMap
}

type ensureAzResult int

const (
	ensureAzResultNotReady ensureAzResult = iota
	ensureAzResultModified
	ensureAzResultNoChange
)

// NewReferenceAdapter creates an adapter to turn what is requested in a ProjectReference into a GCP project and write the output back.
func NewReferenceAdapter(
	projectReference *gcpv1alpha1.ProjectReference,
	logger logr.Logger, client client.Client,
	gcpClient gcpclient.Client,
	manager condition.Conditions,
	cm configmap.OperatorConfigMap,
) (*ReferenceAdapter, error) {
	projectClaim, err := getMatchingClaimLink(projectReference, client)
	if err != nil {
		return &ReferenceAdapter{}, err
	}

	r := &ReferenceAdapter{
		ProjectClaim:     projectClaim,
		ProjectReference: projectReference,
		logger:           logger,
		kubeClient:       client,
		gcpClient:        gcpClient,
		conditionManager: manager,
		OperatorConfig:   cm,
	}
	return r, nil
}

func EnsureProjectClaimReady(r *ReferenceAdapter) (gcputil.OperationResult, error) {
	if r.ProjectReference.Status.State != gcpv1alpha1.ProjectReferenceStatusReady {
		return gcputil.ContinueProcessing()
	}

	if r.ProjectReference.Status.State == gcpv1alpha1.ProjectReferenceStatusReady && r.ProjectClaim.Status.State == gcpv1alpha1.ClaimStatusReady {
		return gcputil.StopProcessing()
	}

	azResult, err := r.ensureClaimAvailabilityZonesSet()
	if err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, "error ensuring availability zones"))
	}

	idModified := r.ensureClaimProjectIDSet()

	if azResult == ensureAzResultModified || idModified {
		err := r.kubeClient.Update(context.TODO(), r.ProjectClaim)
		if err != nil {
			return gcputil.RequeueWithError(operrors.Wrap(err, "error updating ProjectClaim spec"))
		}
		return gcputil.StopProcessing()
	}

	if azResult == ensureAzResultNotReady {
		r.logger.V(2).Info("Compute API not yet fully initialized. Retrying in 30 seconds.")
		return gcputil.RequeueAfter(30*time.Second, nil)
	}

	r.logger.V(2).Info("Project Ready update matchingClaim to ready")
	r.ProjectClaim.Status.State = gcpv1alpha1.ClaimStatusReady
	err = r.kubeClient.Status().Update(context.TODO(), r.ProjectClaim)
	if err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, "error updating ProjectClaim status"))
	}
	return gcputil.StopProcessing()
}

func VerifyProjectClaimPending(r *ReferenceAdapter) (gcputil.OperationResult, error) {
	if r.ProjectClaim.Status.State != gcpv1alpha1.ClaimStatusPendingProject {
		return gcputil.RequeueAfter(5*time.Second, nil)
	}
	return gcputil.ContinueProcessing()
}

func EnsureProjectReferenceStatusCreating(adapter *ReferenceAdapter) (gcputil.OperationResult, error) {
	if adapter.ProjectReference.Status.State != "" {
		return gcputil.ContinueProcessing()
	}
	adapter.ProjectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusCreating
	err := adapter.kubeClient.Status().Update(context.TODO(), adapter.ProjectReference)
	if err != nil {
		err = operrors.Wrap(err, "error updating ProjectReference status")
		return gcputil.RequeueWithError(err)
	}
	return gcputil.StopProcessing()
}

func EnsureProjectID(adapter *ReferenceAdapter) (gcputil.OperationResult, error) {
	if adapter.ProjectReference.Spec.GCPProjectID != "" {
		return gcputil.ContinueProcessing()
	}
	adapter.logger.V(1).Info("Creating ProjectID in ProjectReference CR")
	err := adapter.UpdateProjectID()
	if err != nil {
		err = operrors.Wrap(err, "could not update ProjectID in Project Reference CR")
		return gcputil.RequeueWithError(err)
	}
	return gcputil.StopProcessing()
}

func EnsureProjectCreated(r *ReferenceAdapter) (gcputil.OperationResult, error) {
	if r.isCCS() {
		return gcputil.ContinueProcessing()
	}

	err := r.createProject(r.OperatorConfig.ParentFolderID)
	if err != nil {
		if err == operrors.ErrInactiveProject {
			r.ProjectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusError
			err := r.kubeClient.Status().Update(context.TODO(), r.ProjectReference)
			if err != nil {
				return gcputil.RequeueWithError(operrors.Wrap(err, "error updating ProjectReference status"))
			}
			return gcputil.StopProcessing()
		}
		return gcputil.RequeueWithError(operrors.Wrap(err, "could not create project"))
	}

	// should this be it's own function?
	r.logger.V(1).Info("Configuring Billing APIS")
	err = r.configureBillingAPI()
	if err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, "error configuring Billing APIS"))
	}

	return gcputil.ContinueProcessing()
}

func (r *ReferenceAdapter) isCCS() bool {
	return r.ProjectReference.Spec.CCS
}

func EnsureProjectConfigured(r *ReferenceAdapter) (gcputil.OperationResult, error) {
	r.logger.V(1).Info("Configuring APIS")
	err := r.configureAPIS()
	if err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, "error configuring APIS"))
	}

	r.logger.V(1).Info("Configuring Service Account " + osdServiceAccountName)
	result, err := r.configureServiceAccount(OSDRequiredRoles)
	if err != nil || result.RequeueRequest {
		return result, err
	}

	r.logger.V(1).Info("Creating Credentials")
	result, err = r.createCredentials()
	if err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, "error creating credentials"))
	}

	if r.isCCS() {
		r.logger.V(1).Info("Configuring Service Account Permissions for Console Access")
		for _, email := range r.OperatorConfig.CCSConsoleAccess {
			// TODO(yeya24): Use google API to check whether this email is
			// for a group or a service account.
			if err := r.SetIAMPolicy(email, OSDSREConsoleAccessRoles, true); err != nil {
				return result, err
			}
		}
	}

	return result, nil
}

func EnsureStateReady(r *ReferenceAdapter) (gcputil.OperationResult, error) {
	if r.ProjectReference.Status.State != gcpv1alpha1.ProjectReferenceStatusReady {
		r.logger.V(1).Info("Setting Status on projectReference")
		r.ProjectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusReady
		return gcputil.RequeueOnErrorOrStop(r.kubeClient.Status().Update(context.TODO(), r.ProjectReference))
	}
	return gcputil.ContinueProcessing()
}

func getMatchingClaimLink(projectReference *gcpv1alpha1.ProjectReference, client client.Client) (*gcpv1alpha1.ProjectClaim, error) {
	projectClaim := &gcpv1alpha1.ProjectClaim{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: projectReference.Spec.ProjectClaimCRLink.Name, Namespace: projectReference.Spec.ProjectClaimCRLink.Namespace}, projectClaim)
	if err != nil {
		return &gcpv1alpha1.ProjectClaim{}, err

	}
	return projectClaim, nil
}

// UpdateProjectID updates the ProjectReference with a unique ID for the ProjectID
func (r *ReferenceAdapter) UpdateProjectID() error {
	projectID, err := GenerateProjectID()
	if err != nil {
		return err
	}
	r.ProjectReference.Spec.GCPProjectID = projectID
	return r.kubeClient.Update(context.TODO(), r.ProjectReference)
}

func EnsureDeletionProcessed(adapter *ReferenceAdapter) (gcputil.OperationResult, error) {
	// Cleanup
	if adapter.IsDeletionRequested() {
		err := adapter.EnsureProjectCleanedUp()
		if err != nil {
			return gcputil.RequeueAfter(5*time.Second, err)
		}
		return gcputil.StopProcessing()
	}
	return gcputil.ContinueProcessing()
}

// IsDeletionRequested checks the metadata.deletionTimestamp of ProjectReference instance, and returns if delete requested.
// The controllers watching the ProjectReference use this as a signal to know when to execute the finalizer.
func (r *ReferenceAdapter) IsDeletionRequested() bool {
	return r.ProjectReference.DeletionTimestamp != nil
}

// EnsureFinalizerAdded parses the meta.Finalizers of ProjectReference instance and adds FinalizerName if not found.
func EnsureFinalizerAdded(r *ReferenceAdapter) (gcputil.OperationResult, error) {
	if !clusterapi.Contains(r.ProjectReference.GetFinalizers(), FinalizerName) {
		r.ProjectReference.SetFinalizers(append(r.ProjectReference.GetFinalizers(), FinalizerName))
		return gcputil.RequeueOnErrorOrStop(r.kubeClient.Update(context.TODO(), r.ProjectReference))
	}
	return gcputil.ContinueProcessing()
}

// EnsureFinalizerDeleted parses the meta.Finalizers of ProjectReference instance and removes FinalizerName if found;
func (r *ReferenceAdapter) EnsureFinalizerDeleted() error {
	r.logger.Info("Deleting Finalizer")
	finalizers := r.ProjectReference.GetFinalizers()
	if clusterapi.Contains(finalizers, FinalizerName) {
		r.ProjectReference.SetFinalizers(clusterapi.Filter(finalizers, FinalizerName))
		return r.kubeClient.Update(context.TODO(), r.ProjectReference)
	}
	return nil
}

// EnsureProjectCleanedUp deletes the project, the secret and the finalizer if they still exist
func (r *ReferenceAdapter) EnsureProjectCleanedUp() error {
	if !r.isCCS() {
		err := r.deleteProject()
		if err != nil {
			return err
		}
	}

	err := r.deleteCredentials()
	if err != nil {
		return err
	}

	err = r.EnsureFinalizerDeleted()
	if err != nil {
		return err
	}

	return nil
}

func GenerateProjectID() (string, error) {
	guid := uuid.New().String()
	hashing := sha1.New()
	_, err := hashing.Write([]byte(guid))
	if err != nil {
		return "", err
	}
	uuidsum := fmt.Sprintf("%x", hashing.Sum(nil))
	shortuuid := uuidsum[0:8]
	return "o-" + shortuuid, nil
}

func (r *ReferenceAdapter) clearProjectID() error {
	r.ProjectReference.Spec.GCPProjectID = ""
	return r.kubeClient.Update(context.TODO(), r.ProjectReference)
}

// deleteProject checks the Project's lifecycle state of the projectReference.Spec.GCPProjectID instance in Google GCP
// and deletes it if not active
func (r *ReferenceAdapter) deleteProject() error {
	project, projectExists, err := r.getProject(r.ProjectReference.Spec.GCPProjectID)
	if err != nil {
		return err
	}

	if !projectExists {
		return nil
	}

	switch project.LifecycleState {
	case "DELETE_REQUESTED":
		r.logger.Info("Project lifecycleState == DELETE_REQUESTED") //TODO: change message to be more consice
		return nil
	case "LIFECYCLE_STATE_UNSPECIFIED":
		return operrors.Wrap(operrors.ErrUnexpectedLifecycleState, fmt.Sprintf("unexpected lifecycleState for %s", project.LifecycleState))
	case "ACTIVE":
		r.logger.Info("Deleting Project")
		_, err := r.gcpClient.DeleteProject(project.ProjectId)
		return err
	default:
		return fmt.Errorf("ProjectReference Controller is unable to understand the project.LifecycleState %s", project.LifecycleState)
	}
}

func (r *ReferenceAdapter) createProject(parentFolderID string) error {
	project, projectExists, err := r.getProject(r.ProjectReference.Spec.GCPProjectID)
	if err != nil {
		return err
	}

	if projectExists {
		switch project.LifecycleState {
		case "ACTIVE":
			r.logger.V(1).Info("Project lifecycleState == ACTIVE") //TODO: change message to be more consice
			return nil
		case "DELETE_REQUESTED":
			return operrors.ErrInactiveProject
		default:
			return operrors.Wrap(operrors.ErrUnexpectedLifecycleState, fmt.Sprintf("unexpected lifecycleState for %s", project.LifecycleState))
		}
	}

	r.logger.Info("Creating Project")
	// If we cannot create the project clear the projectID from spec so we can try again with another unique key
	_, creationFailed := r.gcpClient.CreateProject(parentFolderID)
	if creationFailed != nil {
		r.logger.V(1).Info("Clearing gcpProjectID from ProjectReferenceSpec")
		//Todo() We need to requeue here ot it will continue to the next step.
		if err = r.clearProjectID(); err != nil {
			return operrors.Wrap(creationFailed, fmt.Sprintf("could not clear project ID: %v", err))
		}

		return operrors.Wrap(creationFailed, fmt.Sprintf("could not create project. Parent Folder ID: %s, Requested Project ID: %s", parentFolderID, r.ProjectReference.Spec.GCPProjectID))
	}

	return nil
}

func (r *ReferenceAdapter) getProject(projectId string) (*cloudresourcemanager.Project, bool, error) {
	// Get existing projects
	projects, err := r.gcpClient.ListProjects()
	if err != nil {
		return nil, false, err
	}

	projectMap := convertProjectsToMap(projects)
	project, exists := projectMap[projectId]

	return project, exists, err
}

func (r *ReferenceAdapter) configureBillingAPI() error {
	enabledAPIs, err := r.gcpClient.ListAPIs(r.ProjectReference.Spec.GCPProjectID)
	if err != nil {
		return err
	}

	if !util.Contains(enabledAPIs, "cloudbilling.googleapis.com") {
		r.logger.Info("Enabling Billing API")
		err := r.gcpClient.EnableAPI(r.ProjectReference.Spec.GCPProjectID, "cloudbilling.googleapis.com")
		if err != nil {
			return operrors.Wrap(err, fmt.Sprintf("Error enabling cloudbilling.googleapis.com api for project %s", r.ProjectReference.Spec.GCPProjectID))
		}
	}

	err = r.gcpClient.CreateCloudBillingAccount(r.ProjectReference.Spec.GCPProjectID, r.OperatorConfig.BillingAccount)
	if err != nil {
		return operrors.Wrap(err, "error creating CloudBilling")
	}

	return nil
}

func (r *ReferenceAdapter) configureAPIS() error {
	enabledAPIs, err := r.gcpClient.ListAPIs(r.ProjectReference.Spec.GCPProjectID)
	if err != nil {
		return err
	}

	for _, api := range OSDRequiredAPIS {
		if !util.Contains(enabledAPIs, api) {
			err = r.gcpClient.EnableAPI(r.ProjectReference.Spec.GCPProjectID, api)
			if err != nil {
				return operrors.Wrap(err, fmt.Sprintf("error enabling %s api for project %s", api, r.ProjectReference.Spec.GCPProjectID))
			}
		}
	}

	return nil
}

func (r *ReferenceAdapter) configureServiceAccount(policies []string) (gcputil.OperationResult, error) {
	// See if GCP service account exists if not create it
	var serviceAccount *iam.ServiceAccount
	serviceAccount, err := r.gcpClient.GetServiceAccount(osdServiceAccountName)
	if err != nil {
		// Create OSDManged Service account
		r.logger.Info("Creating Service Account")
		account, err := r.gcpClient.CreateServiceAccount(osdServiceAccountName, osdServiceAccountName)
		if err != nil {
			if matchesAlreadyExistsError(err) {
				r.logger.V(2).Info("Service Account not yet fully initialized. Retrying in 30 seconds.")
				return gcputil.RequeueAfter(30*time.Second, nil)
			}
			return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("could not create service account for %s", osdServiceAccountName)))
		}
		serviceAccount = account
	}

	r.logger.V(1).Info("Setting Service Account Policies")
	err = r.SetIAMPolicy(serviceAccount.Email, policies, false)
	if err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("could not update policy on project for %s", r.ProjectReference.Spec.GCPProjectID)))
	}

	return gcputil.ContinueProcessing()
}

func (r *ReferenceAdapter) createCredentials() (gcputil.OperationResult, error) {
	if gcputil.SecretExists(r.kubeClient, r.ProjectClaim.Spec.GCPCredentialSecret.Name, r.ProjectClaim.Spec.GCPCredentialSecret.Namespace) {
		return gcputil.ContinueProcessing()
	}

	r.logger.Info("Creating credentials")
	serviceAccount, err := r.gcpClient.GetServiceAccount(osdServiceAccountName)
	if err != nil {
		if matchesNotFoundError(err) {
			r.logger.V(1).Info("Service Account not yet fully initialized. Retrying in 30 seconds.")
			return gcputil.RequeueAfter(30*time.Second, nil)
		}
		return gcputil.RequeueWithError(operrors.Wrap(err, "could not get service account"))
	}
	r.logger.V(1).Info("Creating Service AccountKey")
	key, err := r.gcpClient.CreateServiceAccountKey(serviceAccount.Email)
	if err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("could not create service account key for %s", serviceAccount.Email)))
	}

	r.logger.V(2).Info("Create secret for the key and store it")
	privateKeyString, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		return gcputil.RequeueWithError(operrors.Wrap(err, "could not decode secret"))
	}

	secret := gcputil.NewGCPSecretCR(string(privateKeyString), types.NamespacedName{
		Namespace: r.ProjectClaim.Spec.GCPCredentialSecret.Namespace,
		Name:      r.ProjectClaim.Spec.GCPCredentialSecret.Name,
	})

	r.logger.V(1).Info(fmt.Sprintf("Creating Secret %s in namespace %s", r.ProjectClaim.Spec.GCPCredentialSecret.Name, r.ProjectClaim.Spec.GCPCredentialSecret.Namespace))
	createErr := r.kubeClient.Create(context.TODO(), secret)
	if createErr != nil {
		return gcputil.RequeueWithError(operrors.Wrap(createErr, fmt.Sprintf("could not create service account secret for %s", r.ProjectClaim.Spec.GCPCredentialSecret.Name)))
	}

	return gcputil.ContinueProcessing()
}

func (r *ReferenceAdapter) deleteCredentials() error {
	secret := types.NamespacedName{
		Namespace: r.ProjectClaim.Spec.GCPCredentialSecret.Namespace,
		Name:      r.ProjectClaim.Spec.GCPCredentialSecret.Name,
	}
	r.logger.Info("Deleting Credentials")

	r.logger.V(2).Info("Check if the Secret exists")
	if gcputil.SecretExists(r.kubeClient, secret.Name, secret.Namespace) {
		r.logger.V(2).Info("Getting Secret")
		key, err := gcputil.GetSecret(r.kubeClient, secret.Name, secret.Namespace)
		if err != nil {
			return operrors.Wrap(err, fmt.Sprintf("could not get the service account secret for %s", secret.Name))
		}

		r.logger.V(2).Info("Deleting secret")
		err = r.kubeClient.Delete(context.TODO(), key)
		if err != nil {
			return operrors.Wrap(err, fmt.Sprintf("could not delete service account secret for %s", secret.Name))
		}
	}

	return nil
}

// ensureAvailabilityZonesSet sets the az in the projectclaim spec if necessary
// returns true if the project claim has been modified
func (r *ReferenceAdapter) ensureClaimAvailabilityZonesSet() (ensureAzResult, error) {
	if len(r.ProjectClaim.Spec.AvailabilityZones) > 0 {
		return ensureAzResultNoChange, nil
	}

	zones, err := r.gcpClient.ListAvailabilityZones(r.ProjectReference.Spec.GCPProjectID, r.ProjectClaim.Spec.Region)
	if err != nil {
		return r.handleAvailabilityZonesError(err)
	}
	conditions := &r.ProjectReference.Status.Conditions
	r.conditionManager.SetCondition(conditions, gcpv1alpha1.ConditionComputeApiReady, corev1.ConditionTrue, "QueryAvailabilityZonesSucceeded", "ComputeAPI ready, successfully queried availability zones")

	r.ProjectClaim.Spec.AvailabilityZones = zones

	return ensureAzResultModified, nil
}

func (r *ReferenceAdapter) handleAvailabilityZonesError(err error) (ensureAzResult, error) {
	conditions := &r.ProjectReference.Status.Conditions
	if matchesComputeApiNotReadyError(err) {
		apiCondition, found := r.conditionManager.FindCondition(conditions, gcpv1alpha1.ConditionComputeApiReady)
		tenMinutesAgo := metav1.NewTime(time.Now().Add(time.Duration(-10 * time.Minute)))
		if found && apiCondition.LastTransitionTime.Before(&tenMinutesAgo) {
			r.conditionManager.SetCondition(conditions, gcpv1alpha1.ConditionComputeApiReady, corev1.ConditionFalse, "QueryAvailabilityZonesFailed", "ComputeAPI not yet ready, couldn't query availability zones")
			_ = r.StatusUpdate()
			return ensureAzResultNoChange, err
		}
		r.conditionManager.SetCondition(conditions, gcpv1alpha1.ConditionComputeApiReady, corev1.ConditionFalse, "QueryAvailabilityZonesFailed", "ComputeAPI not yet ready, couldn't query availability zones")
		return ensureAzResultNotReady, r.StatusUpdate()
	}
	return ensureAzResultNoChange, err
}

func (r *ReferenceAdapter) ensureClaimProjectIDSet() bool {
	if r.ProjectClaim.Spec.GCPProjectID == "" {
		r.ProjectClaim.Spec.GCPProjectID = r.ProjectReference.Spec.GCPProjectID
		return true
	}

	return false
}

func EnsureProjectReferenceInitialized(r *ReferenceAdapter) (gcputil.OperationResult, error) {
	if r.ProjectReference.Status.Conditions == nil {
		r.ProjectReference.Status.Conditions = []gcpv1alpha1.Condition{}
		err := r.StatusUpdate()
		if err != nil {
			return gcputil.RequeueWithError(operrors.Wrap(err, "failed to initalize ProjectReference"))
		}
		return gcputil.StopProcessing()
	}
	return gcputil.ContinueProcessing()
}

// AddorUpdateBindingResponse contines the data that is returned by the AddOrUpdarteBindings function
type AddorUpdateBindingResponse struct {
	modified bool
	policy   *cloudresourcemanager.Policy
}

// AddOrUpdateBindings gets the policy and checks if the bindings match the required roles
func (r *ReferenceAdapter) AddOrUpdateBindings(serviceAccountEmail string, policies []string, group bool) (AddorUpdateBindingResponse, error) {
	policy, err := r.gcpClient.GetIamPolicy(r.ProjectReference.Spec.GCPProjectID)
	if err != nil {
		return AddorUpdateBindingResponse{}, err
	}

	//Checking if policy is modified
	newBindings, modified := gcputil.AddOrUpdateBinding(policy.Bindings, policies, serviceAccountEmail, group)

	// add new bindings to policy
	policy.Bindings = newBindings
	return AddorUpdateBindingResponse{
		modified: modified,
		policy:   policy,
	}, nil
}

// SetIAMPolicy attempts to update policy if the policy needs to be modified
func (r *ReferenceAdapter) SetIAMPolicy(serviceAccountEmail string, policies []string, group bool) error {
	// Checking if policy needs to be updated
	var retry int
	for {
		retry++
		time.Sleep(time.Second)
		addorUpdateResponse, err := r.AddOrUpdateBindings(serviceAccountEmail, policies, group)
		if err != nil {
			return err
		}

		// If existing bindings have been modified update the policy
		if addorUpdateResponse.modified {
			setIamPolicyRequest := &cloudresourcemanager.SetIamPolicyRequest{
				Policy: addorUpdateResponse.policy,
			}
			_, err = r.gcpClient.SetIamPolicy(setIamPolicyRequest)
			if err != nil {
				ae, ok := err.(*googleapi.Error)
				// retry rules below:

				if ok && ae.Code == http.StatusConflict && retry <= 3 {
					continue
				}
				return err
			}
			return nil
		}
		return nil
	}

}

// SetProjectReferenceCondition calls SetCondition() with project reference conditions
// It returns nil if no conditions defined before and the err is nil
// It updates the condition with err message, probe, etc... if err does exist
// It marks the condition as resolved if the err is nil and there is at least one condition defined before
func (r *ReferenceAdapter) SetProjectReferenceCondition(reason string, err error) error {
	conditions := &r.ProjectReference.Status.Conditions
	conditionType := gcpv1alpha1.ConditionError
	if err != nil {
		r.conditionManager.SetCondition(conditions, conditionType, corev1.ConditionTrue, reason, err.Error())
	} else {
		if len(*conditions) != 0 {
			reason = reason + "Resolved"
			r.conditionManager.SetCondition(conditions, conditionType, corev1.ConditionFalse, reason, "")
		} else {
			return nil
		}
	}

	return r.StatusUpdate()
}

// StatusUpdate updates the project reference status
func (r *ReferenceAdapter) StatusUpdate() error {
	err := r.kubeClient.Status().Update(context.TODO(), r.ProjectReference)
	if err != nil {
		return operrors.Wrap(err, fmt.Sprintf("failed to update ProjectClaim state for %s", r.ProjectReference.Name))
	}

	return nil
}

// convertProjectsToMap converts []*cloudresourcemanager.Project map[string]*cloudresourcemanager.Project with the projectID as the map key
func convertProjectsToMap(projects []*cloudresourcemanager.Project) map[string]*cloudresourcemanager.Project {
	projectMap := make(map[string]*cloudresourcemanager.Project)

	for _, project := range projects {
		projectMap[project.ProjectId] = project
	}

	return projectMap
}

func matchesAlreadyExistsError(err error) bool {
	return strings.HasPrefix(err.Error(), "googleapi: Error 409:")
}

func matchesNotFoundError(err error) bool {
	return strings.HasPrefix(err.Error(), "googleapi: Error 404:")
}

func matchesComputeApiNotReadyError(err error) bool {
	return strings.HasPrefix(err.Error(), "googleapi: Error 403: Compute Engine API has not been used in project") ||
		strings.HasPrefix(err.Error(), "googleapi: Error 403: Access Not Configured. Compute Engine API has not been used in project")
}
