package projectreference_test

import (
	"errors"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	clusterapi "github.com/openshift/cluster-api/pkg/util"
	api "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	. "github.com/openshift/gcp-project-operator/pkg/controller/projectreference"
	operrors "github.com/openshift/gcp-project-operator/pkg/util/errors"
	mocks "github.com/openshift/gcp-project-operator/pkg/util/mocks"
	mockGCP "github.com/openshift/gcp-project-operator/pkg/util/mocks/gcpclient"
	testStructs "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	mockutil "github.com/openshift/gcp-project-operator/pkg/util/mocks/util"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var _ = Describe("ProjectreferenceAdapter", func() {
	var (
		adapter          *ReferenceAdapter
		projectReference *api.ProjectReference
		mockKubeClient   *mocks.MockClient
		mockGCPClient    *mockGCP.MockClient
		mockUtil         *mockutil.MockUtil
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
		mockUtil = mockutil.NewMockUtil(ctrl)
	})
	JustBeforeEach(func() {
		claimLink := types.NamespacedName{Name: projectReference.Spec.ProjectClaimCRLink.Name, Namespace: projectReference.Spec.ProjectClaimCRLink.Namespace}
		mockKubeClient.EXPECT().Get(gomock.Any(), claimLink, gomock.Any()).SetArg(2, *projectClaim)
		adapter, err = NewReferenceAdapter(projectReference, logf.Log.WithName("Test Logger"), mockKubeClient, mockGCPClient, mockUtil)
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
				state, err := adapter.EnsureProjectClaimReady()
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(projectClaim.Status.State))
				Expect(adapter.ProjectClaim).To(Equal(oldClaim))
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
					state, err := adapter.EnsureProjectClaimReady()
					Expect(err).NotTo(HaveOccurred())
					Expect(state).To(Equal(projectClaim.Status.State))
					Expect(adapter.ProjectClaim).To(Equal(oldClaim))
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
					mockGCPClient.EXPECT().ListAvilibilityZones(gomock.Any(), gomock.Any()).Return([]string{"zone1", "zone2", "zone3"}, nil)
				})

				It("updates the ProjectClaim, sets GCPProjectID and the state to Ready", func() {
					state, err := adapter.EnsureProjectClaimReady()
					Expect(err).NotTo(HaveOccurred())
					Expect(state).To(Equal(api.ClaimStatusReady))
					Expect(adapter.ProjectClaim.Spec.GCPProjectID).To(Equal(adapter.ProjectReference.Spec.GCPProjectID))
				})
			})

			Context("SetProjectReferenceCondition()", func() {
				var (
					message = "fakeError"
					reason  = "fakeReconcileHandlerFailed"
				)
				Context("when the err comes from reconcileHandler", func() {
					It("should update the CRD", func() {
						mockKubeClient.EXPECT().Status().Return(mockStatusWriter)
						mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
						mockUtil.EXPECT().SetCondition(gomock.Any(), corev1.ConditionTrue, reason, message).Times(1)
						err = adapter.SetProjectReferenceCondition(corev1.ConditionTrue, reason, message)
						Expect(err).NotTo(HaveOccurred())
					})
				})
				Context("when the err comes from reconcileHandler", func() {
					It("shouldn't update the CRD if we took error from SetCondition()", func() {
						mockUtil.EXPECT().SetCondition(gomock.Any(), corev1.ConditionTrue, reason, message).Times(1).Return(errors.New("fake set condition error"))
						err = adapter.SetProjectReferenceCondition(corev1.ConditionTrue, reason, message)
						Expect(err).To(Equal(errors.New("fake set condition error")))
					})
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

		Context("IsDeletionRequested", func() {
			Context("If there is a deletionTimestamp", func() {
				It("returns true", func() {
					deletionTime := metav1.NewTime(time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC))
					projectReference.SetDeletionTimestamp(&deletionTime)
					Expect(adapter.IsDeletionRequested()).To(BeTrue())
				})
			})
			Context("If there is no deletionTimestamp", func() {
				It("returns false", func() {
					projectReference.SetDeletionTimestamp(nil)
					Expect(adapter.IsDeletionRequested()).NotTo(BeTrue())
				})
			})
		})

		Context("EnsureFinalizerDeleted", func() {
			Context("When the finalizer exists", func() {
				It("removes the finalizer and updates the instance", func() {
					adapter.ProjectReference.SetFinalizers([]string{FinalizerName})
					mockKubeClient.EXPECT().Update(gomock.Any(), projectReference)
					err := adapter.EnsureFinalizerDeleted()
					Expect(err).ToNot(HaveOccurred())
					Expect(projectReference.Finalizers).ToNot(ContainElement(FinalizerName))
				})
			})
			Context("When the finalizer does not exist", func() {
				It("does nothing", func() {
					projectReference.SetFinalizers(clusterapi.Filter(projectReference.GetFinalizers(), FinalizerName))
					err := adapter.EnsureFinalizerDeleted()
					Expect(err).ToNot(HaveOccurred())
					Expect(projectReference.Finalizers).ToNot(ContainElement(FinalizerName))
				})
			})
		})

		Context("EnsureFinalizerAdded", func() {
			Context("When the finalizer does not exist", func() {
				It("adds the finalizer and updates the instance", func() {
					projectReference.SetFinalizers(clusterapi.Filter(projectReference.GetFinalizers(), FinalizerName))
					mockKubeClient.EXPECT().Update(gomock.Any(), projectReference)
					err := adapter.EnsureFinalizerAdded()
					Expect(err).ToNot(HaveOccurred())
					Expect(projectReference.Finalizers).To(ContainElement(FinalizerName))
				})
			})
			Context("When the finalizer exists", func() {
				It("does nothing", func() {
					adapter.ProjectReference.SetFinalizers([]string{FinalizerName})
					err := adapter.EnsureFinalizerAdded()
					Expect(err).ToNot(HaveOccurred())
					Expect(projectReference.Finalizers).To(ContainElement(FinalizerName))
				})
			})
		})

		Context("EnsureProjectCleanedUp", func() {
			var (
				projectState string
			)
			JustBeforeEach(func() {
				mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: projectState, ProjectId: projectReference.Spec.GCPProjectID}}, nil)
			})
			BeforeEach(func() {
				projectReference.Spec.GCPProjectID = "fake-id"
				projectState = "ACTIVE"
			})
			Context("When the lifecycleStatus is unknown", func() {
				BeforeEach(func() {
					projectState = "UNKNOWN"
				})
				It("returns an error", func() {
					err := adapter.EnsureProjectCleanedUp()
					Expect(err).To(HaveOccurred())
				})
			})
			Context("When the lifecycleStatus is LIFECYCLE_STATE_UNSPECIFIED", func() {
				BeforeEach(func() {
					projectState = "LIFECYCLE_STATE_UNSPECIFIED"
				})
				It("returns an error", func() {
					err := adapter.EnsureProjectCleanedUp()
					Expect(err).To(HaveOccurred())
				})
			})
			Context("When the lifecycleStatus is DELETE_REQUESTED", func() {
				BeforeEach(func() {
					projectState = "DELETE_REQUESTED"
				})
				It("deletes the project", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, v1.Secret{}).Times(2)
					mockKubeClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1)
					err := adapter.EnsureProjectCleanedUp()
					Expect(err).NotTo(HaveOccurred())
				})
			})
			Context("When the lifecycleStatus is ACTIVE", func() {
				It("deletes the project", func() {
					mockGCPClient.EXPECT().DeleteProject(gomock.Any()).Times(1)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, v1.Secret{}).Times(2)
					mockKubeClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1)
					err := adapter.EnsureProjectCleanedUp()
					Expect(err).NotTo(HaveOccurred())
				})
			})
			Context("When it cannot delete the project", func() {
				It("returns an error", func() {
					mockGCPClient.EXPECT().DeleteProject(gomock.Any()).Times(1)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, v1.Secret{}).Times(2)
					mockKubeClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errors.New("Cannot delete the project"))
					err := adapter.EnsureProjectCleanedUp()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("CheckRequirements", func() {
			Context("When the region is supported", func() {
				BeforeEach(func() {
					projectClaim.Spec.Region = "us-east1"
				})
				It("does not return an error", func() {
					err := adapter.CheckRequirements()
					Expect(err).NotTo(HaveOccurred())
				})
			})
			Context("When the region is not supported", func() {
				BeforeEach(func() {
					projectClaim.Spec.Region = "eu-west6"
				})
				It("does not return an error", func() {
					err := adapter.CheckRequirements()
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(operrors.ErrRegionNotSupported))
				})
			})
		})

		Context("UpdateProjectID", func() {
			BeforeEach(func() {
				mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any())
			})
			It("Sets a new projectid", func() {
				projectIDBefore := projectReference.Spec.GCPProjectID
				err := adapter.UpdateProjectID()
				Expect(err).NotTo(HaveOccurred())
				Expect(projectReference.Spec.GCPProjectID).NotTo(Equal(projectIDBefore))
			})

		})
	})
})
