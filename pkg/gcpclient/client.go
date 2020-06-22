package gcpclient

//go:generate mockgen -destination=../util/mocks/$GOPACKAGE/$GOFILE -package=$GOPACKAGE -source $GOFILE
//go:generate gofmt -s -l -w ../util/mocks/$GOPACKAGE/$GOFILE
//go:generate goimports -local=github.com/openshift/gcp-account-operator -e -w ../util/mocks/$GOPACKAGE/$GOFILE

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	logtypes "github.com/openshift/gcp-project-operator/pkg/util/types"
	"golang.org/x/oauth2/google"
	cloudbilling "google.golang.org/api/cloudbilling/v1"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	compute "google.golang.org/api/compute/v1"
	dns "google.golang.org/api/dns/v1"
	"google.golang.org/api/googleapi"
	iam "google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	serviceManagment "google.golang.org/api/servicemanagement/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("gcpclient")

const gcpAPIRetriesCount = 3

// Client is a wrapper object for actual GCP libraries to allow for easier mocking/testing.
type Client interface {
	// IAM
	GetServiceAccount(accountName string) (*iam.ServiceAccount, error)
	CreateServiceAccount(name, displayName string) (*iam.ServiceAccount, error)
	DeleteServiceAccount(accountEmail string) error
	CreateServiceAccountKey(serviceAccountEmail string) (*iam.ServiceAccountKey, error)
	DeleteServiceAccountKeys(serviceAccountEmail string) error
	// Cloudresourcemanager
	GetIamPolicy(projectName string) (*cloudresourcemanager.Policy, error)
	SetIamPolicy(setIamPolicyRequest *cloudresourcemanager.SetIamPolicyRequest) (*cloudresourcemanager.Policy, error)
	ListProjects() ([]*cloudresourcemanager.Project, error)
	CreateProject(parentFolder string) (*cloudresourcemanager.Operation, error)
	DeleteProject(parentFolder string) (*cloudresourcemanager.Empty, error)
	GetProject(projectID string) (*cloudresourcemanager.Project, error)

	// ServiceManagement
	EnableAPI(projectID, api string) error

	// CloudBilling
	CreateCloudBillingAccount(projectID, billingAccount string) error

	//Compute
	ListAvailabilityZones(projectID, region string) ([]string, error)
}

type gcpClient struct {
	projectName                string
	creds                      *google.Credentials
	cloudResourceManagerClient *cloudresourcemanager.Service
	iamClient                  *iam.Service
	dnsClient                  *dns.Service
	serviceManagmentClient     *serviceManagment.APIService
	cloudBillingClient         *cloudbilling.APIService
	computeClient              *compute.Service
	// Some actions requires new individual client to be
	// initiated. we try to re-use clients, but we store
	// credentials for these methods
	credentials *google.Credentials
}

// NewClient creates our client wrapper object for interacting with GCP.
func NewClient(projectName string, authJSON []byte) (Client, error) {
	ctx := context.TODO()

	// since we're using a single creds var, we should specify all the required scopes when initializing
	creds, err := google.CredentialsFromJSON(ctx, authJSON, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("gcpclient.NewClient.google.CredentialsFromJSON %v", err)
	}

	cloudResourceManagerClient, err := cloudresourcemanager.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("gcpclient.NewClient.cloudresourcemanager.NewService %v", err)
	}

	iamClient, err := iam.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("gcpclient.iam.NewService %v", err)
	}

	serviceManagmentClient, err := serviceManagment.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("gcpclient.serviceManagement.NewService %v", err)
	}

	cloudBillingClient, err := cloudbilling.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("gcpclient.cloudBillingClient.NewService %v", err)
	}

	computeService, err := compute.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("gcpclient.compute.NewService %v", err)
	}

	return &gcpClient{
		projectName:                projectName,
		creds:                      creds,
		cloudResourceManagerClient: cloudResourceManagerClient,
		iamClient:                  iamClient,
		serviceManagmentClient:     serviceManagmentClient,
		cloudBillingClient:         cloudBillingClient,
		computeClient:              computeService,
		credentials:                creds,
	}, nil
}

// ListAvailabilityZones returns a map of all availability zones a project has access to
// where the key is the region and the values is a list of zones
func (c *gcpClient) ListAvailabilityZones(projectID, region string) ([]string, error) {

	zones := []string{}
	req := c.computeClient.Zones.List(projectID)
	err := req.Pages(context.Background(), func(page *compute.ZoneList) error {
		for _, zone := range page.Items {
			if strings.Contains(zone.Region, region) {
				zones = append(zones, zone.Name)
			}
		}
		return nil
	})
	if err != nil {
		return []string{}, err
	}

	return zones, nil
}

// ListProjects returns a list of all projects
func (c *gcpClient) ListProjects() ([]*cloudresourcemanager.Project, error) {
	resp, err := c.cloudResourceManagerClient.Projects.List().Do()
	if err != nil {
		return []*cloudresourcemanager.Project{}, err
	}
	return resp.Projects, nil
}

// GetProject returns a project
func (c *gcpClient) GetProject(projectID string) (*cloudresourcemanager.Project, error) {
	project, err := c.cloudResourceManagerClient.Projects.Get(projectID).Do()
	if err != nil {
		return nil, err
	}
	return project, nil
}

// CreateProject creates a project in a given folder.
func (c *gcpClient) CreateProject(parentFolderID string) (*cloudresourcemanager.Operation, error) {
	log.V(logtypes.GCPClient).Info("gcpClient.CreateProject")
	project := cloudresourcemanager.Project{
		//Labels:          nil,
		Name: c.projectName,
		Parent: &cloudresourcemanager.ResourceId{
			Id:   parentFolderID,
			Type: "folder",
		},
		ProjectId: c.projectName,
	}
	operation, err := c.cloudResourceManagerClient.Projects.Create(&project).Do()
	if err != nil {
		ae, ok := err.(*googleapi.Error)
		// google uses 409 for "already exists"
		// we continue as it was created
		if ok && ae.Code == http.StatusConflict {
			return &cloudresourcemanager.Operation{}, nil
		}
		return &cloudresourcemanager.Operation{}, fmt.Errorf("gcpclient.CreateProject.Projects.Create %v", err)
	}
	return operation, nil
}

// DeleteProject deletes a project from a given folder.
func (c *gcpClient) DeleteProject(parentFolder string) (*cloudresourcemanager.Empty, error) {
	empty, err := c.cloudResourceManagerClient.Projects.Delete(c.projectName).Do()
	if err != nil {
		return &cloudresourcemanager.Empty{}, fmt.Errorf("gcpclient.DeleteProject.Projects.Delete %v", err)
	}
	return empty, nil
}

// GetServiceAccount returns a service account if it exists
func (c *gcpClient) GetServiceAccount(accountName string) (*iam.ServiceAccount, error) {
	resource := fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", c.projectName, accountName, c.projectName)
	sa, err := c.iamClient.Projects.ServiceAccounts.Get(resource).Do()
	if err != nil {
		return &iam.ServiceAccount{}, fmt.Errorf("gcpclient.GetServiceAccount.Projects.ServiceAccounts.Get %v", err)
	}

	return sa, nil
}

// CreateServiceAccount creates a service account with required roles.
func (c *gcpClient) CreateServiceAccount(name, displayName string) (*iam.ServiceAccount, error) {
	CreateServiceAccountRequest := &iam.CreateServiceAccountRequest{
		AccountId: name,
		ServiceAccount: &iam.ServiceAccount{
			DisplayName: displayName,
		},
	}

	serviceAccount, err := c.iamClient.Projects.ServiceAccounts.Create(fmt.Sprintf("projects/%s", c.projectName), CreateServiceAccountRequest).Do()
	if err != nil {
		return &iam.ServiceAccount{}, fmt.Errorf("gcpclient.CreateServiceAccount.Projects.ServiceAccounts.Create %v", err)
	}

	return serviceAccount, nil
}

func (c *gcpClient) DeleteServiceAccount(accountEmail string) error {
	_, err := c.iamClient.Projects.ServiceAccounts.Delete(fmt.Sprintf("projects/%s/serviceAccounts/%s", c.projectName, accountEmail)).Do()
	if err != nil {
		return fmt.Errorf("gcpclient.DeleteServiceAccount.Projects.ServiceAccounts.Delete: %v", err)
	}

	return nil
}

func (c *gcpClient) CreateServiceAccountKey(serviceAccountEmail string) (*iam.ServiceAccountKey, error) {
	key, err := c.iamClient.Projects.ServiceAccounts.Keys.Create(fmt.Sprintf("projects/%s/serviceAccounts/%s", c.projectName, serviceAccountEmail), &iam.CreateServiceAccountKeyRequest{}).Do()
	if err != nil {
		return &iam.ServiceAccountKey{}, fmt.Errorf("gcpclient.CreateServiceAccountKey.Projects.ServiceAccounts.Keys.Create: %v", err)
	}
	return key, nil
}

//DeleteServiceAccountKeys deletes all keys associated with the service account
func (c *gcpClient) DeleteServiceAccountKeys(serviceAccountEmail string) error {
	resource := fmt.Sprintf("projects/%s/serviceAccounts/%s", c.projectName, serviceAccountEmail)
	response, err := c.iamClient.Projects.ServiceAccounts.Keys.List(resource).Do()
	if err != nil {
		return fmt.Errorf("gcpclient.DeleteServiceAccountKeys.Projects.ServiceAccounts.Keys.List: %v", err)
	}

	if len(response.Keys) <= 1 {
		return nil
	}

	for _, key := range response.Keys {
		_, err = c.iamClient.Projects.ServiceAccounts.Keys.Delete(key.Name).Do()
	}

	// ensure only one key exits
	newResponse, err := c.iamClient.Projects.ServiceAccounts.Keys.List(resource).Do()
	if err != nil {
		return fmt.Errorf("gcpclient.DeleteServiceAccountKeys.Projects.ServiceAccounts.Keys.List: %v", err)
	}

	if len(newResponse.Keys) > 1 {
		return fmt.Errorf("gcpclient.DeleteServiceAccountKeys.Projects.ServiceAccounts.Keys.Delete: %v", errors.New("Could not delete all keys"))
	}

	return nil
}

func (c *gcpClient) GetIamPolicy(projectName string) (*cloudresourcemanager.Policy, error) {
	policy, err := c.cloudResourceManagerClient.Projects.GetIamPolicy(projectName, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if err != nil {
		return nil, fmt.Errorf("gcpclient.GetIamPolicy.Projects.ServiceAccounts.GetIamPolicy %v", err)
	}

	return policy, nil
}

func (c *gcpClient) SetIamPolicy(setIamPolicyRequest *cloudresourcemanager.SetIamPolicyRequest) (*cloudresourcemanager.Policy, error) {
	policy, err := c.cloudResourceManagerClient.Projects.SetIamPolicy(c.projectName, setIamPolicyRequest).Do()
	if err != nil {
		return &cloudresourcemanager.Policy{}, err
	}
	return policy, nil
}

func (c *gcpClient) EnableAPI(projectID, api string) error {
	log.V(logtypes.GCPClient).Info(fmt.Sprintf("enable %s api", api))
	enableServicerequest := &serviceManagment.EnableServiceRequest{
		ConsumerId: fmt.Sprintf("project:%s", projectID),
	}

	req := c.serviceManagmentClient.Services.Enable(api, enableServicerequest)

	var retry int
	for {
		retry++
		time.Sleep(time.Second)

		_, err := req.Do()
		if err != nil {
			ae, ok := err.(*googleapi.Error)
			// Retry rules below:

			// sometimes we get 403 - Permission denied when even project
			// creation is completed and marked as Done.
			// Something is not propagating in the backend.
			if ok && ae.Code == http.StatusForbidden && retry <= gcpAPIRetriesCount {
				log.V(logtypes.GCPClient).Info(fmt.Sprintf("retry %d for enable %s api", retry, api))
				continue
			}
			return err
		}
		return nil
	}
}

// CreateCloudBillingAccount associates cloud billing account with project
// TODO: This needs unit testing. Sensitive place
func (c *gcpClient) CreateCloudBillingAccount(projectID, billingAccountID string) error {
	project := fmt.Sprintf("projects/%s", projectID)
	billingAccount := fmt.Sprintf("billingAccounts/%s", strings.TrimSuffix(billingAccountID, "\n"))
	info, err := c.cloudBillingClient.Projects.GetBillingInfo(project).Do()
	if err != nil {
		return err
	}

	// if we dont have set billing account
	if len(info.BillingAccountName) == 0 {
		info.BillingAccountName = billingAccount
		info.BillingEnabled = true
		_, err := c.cloudBillingClient.Projects.UpdateBillingInfo(project, info).Do()
		if err != nil {
			return err
		}
	}
	if len(info.BillingAccountName) > 0 && info.BillingAccountName != billingAccount {
		info.BillingAccountName = billingAccount
		projectBillingDisable := &cloudbilling.ProjectBillingInfo{
			BillingAccountName: "",
			BillingEnabled:     false,
		}
		_, err := c.cloudBillingClient.Projects.UpdateBillingInfo(project, projectBillingDisable).Do()
		if err != nil {
			return err
		}
		_, err = c.cloudBillingClient.Projects.UpdateBillingInfo(project, info).Do()
		if err != nil {
			return err
		}
	}

	return nil
}
