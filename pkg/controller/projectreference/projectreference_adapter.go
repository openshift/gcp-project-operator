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
	clusterapi "github.com/openshift/cluster-api/pkg/util"
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	condition "github.com/openshift/gcp-project-operator/pkg/condition"
	"github.com/openshift/gcp-project-operator/pkg/configmap"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	gcputil "github.com/openshift/gcp-project-operator/pkg/util"
	operrors "github.com/openshift/gcp-project-operator/pkg/util/errors"
	logtypes "github.com/openshift/gcp-project-operator/pkg/util/types"
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

//ReferenceAdapter is used to do all the processing of the ProjectReference type inside the reconcile loop
type ReferenceAdapter struct {
	ProjectClaim     *gcpv1alpha1.ProjectClaim
	ProjectReference *gcpv1alpha1.ProjectReference
	logger           logr.Logger
	kubeClient       client.Client
	gcpClient        gcpclient.Client
	conditionManager condition.Conditions
}

type ensureAzResult int

const (
	ensureAzResultNotReady ensureAzResult = iota
	ensureAzResultModified
	ensureAzResultNoChange
)

// NewReferenceAdapter creates an adapter to turn what is requested in a ProjectReference into a GCP project and write the output back.
func NewReferenceAdapter(projectReference *gcpv1alpha1.ProjectReference, logger logr.Logger, client client.Client, gcpClient gcpclient.Client, manager condition.Conditions) (*ReferenceAdapter, error) {
	projectClaim, err := getMatchingClaimLink(projectReference, client)
	if err != nil {
		return &ReferenceAdapter{}, err
	}
	return &ReferenceAdapter{
		ProjectClaim:     projectClaim,
		ProjectReference: projectReference,
		logger:           logger,
		kubeClient:       client,
		gcpClient:        gcpClient,
		conditionManager: manager,
	}, nil
}

func (r *ReferenceAdapter) EnsureProjectClaimReady() (gcpv1alpha1.ClaimStatus, error) {
	if r.ProjectReference.Status.State != gcpv1alpha1.ProjectReferenceStatusReady {
		return r.ProjectClaim.Status.State, nil
	}

	if r.ProjectReference.Status.State == gcpv1alpha1.ProjectReferenceStatusReady && r.ProjectClaim.Status.State == gcpv1alpha1.ClaimStatusReady {
		return r.ProjectClaim.Status.State, nil
	}

	azResult, err := r.ensureClaimAvailabilityZonesSet()
	if err != nil {
		r.logger.Error(err, "Error ensuring availability zones")
		return r.ProjectClaim.Status.State, err
	}

	idModified := r.ensureClaimProjectIDSet()

	if azResult == ensureAzResultModified || idModified {
		err := r.kubeClient.Update(context.TODO(), r.ProjectClaim)
		if err != nil {
			r.logger.Error(err, "Error updating ProjectClaim Spec")
			return r.ProjectClaim.Status.State, err
		}
	}

	if azResult == ensureAzResultNotReady {
		return r.ProjectClaim.Status.State, nil
	}

	//Project Ready update matchingClaim to ready
	r.ProjectClaim.Status.State = gcpv1alpha1.ClaimStatusReady
	err = r.kubeClient.Status().Update(context.TODO(), r.ProjectClaim)
	if err != nil {
		r.logger.Error(err, "Error updating ProjectClaim Status")
		return r.ProjectClaim.Status.State, err
	}
	return r.ProjectClaim.Status.State, nil
}

func (r *ReferenceAdapter) EnsureProjectConfigured() error {
	configMap, err := r.getConfigMap()
	if err != nil {
		r.logger.Error(err, "could not get ConfigMap:", orgGcpConfigMap, "Operator Namespace", operatorNamespace)
		return err
	}

	err = r.createProject(configMap.ParentFolderID)
	if err != nil {
		if err == operrors.ErrInactiveProject {
			log.Error(err, "Unrecoverable Error")
			r.ProjectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusError
			err := r.kubeClient.Status().Update(context.TODO(), r.ProjectReference)
			if err != nil {
				r.logger.Error(err, "Error updating ProjectReference Status")
				return err
			}
		}
		r.logger.Error(err, "Could not create project")
		return err
	}

	r.logger.V(int(logtypes.ProjectReference)).Info("Configuring APIS")
	err = r.configureAPIS(configMap)
	if err != nil {
		r.logger.Error(err, "Error configuring APIS")
		return err
	}

	r.logger.V(int(logtypes.ProjectReference)).Info("Configuring Service Account")
	err = r.configureServiceAccount()
	if err != nil {
		r.logger.Error(err, "Error configuring service account")
		return err
	}

	r.logger.V(int(logtypes.ProjectReference)).Info("Creating Credentials")
	err = r.createCredentials()
	if err != nil {
		r.logger.Error(err, "Error creating credentials")
	}
	return err
}

func (r *ReferenceAdapter) EnsureStateReady() error {
	if r.ProjectReference.Status.State != gcpv1alpha1.ProjectReferenceStatusReady {
		r.logger.V(int(logtypes.ProjectReference)).Info("Setting Status on projectReference")
		r.ProjectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusReady
		return r.kubeClient.Status().Update(context.TODO(), r.ProjectReference)
	}
	return nil
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

// IsDeletionRequested checks the metadata.deletionTimestamp of ProjectReference instance, and returns if delete requested.
// The controllers watching the ProjectReference use this as a signal to know when to execute the finalizer.
func (r *ReferenceAdapter) IsDeletionRequested() bool {
	return r.ProjectReference.DeletionTimestamp != nil
}

// EnsureFinalizerAdded parses the meta.Finalizers of ProjectReference instance and adds FinalizerName if not found.
func (r *ReferenceAdapter) EnsureFinalizerAdded() error {
	if !clusterapi.Contains(r.ProjectReference.GetFinalizers(), FinalizerName) {
		r.ProjectReference.SetFinalizers(append(r.ProjectReference.GetFinalizers(), FinalizerName))
		return r.kubeClient.Update(context.TODO(), r.ProjectReference)
	}
	return nil
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
	err := r.deleteProject()
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
		r.logger.Info("Project lifecycleState == DELETE_REQUESTED")
		return nil
	case "LIFECYCLE_STATE_UNSPECIFIED":
		r.logger.Error(operrors.ErrUnexpectedLifecycleState, "Unexpected LifecycleState", project.LifecycleState)
		return operrors.ErrUnexpectedLifecycleState
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
			r.logger.V(int(logtypes.ProjectReference)).Info("Project lifecycleState == ACTIVE")
			return nil
		case "DELETE_REQUESTED":
			return operrors.ErrInactiveProject
		default:
			r.logger.Error(operrors.ErrUnexpectedLifecycleState, "Unexpected LifecycleState", project.LifecycleState)
			return operrors.ErrUnexpectedLifecycleState
		}
	}

	r.logger.V(int(logtypes.ProjectReference)).Info("Creating Project")
	// If we cannot create the project clear the projectID from spec so we can try again with another unique key
	_, err = r.gcpClient.CreateProject(parentFolderID)
	if err != nil {
		r.logger.Error(err, "could not create project", "Parent Folder ID", parentFolderID, "Requested Project ID", r.ProjectReference.Spec.GCPProjectID)
		r.logger.V(int(logtypes.ProjectReference)).Info("Clearing gcpProjectID from ProjectReferenceSpec")
		//Todo() We need to requeue here ot it will continue to the next step.
		err = r.clearProjectID()
		if err != nil {
			return err
		}
		return err
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

func (r *ReferenceAdapter) configureAPIS(config configmap.OperatorConfigMap) error {
	r.logger.V(int(logtypes.ProjectReference)).Info("Enabling Billing API")
	err := r.gcpClient.EnableAPI(r.ProjectReference.Spec.GCPProjectID, "cloudbilling.googleapis.com")
	if err != nil {
		r.logger.Error(err, fmt.Sprintf("Error enabling %s api for project %s", "cloudbilling.googleapis.com", r.ProjectReference.Spec.GCPProjectID))
		return err
	}

	r.logger.V(int(logtypes.ProjectReference)).Info("Linking Cloud Billing Account")
	err = r.gcpClient.CreateCloudBillingAccount(r.ProjectReference.Spec.GCPProjectID, config.BillingAccount)
	if err != nil {
		r.logger.Error(err, "error creating CloudBilling")
		return err
	}

	for _, a := range OSDRequiredAPIS {
		err = r.gcpClient.EnableAPI(r.ProjectReference.Spec.GCPProjectID, a)
		if err != nil {
			r.logger.Error(err, fmt.Sprintf("error enabling %s api for project %s", a, r.ProjectReference.Spec.GCPProjectID))
			return err
		}
	}

	return nil
}

func (r *ReferenceAdapter) getConfigMap() (configmap.OperatorConfigMap, error) {
	operatorConfigMap, err := configmap.GetOperatorConfigMap(r.kubeClient)
	if err != nil {
		r.logger.Error(err, "could not find the OperatorConfigMap")
		return operatorConfigMap, err
	}

	if err := configmap.ValidateOperatorConfigMap(operatorConfigMap); err != nil {
		r.logger.Error(err, "configmap didn't get filled properly")
		return operatorConfigMap, err
	}
	return operatorConfigMap, err
}

func (r *ReferenceAdapter) configureServiceAccount() error {
	// See if GCP service account exists if not create it
	var serviceAccount *iam.ServiceAccount
	serviceAccount, err := r.gcpClient.GetServiceAccount(osdServiceAccountName)
	if err != nil {
		// Create OSDManged Service account
		r.logger.V(int(logtypes.ProjectReference)).Info("Creating Service Account")
		account, err := r.gcpClient.CreateServiceAccount(osdServiceAccountName, osdServiceAccountName)
		if err != nil {
			r.logger.Error(err, "could not create service account", "Service Account Name", osdServiceAccountName)
			return err
		}
		serviceAccount = account
	}

	r.logger.V(int(logtypes.ProjectReference)).Info("Setting Service Account Policies")
	err = r.SetIAMPolicy(serviceAccount.Email)
	if err != nil {
		r.logger.Error(err, "could not update policy on project", "Project Name", r.ProjectReference.Spec.GCPProjectID)
		return err
	}
	return nil
}

func (r *ReferenceAdapter) createCredentials() error {
	var serviceAccount *iam.ServiceAccount
	serviceAccount, err := r.gcpClient.GetServiceAccount(osdServiceAccountName)
	if err != nil {
		r.logger.Error(err, "could not get service account")
		return err
	}

	r.logger.V(int(logtypes.ProjectReference)).Info("Creating Service AccountKey")
	key, err := r.gcpClient.CreateServiceAccountKey(serviceAccount.Email)
	if err != nil {
		r.logger.Error(err, "could not create service account key", "Service Account Name", serviceAccount.Email)
		return err
	}

	// Create secret for the key and store it
	privateKeyString, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		r.logger.Error(err, "could not decode secret")
		return err
	}

	secret := gcputil.NewGCPSecretCR(string(privateKeyString), types.NamespacedName{
		Namespace: r.ProjectClaim.Spec.GCPCredentialSecret.Namespace,
		Name:      r.ProjectClaim.Spec.GCPCredentialSecret.Name,
	})

	r.logger.V(int(logtypes.ProjectReference)).Info(fmt.Sprintf("Creating Secret %s in namespace %s", r.ProjectClaim.Spec.GCPCredentialSecret.Name, r.ProjectClaim.Spec.GCPCredentialSecret.Namespace))
	createErr := r.kubeClient.Create(context.TODO(), secret)
	if createErr != nil {
		r.logger.Error(createErr, "could not create service account secret ", "Service Account Secret Name", r.ProjectClaim.Spec.GCPCredentialSecret.Name)
		return createErr
	}

	return nil
}

func (r *ReferenceAdapter) deleteCredentials() error {
	r.logger.Info("Deleting Credentials")
	secret := types.NamespacedName{
		Namespace: r.ProjectClaim.Spec.GCPCredentialSecret.Namespace,
		Name:      r.ProjectClaim.Spec.GCPCredentialSecret.Name,
	}

	// Check if the Secret exists
	if gcputil.SecretExists(r.kubeClient, secret.Name, secret.Namespace) {
		// Get the secret key
		key, err := gcputil.GetSecret(r.kubeClient, secret.Name, secret.Namespace)
		if err != nil {
			r.logger.Error(err, "could not get the service account secret ", "Service Account Secret Name", secret.Name)
			return err
		}

		// Delete the secret
		err = r.kubeClient.Delete(context.TODO(), key)
		if err != nil {
			r.logger.Error(err, "could not delete service account secret ", "Service Account Secret Name", secret.Name)
			return err
		}
	}

	return nil
}

// ensureAvailibilityZonesSet sets the az in the projectclaim spec if necessary
// returns true if the project claim has been modified
func (r *ReferenceAdapter) ensureClaimAvailabilityZonesSet() (ensureAzResult, error) {
	if len(r.ProjectClaim.Spec.AvailibilityZones) > 0 {
		return ensureAzResultNoChange, nil
	}

	if len(r.ProjectClaim.Spec.AvailabilityZones) > 0 {
		return ensureAzResultNoChange, nil
	}

	zones, err := r.gcpClient.ListAvilibilityZones(r.ProjectReference.Spec.GCPProjectID, r.ProjectClaim.Spec.Region)
	if err != nil {
		return r.handleAvailabilityZonesError(err)
	}
	conditions := &r.ProjectReference.Status.Conditions
	r.conditionManager.SetCondition(conditions, gcpv1alpha1.ConditionComputeApiReady, corev1.ConditionTrue, "QueryAvailabilityZonesSucceeded", "ComputeAPI ready, successfully queried availability zones")

	r.ProjectClaim.Spec.AvailibilityZones = zones
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

func matchesComputeApiNotReadyError(err error) bool {
	return strings.HasPrefix(err.Error(), "googleapi: Error 403: Access Not Configured. Compute Engine API has not been used in project")
}
func (r *ReferenceAdapter) ensureClaimProjectIDSet() bool {
	if r.ProjectClaim.Spec.GCPProjectID == "" {
		r.ProjectClaim.Spec.GCPProjectID = r.ProjectReference.Spec.GCPProjectID
		return true
	}

	return false
}

func (r *ReferenceAdapter) EnsureProjectReferenceInitialized() (ObjectState, error) {
	if r.ProjectReference.Status.Conditions == nil {
		r.ProjectReference.Status.Conditions = []gcpv1alpha1.Condition{}
		err := r.StatusUpdate()
		if err != nil {
			r.logger.Error(err, "Failed to initalize ProjectReference")
			return ObjectUnchanged, err
		}
		return ObjectModified, nil
	}
	return ObjectUnchanged, nil
}

// AddorUpdateBindingResponse contines the data that is returned by the AddOrUpdarteBindings function
type AddorUpdateBindingResponse struct {
	modified bool
	policy   *cloudresourcemanager.Policy
}

// AddOrUpdateBindings gets the policy and checks if the bindings match the required roles
func (r *ReferenceAdapter) AddOrUpdateBindings(serviceAccountEmail string) (AddorUpdateBindingResponse, error) {
	policy, err := r.gcpClient.GetIamPolicy(r.ProjectReference.Spec.GCPProjectID)
	if err != nil {
		return AddorUpdateBindingResponse{}, err
	}

	//Checking if policy is modified
	newBindings, modified := gcputil.AddOrUpdateBinding(policy.Bindings, OSDRequiredRoles, serviceAccountEmail)

	// add new bindings to policy
	policy.Bindings = newBindings
	return AddorUpdateBindingResponse{
		modified: modified,
		policy:   policy,
	}, nil
}

// SetIAMPolicy attempts to update policy if the policy needs to be modified
func (r *ReferenceAdapter) SetIAMPolicy(serviceAccountEmail string) error {
	// Checking if policy needs to be updated
	var retry int
	for {
		retry++
		time.Sleep(time.Second)
		addorUpdateResponse, err := r.AddOrUpdateBindings(serviceAccountEmail)
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
		r.logger.Error(err, fmt.Sprintf("failed to update ProjectClaim state for %s", r.ProjectReference.Name))
		return err
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
