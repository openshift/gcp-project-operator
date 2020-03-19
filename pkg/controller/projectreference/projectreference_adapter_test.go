package projectreference

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	api "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	mocks "github.com/openshift/gcp-project-operator/pkg/util/mocks"
	mockGCP "github.com/openshift/gcp-project-operator/pkg/util/mocks/gcpclient"
	testStructs "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
)

var _ = Describe("ProjectreferenceAdapter", func() {
	var (
		adapter          *ReferenceAdapter
		projectReference *api.ProjectReference
		mockKubeClient   *mocks.MockClient
		mockGCPClient    *mockGCP.MockClient
		mockStatusWriter *mocks.MockStatusWriter
		projectClaim     *api.ProjectClaim
		err              error
	)
	BeforeEach(func() {
		projectReference = testStructs.NewProjectReferenceBuilder().GetProjectReference()
		projectClaim = testStructs.NewProjectClaimBuilder().GetProjectClaim()
		ctrl := gomock.NewController(GinkgoT())
		mockStatusWriter = mocks.NewMockStatusWriter(ctrl)
		mockKubeClient = mocks.NewMockClient(ctrl)
		mockGCPClient = mockGCP.NewMockClient(ctrl)
	})
	JustBeforeEach(func() {
		claimLink := types.NamespacedName{Name: projectReference.Spec.ProjectClaimCRLink.Name, Namespace: projectReference.Spec.ProjectClaimCRLink.Namespace}
		mockKubeClient.EXPECT().Get(gomock.Any(), claimLink, gomock.Any()).SetArg(2, *projectClaim)
		adapter, err = newReferenceAdapter(projectReference, logf.Log.WithName("Test Logger"), mockKubeClient, mockGCPClient)
		Expect(err).NotTo(HaveOccurred())
	})
	Context("generated project names", func() {
		It("are shorter than 30 characters", func() {
			projectID, err := GenerateProjectID()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(projectID)).To(BeNumerically("<=", 30))
		})

		It("are longer than 6 characters", func() {
			projectID, err := GenerateProjectID()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(projectID)).To(BeNumerically(">=", 6))
		})

		It("start with a lowercase letter", func() {
			projectID, err := GenerateProjectID()
			Expect(err).NotTo(HaveOccurred())
			Expect("abcdefghijklmnopqrstuvwxyz").To(ContainSubstring(string(projectID[0])))
		})

		It("contains only lowercase letters, numbers or hyphens", func() {
			projectID, err := GenerateProjectID()
			Expect(err).NotTo(HaveOccurred())
			for _, char := range projectID {
				Expect("abcdefghijklmnopqrstuvwxyz1234567890-").To(ContainSubstring(string(char)))
			}
		})
	})

	Context("EnsureProjectClaimUpdated", func() {
		Context("When ProjectReference is in creating state", func() {
			BeforeEach(func() {
				projectReference.Status.State = api.ProjectReferenceStatusCreating
			})

			It("returns without altering ProjectClaim", func() {
				oldClaim := projectClaim.DeepCopy()
				state, err := adapter.EnsureProjectClaimUpdated()
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(projectClaim.Status.State))
				Expect(adapter.projectClaim).To(Equal(oldClaim))
			})
		})
		Context("When ProjectReference is in Ready state", func() {
			BeforeEach(func() {
				projectReference.Status.State = api.ProjectReferenceStatusReady
			})

			Context("When ProjectClaim is in Ready state", func() {
				BeforeEach(func() {
					projectClaim.Status.State = api.ClaimStatusReady
				})

				It("returns without altering ProjectClaim", func() {
					oldClaim := projectClaim.DeepCopy()
					state, err := adapter.EnsureProjectClaimUpdated()
					Expect(err).NotTo(HaveOccurred())
					Expect(state).To(Equal(projectClaim.Status.State))
					Expect(adapter.projectClaim).To(Equal(oldClaim))
				})
			})

			Context("When ProjectClaim is not in Ready state", func() {
				BeforeEach(func() {
					projectClaim.Status.State = api.ClaimStatusPending
					projectClaim.Spec.GCPProjectID = ""
					projectReference.Spec.GCPProjectID = "fake-gcp-project"

					mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any())
					mockKubeClient.EXPECT().Status().Return(mockStatusWriter)
					mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any())
				})

				It("updates the ProjectClaim, sets GCPProjectID and the state to Ready", func() {
					state, err := adapter.EnsureProjectClaimUpdated()
					Expect(err).NotTo(HaveOccurred())
					Expect(state).To(Equal(api.ClaimStatusReady))
					Expect(adapter.projectClaim.Spec.GCPProjectID).To(Equal(adapter.projectReference.Spec.GCPProjectID))
				})
			})
		})
	})

	Context("EnsureProjectConfigured", func() {
		var (
			configMap corev1.ConfigMap
		)

		BeforeEach(func() {
			configMap = corev1.ConfigMap{
				Data: map[string]string{
					"billingAccount": "fake-account",
					"parentFolderId": "fake-folder",
				},
			}
		})

		JustBeforeEach(func() {
			projectReference.Spec.GCPProjectID = "Some fake id"
			projectReference.Status.State = api.ProjectReferenceStatusCreating
		})

		Context("When it fails to get Parent Folder ID", func() {
			It("requeues with error", func() {
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
					Data: map[string]string{},
				})
				err := adapter.EnsureProjectConfigured()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When it fails to create Project", func() {

			Context("When the project is Inactive", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "DELETE_REQUESTED", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					err := adapter.EnsureProjectConfigured()
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the project is Inactive and fails to update", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "DELETE_REQUESTED", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					err := adapter.EnsureProjectConfigured()
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the project is Inactive and fails to update", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "DELETE_REQUESTED", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					err := adapter.EnsureProjectConfigured()
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the project is Inactive and fails to update", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "DELETE_REQUESTED", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					err := adapter.EnsureProjectConfigured()
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When failing to configure APIS", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap).Times(1)
					mockGCPClient.EXPECT().EnableAPI(gomock.Any(), gomock.Any()).Times(1)
					mockGCPClient.EXPECT().CreateCloudBillingAccount(gomock.Any(), gomock.Any()).Return(errors.New("Fake Enable APIS Error"))
					err := adapter.EnsureProjectConfigured()
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the failing to configure Service Accounts", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap).Times(1)
					mockGCPClient.EXPECT().EnableAPI(gomock.Any(), gomock.Any()).AnyTimes()
					mockGCPClient.EXPECT().CreateCloudBillingAccount(gomock.Any(), gomock.Any()).Return(nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "Some Email"}, nil)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, errors.New("Some Fake Set IAM Error"))
					err := adapter.EnsureProjectConfigured()
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the failing to create credentials", func() {
				It("It requeues with error", func() {
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap).Times(2)
					mockGCPClient.EXPECT().EnableAPI(gomock.Any(), gomock.Any()).AnyTimes()
					mockGCPClient.EXPECT().CreateCloudBillingAccount(gomock.Any(), gomock.Any()).Return(nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "Some Email"}, nil).Times(2)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "dGVzdAo="}, nil)
					mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("Fake Create Error"))
					err := adapter.EnsureProjectConfigured()
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When processes the project reference correctly", func() {
				It("It does not requeue", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap).Times(2)
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					mockGCPClient.EXPECT().EnableAPI(gomock.Any(), gomock.Any()).AnyTimes()
					mockGCPClient.EXPECT().CreateCloudBillingAccount(gomock.Any(), gomock.Any()).Return(nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "Some Email"}, nil).Times(2)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "dGVzdAo="}, nil)
					mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
					mockKubeClient.EXPECT().Status().Return(mockStatusWriter)
					mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("Fake update Error"))
					err := adapter.EnsureProjectConfigured()
					Expect(err).ToNot(HaveOccurred())
				})
			})

		})

	})
})
