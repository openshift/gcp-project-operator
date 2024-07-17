package projectreference

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/openshift/gcp-project-operator/pkg/configmap"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	"github.com/openshift/gcp-project-operator/pkg/util"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iam/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gcpv1alpha1 "github.com/openshift/gcp-project-operator/api/v1alpha1"
	condition "github.com/openshift/gcp-project-operator/pkg/condition"
	operrors "github.com/openshift/gcp-project-operator/pkg/util/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

const (
	osdServiceAccountNameDefault = "osd-managed-admin"
	FinalizerName                = "finalizer.gcp.managed.openshift.io"
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
	"networksecurity.googleapis.com", // https://bugzilla.redhat.com/show_bug.cgi?id=2021731
}

// OSDRequiredRoles is a list of Roles for service account osd-managed-admin
// used by the cloud-credential-operator to setup Openshift cluster
var OSDRequiredRoles = []string{
	"roles/compute.admin",
	"roles/dns.admin",
	"roles/iam.roleAdmin",
	"roles/iam.securityAdmin",
	"roles/iam.serviceAccountAdmin",
	"roles/iam.serviceAccountKeyAdmin",
	"roles/iam.serviceAccountUser",
	"roles/storage.admin",
}

// OSDSREConsoleAccessRoles is a list of Roles that a service account
// required to get console access.
var OSDSREConsoleAccessRoles = []string{
	"roles/compute.admin",
	"roles/editor",
	"roles/resourcemanager.projectIamAdmin",
	"roles/servicemanagement.quotaAdmin",
	"roles/iam.serviceAccountAdmin",
	"roles/serviceusage.serviceUsageAdmin",
	"roles/iam.roleAdmin",
	"roles/cloudsupport.techSupportEditor",
}

// OSDReadOnlyConsoleAccessRoles is a list of Roles that a service account
// required to get read only console access.
var OSDReadOnlyConsoleAccessRoles = []string{
	"roles/viewer",
}

// OSDSharedVPCRoles is a list of Roles that a service account
// required to get shared VPC access
var OSDSharedVPCRoles = []string{
	"roles/iam.securityReviewer",
	"roles/compute.loadBalancerAdmin",
	"roles/resourcemanager.tagUser",
	"roles/compute.networkAdmin",
}

// ReferenceAdapter is used to do all the processing of the ProjectReference type inside the reconcile loop
type ReferenceAdapter struct {
	ProjectClaim     *gcpv1alpha1.ProjectClaim
	ProjectReference *gcpv1alpha1.ProjectReference
	logger           logr.Logger
	kubeClient       client.Client
	gcpClient        gcpclient.Client
	conditionManager condition.Conditions
	OperatorConfig   configmap.OperatorConfigMap
}

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

// EnsureProjectClaimReady sets the ProjectClaim to Ready after the ProjectReference was reconciled correctly and gcp project has been created
func EnsureProjectClaimReady(r *ReferenceAdapter) (util.OperationResult, error) {
	if r.ProjectReference.Status.State != gcpv1alpha1.ProjectReferenceStatusReady {
		return util.ContinueProcessing()
	}

	if r.ProjectReference.Status.State == gcpv1alpha1.ProjectReferenceStatusReady && r.ProjectClaim.Status.State == gcpv1alpha1.ClaimStatusReady {
		return util.StopProcessing()
	}

	res, err := r.ensureClaimAvailabilityZonesSet()
	if err != nil || res.RequeueOrCancel() {
		return res, err
	}

	idModified := r.ensureClaimProjectIDSet()
	if idModified {
		err := r.kubeClient.Update(context.TODO(), r.ProjectClaim)
		if err != nil {
			return util.RequeueWithError(operrors.Wrap(err, "error updating ProjectClaim spec"))
		}
	}
	r.ProjectClaim.Status.State = gcpv1alpha1.ClaimStatusReady
	if err := r.kubeClient.Status().Update(context.TODO(), r.ProjectClaim); err != nil {
		return util.RequeueWithError(operrors.Wrap(err, "error updating ProjectClaim status"))
	}
	return util.StopProcessing()
}

// VerifyProjectClaimPending waits until the ProjectClaim has been initialized, meaning is in state PendingProject
func VerifyProjectClaimPending(r *ReferenceAdapter) (util.OperationResult, error) {
	if r.ProjectClaim.Status.State != gcpv1alpha1.ClaimStatusPendingProject {
		return util.RequeueAfter(5*time.Second, nil)
	}
	return util.ContinueProcessing()
}

func EnsureProjectReferenceStatusCreating(adapter *ReferenceAdapter) (util.OperationResult, error) {
	if adapter.ProjectReference.Status.State != "" {
		return util.ContinueProcessing()
	}
	adapter.ProjectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusCreating
	err := adapter.kubeClient.Status().Update(context.TODO(), adapter.ProjectReference)
	if err != nil {
		err = operrors.Wrap(err, "error updating ProjectReference status")
		return util.RequeueWithError(err)
	}
	return util.StopProcessing()
}

func serviceNameAlreadyGenerated(projectReference *gcpv1alpha1.ProjectReference) bool {
	osdServiceAccountNameDefaultPrefix := serviceAccountNameTemplate("")
	serviceAccountName := projectReference.Spec.ServiceAccountName
	return strings.HasPrefix(serviceAccountName, osdServiceAccountNameDefaultPrefix) &&
		len(serviceAccountName) > len(osdServiceAccountNameDefaultPrefix)
}

func serviceNameAlreadyFilled(projectReference *gcpv1alpha1.ProjectReference) bool {
	return projectReference.Spec.ServiceAccountName == osdServiceAccountNameDefault
}
func serviceAccountNameTemplate(suffix string) string {
	return fmt.Sprintf("%s-%s", osdServiceAccountNameDefault, suffix)
}

func EnsureServiceAccountName(adapter *ReferenceAdapter) (util.OperationResult, error) {
	if serviceNameAlreadyGenerated(adapter.ProjectReference) ||
		serviceNameAlreadyFilled(adapter.ProjectReference) {
		return util.ContinueProcessing()
	}

	adapter.logger.V(1).Info("Creating ServiceAccountName in ProjectReference CR")
	err := adapter.UpdateServiceAccountName()
	if err != nil {
		err = operrors.Wrap(err, "could not update ServiceAccountName in Project Reference CR")
		return util.RequeueWithError(err)
	}
	return util.StopProcessing()

}

func EnsureProjectID(adapter *ReferenceAdapter) (util.OperationResult, error) {
	if adapter.ProjectReference.Spec.GCPProjectID != "" {
		return util.ContinueProcessing()
	}
	adapter.logger.V(1).Info("Creating ProjectID in ProjectReference CR")
	err := adapter.UpdateProjectID()
	if err != nil {
		err = operrors.Wrap(err, "could not update ProjectID in Project Reference CR")
		return util.RequeueWithError(err)
	}
	return util.StopProcessing()
}

func EnsureProjectCreated(r *ReferenceAdapter) (util.OperationResult, error) {
	if r.isCCS() {
		return util.ContinueProcessing()
	}

	err := r.createProject(r.OperatorConfig.ParentFolderID)
	if err != nil {
		if err == operrors.ErrInactiveProject {
			r.ProjectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusError
			err := r.kubeClient.Status().Update(context.TODO(), r.ProjectReference)
			if err != nil {
				return util.RequeueWithError(operrors.Wrap(err, "error updating ProjectReference status"))
			}
			return util.StopProcessing()
		}
		return util.RequeueWithError(operrors.Wrap(err, "could not create project"))
	}

	// should this be it's own function?
	r.logger.V(1).Info("Configuring Billing APIS")
	err = r.configureBillingAPI()
	if err != nil {
		return util.RequeueWithError(operrors.Wrap(err, "error configuring Billing APIS"))
	}

	return util.ContinueProcessing()
}

func (r *ReferenceAdapter) isCCS() bool {
	return r.ProjectReference.Spec.CCS
}

func EnsureProjectConfigured(r *ReferenceAdapter) (util.OperationResult, error) {
	r.logger.V(1).Info("Configuring APIS")
	err := r.configureAPIS()
	if err != nil {
		return util.RequeueWithError(operrors.Wrap(err, "error configuring APIS"))
	}

	osdServiceAccountName := r.ProjectReference.Spec.ServiceAccountName
	r.logger.V(1).Info("Configuring Service Account " + osdServiceAccountName)

	serviceAccountRoles := OSDRequiredRoles
	if r.ProjectReference.Spec.SharedVPCAccess {
		r.logger.V(1).Info("Adding shared VPC access " + osdServiceAccountName)
		serviceAccountRoles = append(serviceAccountRoles, OSDSharedVPCRoles...)
	}

	result, err := r.configureServiceAccount(serviceAccountRoles)
	if err != nil || result.RequeueRequest {
		return result, err
	}

	r.logger.V(1).Info("Creating Credentials")
	result, err = r.createCredentials()
	if err != nil || result.RequeueRequest {
		return result, err
	}

	if r.isCCS() {
		r.logger.V(1).Info("Configuring IAM to allow console access")
		for _, email := range r.OperatorConfig.CCSConsoleAccess {
			// TODO(yeya24): Use google API to check whether this email is
			// for a group or a service account.
			if err := r.SetIAMPolicy(email, OSDSREConsoleAccessRoles, util.GoogleGroup); err != nil {
				return util.RequeueWithError(err)
			}
		}

		for _, email := range r.OperatorConfig.CCSReadOnlyConsoleAccess {
			if err := r.SetIAMPolicy(email, OSDReadOnlyConsoleAccessRoles, util.GoogleGroup); err != nil {
				return util.RequeueWithError(err)
			}
		}
	}
	return util.ContinueProcessing()
}

func EnsureStateReady(r *ReferenceAdapter) (util.OperationResult, error) {
	if r.ProjectReference.Status.State != gcpv1alpha1.ProjectReferenceStatusReady {
		r.logger.V(1).Info("Setting Status on projectReference")
		r.ProjectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusReady
		return util.RequeueOnErrorOrStop(r.kubeClient.Status().Update(context.TODO(), r.ProjectReference))
	}
	return util.ContinueProcessing()
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

func (r *ReferenceAdapter) UpdateServiceAccountName() error {
	const serviceAccountNameSuffixLength = 8
	// using k8s library instead of local implementation
	serviceAccountNameSuffix := utilrand.String(serviceAccountNameSuffixLength)

	r.ProjectReference.Spec.ServiceAccountName = serviceAccountNameTemplate(serviceAccountNameSuffix)
	return r.kubeClient.Update(context.TODO(), r.ProjectReference)
}

func EnsureDeletionProcessed(adapter *ReferenceAdapter) (util.OperationResult, error) {
	// Cleanup
	if adapter.IsDeletionRequested() {
		err := adapter.EnsureProjectCleanedUp()
		if err != nil {
			return util.RequeueAfter(5*time.Second, err)
		}
		return util.StopProcessing()
	}
	return util.ContinueProcessing()
}

// IsDeletionRequested checks the metadata.deletionTimestamp of ProjectReference instance, and returns if delete requested.
// The controllers watching the ProjectReference use this as a signal to know when to execute the finalizer.
func (r *ReferenceAdapter) IsDeletionRequested() bool {
	return r.ProjectReference.DeletionTimestamp != nil
}

// EnsureFinalizerAdded parses the meta.Finalizers of ProjectReference instance and adds FinalizerName if not found.
func EnsureFinalizerAdded(r *ReferenceAdapter) (util.OperationResult, error) {
	if !util.Contains(r.ProjectReference.GetFinalizers(), FinalizerName) {
		r.ProjectReference.SetFinalizers(append(r.ProjectReference.GetFinalizers(), FinalizerName))
		return util.RequeueOnErrorOrStop(r.kubeClient.Update(context.TODO(), r.ProjectReference))
	}
	return util.ContinueProcessing()
}

// EnsureFinalizerDeleted parses the meta.Finalizers of ProjectReference instance and removes FinalizerName if found;
func (r *ReferenceAdapter) EnsureFinalizerDeleted() error {
	r.logger.Info("Deleting Finalizer")
	finalizers := r.ProjectReference.GetFinalizers()
	if util.Contains(finalizers, FinalizerName) {
		r.ProjectReference.SetFinalizers(util.Filter(finalizers, FinalizerName))
		return r.kubeClient.Update(context.TODO(), r.ProjectReference)
	}
	return nil
}

// EnsureProjectCleanedUp deletes the project, the secret and the finalizer if they still exist
func (r *ReferenceAdapter) EnsureProjectCleanedUp() error {
	var err error

	err = r.deleteServiceAccount()
	if err != nil {
		return err
	}

	if !r.isCCS() {
		err = r.deleteProject()
		if err != nil {
			return err
		}
	}

	err = r.ensureCCSProjectCleanedUp(r.isCCS())
	if err != nil {
		return err
	}

	err = r.deleteCredentials()
	if err != nil {
		return err
	}

	err = r.EnsureFinalizerDeleted()
	if err != nil {
		return err
	}

	return nil
}

func (r *ReferenceAdapter) ensureCCSProjectCleanedUp(isCCS bool) error {
	if !isCCS {
		return nil
	}
	r.logger.Info("Deleting IAM Policy for console access")
	for _, email := range r.OperatorConfig.CCSConsoleAccess {
		if err := r.DeleteIAMPolicy(email, util.GoogleGroup); err != nil {
			return err
		}
	}
	for _, email := range r.OperatorConfig.CCSReadOnlyConsoleAccess {
		if err := r.DeleteIAMPolicy(email, util.GoogleGroup); err != nil {
			return err
		}
	}
	return nil
}

func GenerateProjectID() (string, error) {
	guid := uuid.New().String()
	hashing := sha256.New()
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

func (r *ReferenceAdapter) deleteServiceAccount() error {
	serviceAccountName := r.ProjectReference.Spec.ServiceAccountName

	r.logger.V(1).Info("SA delete started", "serviceAccountName", serviceAccountName)

	sa, err := r.gcpClient.GetServiceAccount(serviceAccountName)
	if err != nil {
		if matchesNotFoundError(err) {
			return nil
		}

		return operrors.Wrap(err, "could not get the SA, something happened")
	}

	r.logger.V(1).Info("after get")

	if err := r.DeleteIAMPolicy(sa.Email, util.ServiceAccount); err != nil {
		return operrors.Wrap(err, "could not delete the IAM policy, something happened")
	}

	if err := r.gcpClient.DeleteServiceAccount(sa.Email); err != nil {
		return operrors.Wrap(err, "could not delete the SA, something happened")
	}

	r.logger.V(1).Info("done")

	return nil
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
	_, creationFailed := r.gcpClient.CreateProject(parentFolderID, r.ProjectClaim.ObjectMeta.Name)
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

func (r *ReferenceAdapter) configureServiceAccount(policies []string) (util.OperationResult, error) {
	// See if GCP service account exists if not create it
	var serviceAccount *iam.ServiceAccount

	osdServiceAccountName := r.ProjectReference.Spec.ServiceAccountName
	serviceAccount, err := r.gcpClient.GetServiceAccount(osdServiceAccountName)
	if err != nil {
		// Create OSDManged Service account
		r.logger.Info("Creating Service Account")
		account, err := r.gcpClient.CreateServiceAccount(osdServiceAccountName, osdServiceAccountName)
		if err != nil {
			if matchesAlreadyExistsError(err) {
				r.logger.V(2).Info("Service Account not yet fully initialized. Retrying in 30 seconds.")
				return util.RequeueAfter(30*time.Second, nil)
			}
			return util.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("could not create service account for %s", osdServiceAccountName)))
		}
		serviceAccount = account
	}

	r.logger.V(1).Info("Setting Service Account Policies")
	err = r.SetIAMPolicy(serviceAccount.Email, policies, util.ServiceAccount)
	if err != nil {
		return util.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("could not update policy on project for %s", r.ProjectReference.Spec.GCPProjectID)))
	}

	return util.ContinueProcessing()
}

func (r *ReferenceAdapter) createCredentials() (util.OperationResult, error) {
	if util.SecretExists(r.kubeClient, r.ProjectClaim.Spec.GCPCredentialSecret.Name, r.ProjectClaim.Spec.GCPCredentialSecret.Namespace) {
		return util.ContinueProcessing()
	}

	r.logger.Info("Creating credentials")
	osdServiceAccountName := r.ProjectReference.Spec.ServiceAccountName
	serviceAccount, err := r.gcpClient.GetServiceAccount(osdServiceAccountName)
	if err != nil {
		if matchesNotFoundError(err) {
			r.logger.V(1).Info("Service Account not yet fully initialized. Retrying in 30 seconds.")
			return util.RequeueAfter(30*time.Second, nil)
		}
		return util.RequeueWithError(operrors.Wrap(err, "could not get service account"))
	}
	r.logger.V(1).Info("Creating Service AccountKey")
	key, err := r.gcpClient.CreateServiceAccountKey(serviceAccount.Email)
	if err != nil {
		return util.RequeueWithError(operrors.Wrap(err, fmt.Sprintf("could not create service account key for %s", serviceAccount.Email)))
	}

	r.logger.V(2).Info("Create secret for the key and store it")
	privateKeyString, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		return util.RequeueWithError(operrors.Wrap(err, "could not decode secret"))
	}

	secret := util.NewGCPSecretCR(string(privateKeyString), types.NamespacedName{
		Namespace: r.ProjectClaim.Spec.GCPCredentialSecret.Namespace,
		Name:      r.ProjectClaim.Spec.GCPCredentialSecret.Name,
	})

	r.logger.V(1).Info(fmt.Sprintf("Creating Secret %s in namespace %s", r.ProjectClaim.Spec.GCPCredentialSecret.Name, r.ProjectClaim.Spec.GCPCredentialSecret.Namespace))
	createErr := r.kubeClient.Create(context.TODO(), secret)
	if createErr != nil {
		return util.RequeueWithError(operrors.Wrap(createErr, fmt.Sprintf("could not create service account secret for %s", r.ProjectClaim.Spec.GCPCredentialSecret.Name)))
	}

	return util.ContinueProcessing()
}

func (r *ReferenceAdapter) deleteCredentials() error {
	secret := types.NamespacedName{
		Namespace: r.ProjectClaim.Spec.GCPCredentialSecret.Namespace,
		Name:      r.ProjectClaim.Spec.GCPCredentialSecret.Name,
	}
	r.logger.Info("Deleting Credentials")

	r.logger.V(2).Info("Check if the Secret exists")
	if util.SecretExists(r.kubeClient, secret.Name, secret.Namespace) {
		r.logger.V(2).Info("Getting Secret")
		key, err := util.GetSecret(r.kubeClient, secret.Name, secret.Namespace)
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
func (r *ReferenceAdapter) ensureClaimAvailabilityZonesSet() (util.OperationResult, error) {
	r.logger.V(1).Info("enter ensureClaimProjectIDSet")
	if len(r.ProjectClaim.Spec.AvailabilityZones) > 0 {
		return util.ContinueProcessing()
	}

	zones, err := r.gcpClient.ListAvailabilityZones(r.ProjectReference.Spec.GCPProjectID, r.ProjectClaim.Spec.Region)
	if err != nil {
		return r.handleAvailabilityZonesError(err)
	}
	conditions := &r.ProjectReference.Status.Conditions
	r.conditionManager.SetCondition(conditions, gcpv1alpha1.ConditionComputeApiReady, corev1.ConditionTrue, "QueryAvailabilityZonesSucceeded", "ComputeAPI ready, successfully queried availability zones")

	r.ProjectClaim.Spec.AvailabilityZones = zones
	err = r.kubeClient.Update(context.TODO(), r.ProjectClaim)
	if err != nil {
		return util.RequeueWithError(operrors.Wrap(err, "error updating ProjectClaim spec"))
	}
	// as the ProjectClaim is modified, we need to requeue
	return util.Requeue()
}

func (r *ReferenceAdapter) handleAvailabilityZonesError(err error) (util.OperationResult, error) {
	if !matchesComputeApiNotReadyError(err) {
		return util.RequeueWithError(err)
	}

	conditions := &r.ProjectReference.Status.Conditions
	apiCondition, found := r.conditionManager.FindCondition(conditions, gcpv1alpha1.ConditionComputeApiReady)
	tenMinutesAgo := metav1.NewTime(time.Now().Add(time.Duration(-10 * time.Minute)))
	if found && apiCondition.LastTransitionTime.Before(&tenMinutesAgo) {
		r.conditionManager.SetCondition(conditions, gcpv1alpha1.ConditionComputeApiReady, corev1.ConditionFalse, "QueryAvailabilityZonesFailed", "ComputeAPI not yet ready, couldn't query availability zones")
		_ = r.StatusUpdate()
		return util.RequeueWithError(err)
	}
	r.conditionManager.SetCondition(conditions, gcpv1alpha1.ConditionComputeApiReady, corev1.ConditionFalse, "QueryAvailabilityZonesFailed", "ComputeAPI not yet ready, couldn't query availability zones")
	if statusUpdateErr := r.StatusUpdate(); statusUpdateErr != nil {
		return util.RequeueWithError(statusUpdateErr)
	}
	r.logger.V(2).Info("Compute API not yet fully initialized. Retrying in 30 seconds.")
	return util.RequeueAfter(30*time.Second, nil)
}

func (r *ReferenceAdapter) ensureClaimProjectIDSet() bool {
	if r.ProjectClaim.Spec.GCPProjectID == "" {
		r.ProjectClaim.Spec.GCPProjectID = r.ProjectReference.Spec.GCPProjectID
		return true
	}

	return false
}

func EnsureProjectReferenceInitialized(r *ReferenceAdapter) (util.OperationResult, error) {
	if r.ProjectReference.Status.Conditions == nil {
		r.ProjectReference.Status.Conditions = []gcpv1alpha1.Condition{}
		err := r.StatusUpdate()
		if err != nil {
			return util.RequeueWithError(operrors.Wrap(err, "failed to initialize ProjectReference"))
		}
		return util.StopProcessing()
	}
	return util.ContinueProcessing()
}

// AddorUpdateBindingResponse contains the data that is returned by the AddOrUpdarteBindings function
type AddorUpdateBindingResponse struct {
	modified bool
	policy   *cloudresourcemanager.Policy
}

// AddOrUpdateBindings gets the policy and checks if the bindings match the required roles
func (r *ReferenceAdapter) AddOrUpdateBindings(serviceAccountEmail string, policies []string, memberType util.IamMemberType) (AddorUpdateBindingResponse, error) {
	policy, err := r.gcpClient.GetIamPolicy(r.ProjectReference.Spec.GCPProjectID)
	if err != nil {
		return AddorUpdateBindingResponse{}, err
	}

	//Checking if policy is modified
	newBindings, modified := util.AddOrUpdateBinding(policy.Bindings, policies, serviceAccountEmail, memberType)

	// add new bindings to policy
	policy.Bindings = newBindings
	return AddorUpdateBindingResponse{
		modified: modified,
		policy:   policy,
	}, nil
}

// SetIAMPolicy attempts to update policy if the policy needs to be modified
func (r *ReferenceAdapter) SetIAMPolicy(serviceAccountEmail string, policies []string, memberType util.IamMemberType) error {
	// Checking if policy needs to be updated
	var retry int
	for {

		addorUpdateResponse, err := r.AddOrUpdateBindings(serviceAccountEmail, policies, memberType)
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

				if ok && ae.Code == http.StatusConflict && retry < 3 {
					retry++
					time.Sleep(time.Second)
					continue
				}
				return err
			}
			return nil
		}
		return nil
	}
}

func (r *ReferenceAdapter) DeleteIAMPolicy(serviceAccountEmail string, memberType util.IamMemberType) error {
	// Checking if policy needs to be updated
	var retry int
	for {

		policies, err := r.gcpClient.GetIamPolicy(r.ProjectReference.Spec.GCPProjectID)
		if err != nil {
			return err
		}

		newBindings, modified := util.RemoveOrUpdateBinding(policies.Bindings, serviceAccountEmail, memberType)
		if !modified {
			return nil
		}

		policies.Bindings = newBindings
		setIamPolicyRequest := &cloudresourcemanager.SetIamPolicyRequest{
			Policy: policies,
		}
		_, err = r.gcpClient.SetIamPolicy(setIamPolicyRequest)
		if err != nil {
			ae, ok := err.(*googleapi.Error)
			// retry rules below:

			if ok && ae.Code == http.StatusConflict && retry < 3 {
				retry++
				time.Sleep(time.Second)
				continue
			}
			return err
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
		return operrors.Wrap(err, fmt.Sprintf("failed to update ProjectReference status of %s", r.ProjectReference.Name))
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
	return strings.HasPrefix(err.Error(), "googleapi: Error 404:") ||
		(strings.Contains(err.Error(), "400 Bad Request") && strings.Contains(err.Error(), "Invalid grant: account not found"))
}

func matchesComputeApiNotReadyError(err error) bool {
	return strings.HasPrefix(err.Error(), "googleapi: Error 403: Compute Engine API has not been used in project") ||
		strings.HasPrefix(err.Error(), "googleapi: Error 403: Access Not Configured. Compute Engine API has not been used in project")
}
