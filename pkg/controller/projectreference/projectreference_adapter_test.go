package projectreference_test

import (
	"errors"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	clusterapi "github.com/openshift/cluster-api/pkg/util"
	api "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	. "github.com/openshift/gcp-project-operator/pkg/controller/projectreference"
	mocks "github.com/openshift/gcp-project-operator/pkg/util/mocks"
	mockconditions "github.com/openshift/gcp-project-operator/pkg/util/mocks/condition"
	mockGCP "github.com/openshift/gcp-project-operator/pkg/util/mocks/gcpclient"
	testStructs "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
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
		mockConditions   *mockconditions.MockConditions
		mockStatusWriter *mocks.MockStatusWriter
		projectClaim     *api.ProjectClaim
		err              error
		mockCtrl         *gomock.Controller
	)
	BeforeEach(func() {
		projectReference = testStructs.NewProjectReferenceBuilder().GetProjectReference()
		projectClaim = testStructs.NewProjectClaimBuilder().GetProjectClaim()
		mockCtrl = gomock.NewController(GinkgoT())
		mockStatusWriter = mocks.NewMockStatusWriter(mockCtrl)
		mockKubeClient = mocks.NewMockClient(mockCtrl)
		mockGCPClient = mockGCP.NewMockClient(mockCtrl)
		mockConditions = mockconditions.NewMockConditions(mockCtrl)
	})
	AfterEach(func() {
		mockCtrl.Finish()
	})
	JustBeforeEach(func() {
		claimLink := types.NamespacedName{Name: projectReference.Spec.ProjectClaimCRLink.Name, Namespace: projectReference.Spec.ProjectClaimCRLink.Namespace}
		mockKubeClient.EXPECT().Get(gomock.Any(), claimLink, gomock.Any()).SetArg(2, *projectClaim)
		adapter, err = NewReferenceAdapter(projectReference, logf.Log.WithName("Test Logger"), mockKubeClient, mockGCPClient, mockConditions)
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
				_, err := EnsureProjectClaimReady(adapter)
				Expect(err).NotTo(HaveOccurred())
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
					_, err := EnsureProjectClaimReady(adapter)
					Expect(err).NotTo(HaveOccurred())
					Expect(adapter.ProjectClaim).To(Equal(oldClaim))
				})
			})

			Context("When ProjectClaim is not in Ready state", func() {
				Context("When compute API is ready", func() {
					BeforeEach(func() {
						projectClaim.Status.State = api.ClaimStatusPending
						projectClaim.Spec.GCPProjectID = ""
						projectReference.Spec.GCPProjectID = "fake-gcp-project"

						mockConditions.EXPECT().SetCondition(gomock.Any(), gcpv1alpha1.ConditionComputeApiReady, corev1.ConditionTrue, "QueryAvailabilityZonesSucceeded", "ComputeAPI ready, successfully queried availability zones").Times(1)
						mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any())
						mockKubeClient.EXPECT().Status().Return(mockStatusWriter)
						mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any())
						mockGCPClient.EXPECT().ListAvailabilityZones(gomock.Any(), gomock.Any()).Return([]string{"zone1", "zone2", "zone3"}, nil)
					})

					It("updates the ProjectClaim, sets GCPProjectID and the state to Ready", func() {
						_, err := EnsureProjectClaimReady(adapter)
						Expect(err).NotTo(HaveOccurred())
						Expect(adapter.ProjectClaim.Status.State).To(Equal(api.ClaimStatusReady))
						Expect(adapter.ProjectClaim.Spec.GCPProjectID).To(Equal(adapter.ProjectReference.Spec.GCPProjectID))
						Expect(adapter.ProjectClaim.Spec.AvailabilityZones).To(Equal([]string{"zone1", "zone2", "zone3"}))
					})
				})
				Context("When compute API is not ready", func() {
					var (
						fakeCondition  gcpv1alpha1.Condition
						conditionFound bool
					)
					JustBeforeEach(func() {
						mockConditions.EXPECT().FindCondition(gomock.Any(), gcpv1alpha1.ConditionComputeApiReady).Return(&fakeCondition, conditionFound).Times(1)
					})
					BeforeEach(func() {
						projectClaim.Status.State = api.ClaimStatusPending
						projectClaim.Spec.GCPProjectID = ""
						projectReference.Spec.GCPProjectID = "fake-gcp-project"

						conditionFound = false
						mockConditions.EXPECT().SetCondition(gomock.Any(), gcpv1alpha1.ConditionComputeApiReady, corev1.ConditionFalse, "QueryAvailabilityZonesFailed", "ComputeAPI not yet ready, couldn't query availability zones").Times(1)
						mockKubeClient.EXPECT().Status().Return(mockStatusWriter)
						mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any())
						mockGCPClient.EXPECT().ListAvailabilityZones(gomock.Any(), gomock.Any()).Return([]string{}, errors.New("googleapi: Error 403: Access Not Configured. Compute Engine API has not been used in project ...."))
					})

					Context("When compute API is not ready and no condition is set, yet", func() {
						BeforeEach(func() {
							mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any())
						})
						It("does not return an error", func() {
							_, err := EnsureProjectClaimReady(adapter)
							Expect(err).NotTo(HaveOccurred())
							Expect(adapter.ProjectClaim.Status.State).NotTo(Equal(api.ClaimStatusReady))
						})
					})
					Context("When compute API is not ready after 8 minutes", func() {
						BeforeEach(func() {
							conditionFound = true
							fakeCondition = gcpv1alpha1.Condition{
								Type:               gcpv1alpha1.ConditionComputeApiReady,
								Status:             corev1.ConditionFalse,
								LastProbeTime:      metav1.NewTime(time.Now()),
								LastTransitionTime: metav1.NewTime(time.Now().Add(time.Duration(-9 * time.Minute))),
								Reason:             "fake-reason",
								Message:            "fake-message",
							}
							mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any())

						})
						It("does not return an error", func() {
							_, err := EnsureProjectClaimReady(adapter)
							Expect(err).NotTo(HaveOccurred())
							Expect(adapter.ProjectClaim.Status.State).NotTo(Equal(api.ClaimStatusReady))
						})
					})
					Context("When compute API is not ready after 11 minutes", func() {
						BeforeEach(func() {
							conditionFound = true
							fakeCondition = gcpv1alpha1.Condition{
								Type:               gcpv1alpha1.ConditionComputeApiReady,
								Status:             corev1.ConditionFalse,
								LastProbeTime:      metav1.NewTime(time.Now()),
								LastTransitionTime: metav1.NewTime(time.Now().Add(time.Duration(-11 * time.Minute))),
								Reason:             "fake-reason",
								Message:            "fake-message",
							}

						})
						It("returns an error", func() {
							_, err := EnsureProjectClaimReady(adapter)
							Expect(err).To(HaveOccurred())
							Expect(adapter.ProjectClaim.Status.State).NotTo(Equal(api.ClaimStatusReady))
						})
					})
				})
			})

			Context("SetProjectReferenceCondition()", func() {
				var (
					err           = errors.New("fake reconcile error")
					reason        = "ReconcileError"
					conditionType = gcpv1alpha1.ConditionError
				)
				Context("when no conditions defined before and the err is nil", func() {
					It("It returns nil ", func() {
						errTemp := adapter.SetProjectReferenceCondition(reason, nil)
						Expect(errTemp).To(BeNil())
					})
				})
				Context("when the err comes from reconcileHandler", func() {
					It("It should update the CRD", func() {
						conditions := &adapter.ProjectReference.Status.Conditions
						mockKubeClient.EXPECT().Status().Return(mockStatusWriter)
						mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
						mockConditions.EXPECT().SetCondition(conditions, conditionType, corev1.ConditionTrue, reason, err.Error()).Times(1)
						_ = adapter.SetProjectReferenceCondition(reason, err)
					})
				})
				Context("when the err has been resolved", func() {
					It("It should update the CRD condition status as resolved", func() {
						conditions := &adapter.ProjectReference.Status.Conditions
						*conditions = append(*conditions, gcpv1alpha1.Condition{})
						mockKubeClient.EXPECT().Status().Return(mockStatusWriter)
						mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
						mockConditions.EXPECT().SetCondition(conditions, conditionType, corev1.ConditionFalse, "ReconcileErrorResolved", "").Times(1)
						_ = adapter.SetProjectReferenceCondition(reason, nil)
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
				_, err := EnsureProjectConfigured(adapter)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When it fails to create Project", func() {

			Context("When the project is Inactive", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the project is Inactive and fails to update", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the project is Inactive and fails to update", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the project is Inactive and fails to update", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When failing to configure APIS", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the failing to configure Service Accounts", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					mockGCPClient.EXPECT().EnableAPI(gomock.Any(), gomock.Any()).AnyTimes()
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the failing to create credentials", func() {
				It("It requeues with error", func() {
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap).Times(1)
					mockGCPClient.EXPECT().EnableAPI(gomock.Any(), gomock.Any()).AnyTimes()
					mockGCPClient.EXPECT().CreateCloudBillingAccount(gomock.Any(), gomock.Any()).Return(nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "Some Email"}, nil).Times(2)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("Ooops not found")).Times(1)
					mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "dGVzdAo="}, nil)
					mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("Fake Create Error"))
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When processes the project reference correctly", func() {
				It("It does not requeue", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap).Times(1)
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					mockGCPClient.EXPECT().EnableAPI(gomock.Any(), gomock.Any()).AnyTimes()
					mockGCPClient.EXPECT().CreateCloudBillingAccount(gomock.Any(), gomock.Any()).Return(nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "Some Email"}, nil).Times(2)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("Ooops not found")).Times(1)
					mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "dGVzdAo="}, nil)
					mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
					_, err := EnsureProjectConfigured(adapter)
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
					_, err := EnsureFinalizerAdded(adapter)
					Expect(err).ToNot(HaveOccurred())
					Expect(projectReference.Finalizers).To(ContainElement(FinalizerName))
				})
			})
			Context("When the finalizer exists", func() {
				It("does nothing", func() {
					adapter.ProjectReference.SetFinalizers([]string{FinalizerName})
					_, err := EnsureFinalizerAdded(adapter)
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
