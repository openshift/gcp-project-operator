package clusterdeployment

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	hivev1alpha1 "github.com/openshift/hive/pkg/apis/hive/v1alpha1"
	"google.golang.org/api/cloudresourcemanager/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubetypes "k8s.io/apimachinery/pkg/types"
	kubeclientpkg "sigs.k8s.io/controller-runtime/pkg/client"
	//corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	//"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	//"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	"asia-east1":              false,
	"asia-east2":              false,
	"asia-northeast1":         false,
	"asia-northeast2":         false,
	"asia-south1":             false,
	"asia-southeast1":         false,
	"australia-southeast1":    false,
	"europe-north1":           false,
	"europe-west1":            false,
	"europe-west2":            false,
	"europe-west3 ":           false,
	"europe-west4":            false,
	"europe-west6":            false,
	"northamerica-northeast1": false,
	"southamerica-east1":      false,
	"us-central1":             false,
	"us-east1":                true,
	"us-east4":                false,
	"us-west1":                false,
	"us-west2":                false,
}

// Custom errors

// ErrRegionNotSupported indicates the region is not supported by OSD on GCP.
var ErrRegionNotSupported = errors.New("RegionNotSupported")

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

	// Do not make do anything if the cluster is not a GCP cluster.
	val, ok := cd.Labels[clusterPlatformLabel]
	if !ok || val != clusterPlatformGCP {
		reqLogger.Info("not a gcp cluster")
		return reconcile.Result{}, nil
	}

	// Do not do anything if the cluster is not a Red Hat managed cluster.
	val, ok = cd.Labels[clusterDeploymentManagedLabel]
	if !ok || val != "true" {
		reqLogger.Info("not a managed cluster")
		return reconcile.Result{}, nil
	}

	//Do not reconcile if cluster is installed or remove cleanup and remove project
	if cd.Spec.Installed {
		// TODO(Raf) Cleanup and remove project if being deleted once Hive is finished uninstalling
		reqLogger.Info(fmt.Sprintf("cluster %v is in installed state", cd.Name))
		return reconcile.Result{}, nil
	}

	// Get org creds from secret
	creds, err := GetOrgGCPCreds(r.client, operatorNamespace)
	if err != nil {
		reqLogger.Error(err, "Could not get org Creds from secret", "Secret Name", orgGcpSecretName, "Operator Namespace", operatorNamespace)
		return reconcile.Result{}, err
	}

	// Check if gcpSecretName in cd.Namesapce exists we are done
	if SecretExists(r.client, gcpSecretName, cd.Namespace){
		reqLogger.Info(fmt.Sprintf("Secret: %s already exists in Namespace: %s :: Nothing to do", gcpSecretName, cd.Namespace))
		return reconcile.Result{}, nil
	}

	// Skip code block to create project for now until we have permissions to test
	if false {
		// Check if platform projectID string exists.
		if cd.Spec.Platform.GCP.ProjectID != "" && cd.Spec.Platform.GCP.Region != "" {
			// check that region is supported
			if !supportedRegions[cd.Spec.Platform.GCP.Region] {
				reqLogger.Error(ErrRegionNotSupported, "Regions is not supported", "Region", cd.Spec.Platform.GCP.Region)
				// TODO(Raf) Should we be requeuing here or stop looping at this error
				return reconcile.Result{}, ErrRegionNotSupported
			}
			// Get gcpclient with creds
			gClient, err := gcpclient.NewClient(cd.Spec.GCP.ProjectID, creds)
			if err != nil {
				reqLogger.Error(err, "Could not get gcp client with secret creds", "Secret Name", orgGcpSecretName, "Operator Namespace", operatorNamespace)
				return reconcile.Result{}, err
			}

			// TODO(Raf) Check that operation is complete before continuing , make sure project Name does not exits , How to handle those errors
			_, err = gClient.CreateProject(orgParentFolderID)
			if err != nil {
				reqLogger.Error(err, "Could create project", "Parent Folder ID", orgParentFolderID, "Requested Project Name", cd.Spec.Platform.GCP.ProjectID, "Requested Region Name", cd.Spec.GCP.Region)
				return reconcile.Result{}, err
			}

			// TODO(Raf) Set quotas
			// TODO(Raf) Enable APIs
		} else {
			// TODO(Raf) Should we requeue in this case or not ?
			reqLogger.Error(err, "Could create project because cluster deployment does not have required fields", "Requested Project ID", cd.Spec.GCP.ProjectID, "Requested Region Name", cd.Spec.GCP.Region)
			return reconcile.Result{}, err
		}
	}

	gClient, err := gcpclient.NewClient(cd.Spec.GCP.ProjectID, creds)
	if err != nil {
		reqLogger.Error(err, "Could not get gcp client with secret creds", "Secret Name", orgGcpSecretName, "Operator Namespace", operatorNamespace)
		return reconcile.Result{}, err
	}

	// See if service account exists if not create it
	ServiceAccount, err := gClient.GetServiceAccount(osdServiceAccountName)
	if err != nil {
		// Create OSDManged Service account
		Account, err := gClient.CreateServiceAccount(osdServiceAccountName, osdServiceAccountName)
		if err != nil {
			reqLogger.Error(err, "Could create service account", "Service Account Name", osdServiceAccountName)
			return reconcile.Result{}, err
		}
		ServiceAccount = Account
	}

	// Configure policy
	// Get policy from project
	policy, err := gClient.GetIamPolicy()
	if err != nil {
		reqLogger.Error(err, "Could not get policy from project", "Project Name", cd.Spec.GCP.ProjectID)
		return reconcile.Result{}, err
	}

	// Create requiredBindings with the new member
	requiredBindings := GetOSDRequiredBindingMap(OSDRequiredRoles, ServiceAccount.Email)
	// Get combined bindings
	modified, newBindings := AddOrUpdateBinding(policy.Bindings, requiredBindings)
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
			reqLogger.Error(err, "Could not update policy on project", "Project Name", cd.Spec.GCP.ProjectID)
			return reconcile.Result{}, err
		}
	}

	// Delete service account keys if any exist
	err = gClient.DeleteServiceAccountKeys(ServiceAccount.Email)
	if err != nil {
		reqLogger.Error(err, "Could delete service account key", "Service Account Name", ServiceAccount.Email)
		return reconcile.Result{}, err
	}

	key, err := gClient.CreateServiceAccountKey(ServiceAccount.Email)
	if err != nil {
		reqLogger.Error(err, "Could create service account key", "Service Account Name", ServiceAccount.Email)
		return reconcile.Result{}, err
	}

	// Create secret for the key and store it
	privateKeyString, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		reqLogger.Error(err, "Could not decode secret")
		return reconcile.Result{}, err
	}

	secret := NewGcpSecretCR(cd.Namespace, string(privateKeyString))

	createErr := r.client.Create(context.TODO(), secret)
	if createErr != nil {
		reqLogger.Error(createErr, "Could not create service account cred secret ", "Service Account Secret Name", gcpSecretName)
		return reconcile.Result{}, createErr
	}

	return reconcile.Result{}, nil
}

func GetOrgGCPCreds(kubeClient kubeclientpkg.Client, namespace string) ([]byte, error) {
	secret := &corev1.Secret{}
	err := kubeClient.Get(context.TODO(),
		kubetypes.NamespacedName{
			Namespace: namespace,
			Name:      orgGcpSecretName,
		},
		secret)
	if err != nil {
		return []byte{}, fmt.Errorf("clusterdeployment.GetGCPClientFromSecret.Get %v", err)
	}

	osServiceAccountJson, ok := secret.Data[osServiceAccountKey]
	if !ok {
		return []byte{}, fmt.Errorf("GCP credentials secret %v did not contain key %v",
			orgGcpSecretName, osServiceAccountKey)
	}

	return osServiceAccountJson, nil
}

// GetOSDRequiredBindingMap returns a map of requiredBindings OSD role bindings for the added members
func GetOSDRequiredBindingMap(roles []string, members string) map[string]cloudresourcemanager.Binding {
	requiredBindings := make(map[string]cloudresourcemanager.Binding)
	for _, role := range roles {
		requiredBindings[role] = cloudresourcemanager.Binding{
			Members: []string{"serviceAccount:" + members},
			Role:    role,
		}
	}
	return requiredBindings
}

// NewGCPSecretCR returns a Secret CR formatted for GCP
func NewGcpSecretCR(namespace, creds string) *corev1.Secret {
	return &corev1.Secret{
		Type: "Opaque",
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      gcpSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"osServiceAccount.json": []byte(creds),
		},
	}
}

// AddOrUpdateBinding checks if a binding from a map of bindings whose keys are the binding.Role exists in a list and if so it appends any new members to that binding.
// If the required binding does not exist it creates a new binding for the role
// it returns a []*cloudresourcemanager.Binding that contains all the previous bindings and the new ones if no new bindings are required it returns false
func AddOrUpdateBinding(existingBindings []*cloudresourcemanager.Binding, requiredBindings map[string]cloudresourcemanager.Binding) (bool, []*cloudresourcemanager.Binding) {
	Modified := false

	for _, eBinding := range existingBindings {
		if rBinding, ok := requiredBindings[eBinding.Role]; ok {
			// check if members list contains from existing contains members from required
			for _, rMember := range rBinding.Members {
				if !stringInSlice(rMember, eBinding.Members) {
					Modified = true
					// If required member is not in existing member list add it
					eBinding.Members = append(eBinding.Members, rMember)
				}
			}
			// delete processed key from requiredBindings
			delete(requiredBindings, eBinding.Role)
		}

	}

	// take the remaining bindings from map of required bindings and append it to the list of existing bindings
	if len(requiredBindings) > 0 {
		Modified = true
		for _, binding := range requiredBindings {
			existingBindings = append(existingBindings, &binding)
		}
	}

	return Modified, existingBindings
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// TODO(Raf) Clean serviceAccount from member in bindings

func findMemberIndex(searchMember string, members []string) int {
	for index, value := range members {
		if value == searchMember {
			return index
		}
	}
	return -1
}

// remove removes a given index from a []string
func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

// SecretExists returns a boolean to the caller basd on the secretName and namespace args.
func SecretExists(kubeClient client.Client, secretName, namespace string) bool {

	s := &corev1.Secret{}

	err := kubeClient.Get(context.TODO(), kubetypes.NamespacedName{Name: secretName, Namespace: namespace}, s)
	if err != nil {
		return false
	}

	return true
}

// GetSecret returns a secret based on a secretName and namespace.
func GetSecret(kubeClient client.Client, secretName, namespace string) (*corev1.Secret, error) {

	s := &corev1.Secret{}

	err := kubeClient.Get(context.TODO(), kubetypes.NamespacedName{Name: secretName, Namespace: namespace}, s)

	if err != nil {
		return nil, err
	}
	return s, nil
}