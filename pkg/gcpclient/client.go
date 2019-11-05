package gcpclient

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/oauth2/google"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	iam "google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

// Client is a wrapper object for actual GCP libraries to allow for easier mocking/testing.
type Client interface {
	// IAM
	GetServiceAccount(accountName string) (*iam.ServiceAccount, error)
	CreateServiceAccount(name, displayName string) (*iam.ServiceAccount, error)
	DeleteServiceAccount(accountEmail string) error
	CreateServiceAccountKey(serviceAccountEmail string) (*iam.ServiceAccountKey, error)
	DeleteServiceAccountKeys(serviceAccountEmail string) error
	// Cloudresourcemanager
	GetIamPolicy() (*cloudresourcemanager.Policy, error)
	SetIamPolicy(setIamPolicyRequest *cloudresourcemanager.SetIamPolicyRequest) (*cloudresourcemanager.Policy, error)
	CreateProject(parentFolder string) (*cloudresourcemanager.Operation, error)
	DeleteProject(parentFolder string) (*cloudresourcemanager.Empty, error)
}

type gcpClient struct {
	projectName                string
	creds                      *google.Credentials
	cloudResourceManagerClient *cloudresourcemanager.Service
	iamClient                  *iam.Service
}

// NewClient creates our client wrapper object for interacting with GCP.
func NewClient(projectName string, authJSON []byte) (Client, error) {
	ctx := context.TODO()

	// since we're using a single creds var, we should specify all the required scopes when initializing
	creds, err := google.CredentialsFromJSON(context.TODO(), authJSON, "https://www.googleapis.com/auth/cloud-platform")
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

	return &gcpClient{
		projectName:                projectName,
		creds:                      creds,
		cloudResourceManagerClient: cloudResourceManagerClient,
		iamClient:                  iamClient,
	}, nil
}

// CreateProject creates a project in a given folder.
func (c *gcpClient) CreateProject(parentFolderID string) (*cloudresourcemanager.Operation, error) {
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

// TODO(Raf) SetQuotas
// TODO(Raf) Enable APIs

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

func (c *gcpClient) GetIamPolicy() (*cloudresourcemanager.Policy, error) {
	policy, err := c.cloudResourceManagerClient.Projects.GetIamPolicy(c.projectName, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if err != nil {
		return nil, fmt.Errorf("gcpclient.GetIamPolicy.Projects.ServiceAccounts.GetIamPolicy %v", err)
	}

	return policy, nil
}

func (c *gcpClient) SetIamPolicy(setIamPolicyRequest *cloudresourcemanager.SetIamPolicyRequest) (*cloudresourcemanager.Policy, error) {
	policy, err := c.cloudResourceManagerClient.Projects.SetIamPolicy(c.projectName, setIamPolicyRequest).Do()
	if err != nil {
		return &cloudresourcemanager.Policy{}, fmt.Errorf("gcpclient.SetIamPolicy.Projects.ServiceAccounts.SetIamPolicy %v", err)
	}
	return policy, nil
}
