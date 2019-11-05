package clusterdeployment

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	hivev1alpha1 "github.com/openshift/hive/pkg/apis/hive/v1alpha1"
	"google.golang.org/api/cloudresourcemanager/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_clusterdeployment")

const (
	// Operator config
	operatorNamespace = "gcp-project-operator"
	controllerName    = "clusterdeployment"

	// clusterDeploymentManagedLabel is the label on the cluster deployment which indicates whether or not a cluster is OSD
	clusterDeploymentManagedLabel = "api.openshift.com/managed"
	// clusterPlatformLabel is the label on a cluster deployment which indicates whether or not a cluster is on GCP platform
	clusterPlatformLabel = "hive.openshift.io/cluster-platform"
	clusterPlatformGCP   = "gcp"
	// TODO(Raf) get name of org parent folder and ensure it exists
	orgParentFolderID = ""

	// secret information
	gcpSecretName       = "gcp"
	orgGcpSecretName    = "gcp-project-operator-creds"
	osServiceAccountKey = "osServiceAccount.json"
	//
	osdServiceAccountName = "osdmangedadmin"
)

var OSDRequiredRoles = []string{
	"roles/storage.admin",
	"roles/iam.serviceAccountUser",
	"roles/iam.serviceAccountKeyAdmin",
	"roles/iam.serviceAccountAdmin",
	"roles/iam.securityAdmin",
	"roles/dns.admin",
	"roles/compute.admin",
}

var supportedRegions = map[string]bool{
	"us-east1": true,
}

// Custom errors

// ErrRegionNotSupported indicates the region is not supported by OSD on GCP.
var ErrRegionNotSupported = errors.New("RegionNotSupported")

// ErrNotGCPCluster indicates that the cluster is not a gcp cluster
var ErrNotGCPCluster = errors.New("NotGCPCluster")

// ErrNotManagedCluster indicates this is not an OSD managed cluster
var ErrNotManagedCluster = errors.New("NotManagedCluster")

// ErrClusterInstalled indicates the cluster is already installed
var ErrClusterInstalled = errors.New("ClusterInstalled")

// ErrMissingProjectID indicates that the cluster deployment is missing the field ProjectID
var ErrMissingProjectID = errors.New("MissingProjectID")

// ErrMissingRegion indicates that the cluster deployment is missing the field Region
var ErrMissingRegion = errors.New("MissingRegion")

// Add creates a new ClusterDeployment Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileClusterDeployment{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterdeployment-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ClusterDeployment
	err = c.Watch(&source.Kind{Type: &hivev1alpha1.ClusterDeployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileClusterDeployment implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileClusterDeployment{}

// ReconcileClusterDeployment reconciles a ClusterDeployment object
type ReconcileClusterDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ClusterDeployment object and makes changes based on the state read
// and what is in the ClusterDeployment.Spec
// TODO(Raf) Add finalizers and clean up
func (r *ReconcileClusterDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ClusterDeployment")

	// Fetch the ClusterDeployment instance
	cd := &hivev1alpha1.ClusterDeployment{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cd)
	if err != nil {
		if k8serr.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	err = checkDeploymentConfigRequirements(cd)
	switch err {
	case nil:
		break
	case ErrMissingRegion, ErrMissingProjectID, ErrRegionNotSupported:
		reqLogger.Error(err, "clusterDeployment failed validation", "Validation Error", err)
		return reconcile.Result{}, err
	case ErrClusterInstalled:
		// TODO(Raf) Cleanup and remove project if being deleted once Hive is finished uninstalling
		reqLogger.Info(fmt.Sprintf("cluster %v is in installed state", cd.Name))
		return reconcile.Result{}, nil
	default:
		reqLogger.Info(fmt.Sprintf("clusterDeployment failed validation due to Error:%s", err))
		return reconcile.Result{}, nil
	}

	// Get org creds from secret
	creds, err := getOrgGCPCreds(r.client, operatorNamespace)
	if err != nil {
		reqLogger.Error(err, "could not get org Creds from secret", "Secret Name", orgGcpSecretName, "Operator Namespace", operatorNamespace)
		return reconcile.Result{}, err
	}

	// Check if gcpSecretName in cd.Namespace exists we are done
	// TODO(Raf) check if secret is a valid gcp secret
	if secretExists(r.client, gcpSecretName, cd.Namespace) {
		reqLogger.Info(fmt.Sprintf("secret: %s already exists in Namespace: %s :: Nothing to do", gcpSecretName, cd.Namespace))
		return reconcile.Result{}, nil
	}

	// Skip code block to create project for now until we have permissions to test
	if false {
		// Get gcpclient with creds
		gClient, err := gcpclient.NewClient(cd.Spec.GCP.ProjectID, creds)
		if err != nil {
			reqLogger.Error(err, "could not get gcp client with secret creds", "Secret Name", orgGcpSecretName, "Operator Namespace", operatorNamespace)
			return reconcile.Result{}, err
		}

		// TODO(Raf) Check that operation is complete before continuing , make sure project Name does not exits , How to handle those errors
		_, err = gClient.CreateProject(orgParentFolderID)
		if err != nil {
			reqLogger.Error(err, "could create project", "Parent Folder ID", orgParentFolderID, "Requested Project Name", cd.Spec.Platform.GCP.ProjectID, "Requested Region Name", cd.Spec.GCP.Region)
			return reconcile.Result{}, err
		}

		// TODO(Raf) Set quotas
		// TODO(Raf) Enable APIs
	}

	gClient, err := gcpclient.NewClient(cd.Spec.GCP.ProjectID, creds)
	if err != nil {
		reqLogger.Error(err, "could not get gcp client with secret creds", "Secret Name", orgGcpSecretName, "Operator Namespace", operatorNamespace)
		return reconcile.Result{}, err
	}

	// See if GCP service account exists if not create it
	ServiceAccount, err := gClient.GetServiceAccount(osdServiceAccountName)
	if err != nil {
		// Create OSDManged Service account
		Account, err := gClient.CreateServiceAccount(osdServiceAccountName, osdServiceAccountName)
		if err != nil {
			reqLogger.Error(err, "could create service account", "Service Account Name", osdServiceAccountName)
			return reconcile.Result{}, err
		}
		ServiceAccount = Account
	}

	// Configure policy
	// Get policy from project
	policy, err := gClient.GetIamPolicy()
	if err != nil {
		reqLogger.Error(err, "could not get policy from project", "Project Name", cd.Spec.GCP.ProjectID)
		return reconcile.Result{}, err
	}

	// Create requiredBindings with the new member
	requiredBindings := getOSDRequiredBindingMap(OSDRequiredRoles, ServiceAccount.Email)
	// Get combined bindings
	newBindings, modified := addOrUpdateBinding(policy.Bindings, requiredBindings)
	// If existing bindings have been modified update the policy
	if modified {
		// update policy
		policy.Bindings = newBindings

		setIamPolicyRequest := &cloudresourcemanager.SetIamPolicyRequest{
			Policy: policy,
		}

		//TODO(Raf) Set Etag in policy to version policies so we get the latest always
		_, err = gClient.SetIamPolicy(setIamPolicyRequest)
		if err != nil {
			reqLogger.Error(err, "could not update policy on project", "Project Name", cd.Spec.GCP.ProjectID)
			return reconcile.Result{}, err
		}
	}

	// Delete service account keys if any exist
	err = gClient.DeleteServiceAccountKeys(ServiceAccount.Email)
	if err != nil {
		reqLogger.Error(err, "could delete service account key", "Service Account Name", ServiceAccount.Email)
		return reconcile.Result{}, err
	}

	key, err := gClient.CreateServiceAccountKey(ServiceAccount.Email)
	if err != nil {
		reqLogger.Error(err, "could create service account key", "Service Account Name", ServiceAccount.Email)
		return reconcile.Result{}, err
	}

	// Create secret for the key and store it
	privateKeyString, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		reqLogger.Error(err, "could not decode secret")
		return reconcile.Result{}, err
	}

	secret := newGCPSecretCR(cd.Namespace, string(privateKeyString))

	createErr := r.client.Create(context.TODO(), secret)
	if createErr != nil {
		reqLogger.Error(createErr, "could not create service account cred secret ", "Service Account Secret Name", gcpSecretName)
		return reconcile.Result{}, createErr
	}

	return reconcile.Result{}, nil
}

// TODO(Raf) Clean serviceAccount from member in bindings
