package projectclaim_test

import (
	"context"
	er "errors"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/openshift/gcp-project-operator/pkg/configmap"
	"github.com/openshift/gcp-project-operator/pkg/util"
	"github.com/openshift/gcp-project-operator/pkg/util/mocks"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/openshift/gcp-project-operator/controllers/projectclaim"

	gcpv1alpha1 "github.com/openshift/gcp-project-operator/api/v1alpha1"
	mockconditions "github.com/openshift/gcp-project-operator/pkg/util/mocks/condition"
	testStructs "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("Customresourceadapter", func() {
	var (
		adapter             *ProjectClaimAdapter
		mockCtrl            *gomock.Controller
		mockClient          *mocks.MockClient
		mockStatusWriter    *mocks.MockStatusWriter
		projectClaim        *gcpv1alpha1.ProjectClaim
		mockConditions      *mockconditions.MockConditions
		ccsSecret           corev1.Secret
		GCPCredentialSecret corev1.Secret
	)

	BeforeEach(func() {
		projectClaim = testStructs.NewProjectClaimBuilder().Initialized().GetProjectClaim()
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(mockCtrl)
		mockConditions = mockconditions.NewMockConditions(mockCtrl)
		mockStatusWriter = mocks.NewMockStatusWriter(mockCtrl)
		ccsSecret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-name",
				Namespace: projectClaim.Namespace,
			},
		}
		GCPCredentialSecret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      projectClaim.Spec.GCPCredentialSecret.Name,
				Namespace: projectClaim.Spec.GCPCredentialSecret.Namespace,
			},
		}
	})
	JustBeforeEach(func() {
		adapter = NewProjectClaimAdapter(projectClaim, logf.Log.WithName("Test Logger"), mockClient, mockConditions)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("ProjectReferenceExists", func() {
		Context("when ProjectReference exists", func() {
			It("returns true", func() {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *testStructs.NewProjectReferenceBuilder().GetProjectReference())
				exists, err := adapter.ProjectReferenceExists()
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})

		Context("when ProjectReference doesn't exist", func() {
			It("returns false", func() {
				notFound := errors.NewNotFound(schema.GroupResource{}, "FakeProjectReference")
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(notFound)
				exists, err := adapter.ProjectReferenceExists()
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())
			})
		})
	})

	Context("IsProjectClaimDeletion", func() {
		It("returns true when DeletionTimeStamp is set on ProjectClaim", func() {
			deletionTime := metav1.NewTime(time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC))
			projectClaim.SetDeletionTimestamp(&deletionTime)
			Expect(adapter.IsProjectClaimDeletion()).To(BeTrue())
		})

		It("returns false when DeletionTimeStamp is not set on ProjectClaim", func() {
			projectClaim.SetDeletionTimestamp(nil)
			Expect(adapter.IsProjectClaimDeletion()).To(BeFalse())
		})
	})

	Context("When the EnsureRegionSupported() is called", func() {
		BeforeEach(func() {
			configMap := corev1.ConfigMap{
				Data: map[string]string{
					configmap.OperatorConfigMapKey: `
billingAccount: fake-account
parentFolderID: fake-folder
disabledRegions:
- australia-southeast1
- northamerica-northeast1
- southamerica-east1

- europe-west3
- europe-west6
- europe-north1
- asia-northeast2
- asia-south1
`,
				},
			}
			mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap).AnyTimes()

		})
		Context("if the projectclaim has a supported region", func() {
			BeforeEach(func() {
				mockConditions.EXPECT().HasCondition(gomock.Any(), gcpv1alpha1.ConditionInvalid).Return(false)
				projectClaim.Spec.Region = "us-east1"
			})
			It("should return nil", func() {
				res, err := adapter.EnsureRegionSupported()
				Expect(res).To(Equal(util.ContinueOperationResult()))
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("if the projectclaim has an unsupported region", func() {
			BeforeEach(func() {
				projectClaim.Spec.Region = "europe-west3"
			})
			Context("when it is not a CCS cluster", func() {
				BeforeEach(func() {
					mockConditions.EXPECT().SetCondition(gomock.Any(), gcpv1alpha1.ConditionInvalid, corev1.ConditionTrue, RegionCheckFailed, gomock.Any())
					matcher := testStructs.NewProjectClaimMatcher()
					mockClient.EXPECT().Status().Return(mockStatusWriter)
					mockStatusWriter.EXPECT().Update(gomock.Any(), matcher)
				})
				It("should return err", func() {
					res, err := adapter.EnsureRegionSupported()
					Expect(err).NotTo(HaveOccurred())
					Expect(res).To(Equal(util.StopOperationResult()))
				})
			})
			Context("when it is a CCS cluster", func() {
				BeforeEach(func() {
					mockConditions.EXPECT().HasCondition(gomock.Any(), gcpv1alpha1.ConditionInvalid).Return(false)
					projectClaim.Spec.CCS = true
				})
				It("should return nil", func() {
					res, err := adapter.EnsureRegionSupported()
					Expect(res).To(Equal(util.ContinueOperationResult()))
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})

	Context("FinalizeProjectClaim", func() {
		var (
			matcher *testStructs.ProjectClaimMatcher
		)
		BeforeEach(func() {
			projectClaim = testStructs.NewProjectClaimBuilder().WithFinalizer([]string{ProjectClaimFinalizer}).Initialized().GetProjectClaim()
			matcher = testStructs.NewProjectClaimMatcher()
		})

		Context("when the project reference doesn't exist", func() {
			var (
				notFound error
			)
			BeforeEach(func() {
				notFound = errors.NewNotFound(schema.GroupResource{}, "FakeProjectReference")
			})
			Context("when it's a not CCS cluster", func() {
				BeforeEach(func() {
					mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(notFound)
					mockClient.EXPECT().Update(gomock.Any(), matcher).Times(1)
				})

				It("removes the finalizer", func() {
					crStatus, err := adapter.FinalizeProjectClaim()
					Expect(err).ToNot(HaveOccurred())
					Expect(crStatus).To(Equal(ObjectModified))
					Expect(matcher.ActualProjectClaim.Finalizers).ToNot(ContainElement(ProjectClaimFinalizer))
				})
			})

			Context("when it's a CCS cluster", func() {
				var (
					secretMatcher *testStructs.SecretMatcher
				)
				BeforeEach(func() {
					secretMatcher = testStructs.NewSecretMatcher()
					projectClaim.Spec.CCS = true
					ccsSecret.Finalizers = []string{CCSSecretFinalizer}
				})
				JustBeforeEach(func() {
					mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(notFound)
					mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, ccsSecret)
					mockClient.EXPECT().Update(gomock.Any(), secretMatcher).Times(1)
					mockClient.EXPECT().Update(gomock.Any(), matcher).Times(1)
				})

				It("removes the finalizer as well from the CCS secret", func() {
					crStatus, err := adapter.FinalizeProjectClaim()
					Expect(err).ToNot(HaveOccurred())
					Expect(crStatus).To(Equal(ObjectModified))
					Expect(matcher.ActualProjectClaim.Finalizers).ToNot(ContainElement(ProjectClaimFinalizer))
					Expect(secretMatcher.ActualSecret.Finalizers).ToNot(ContainElement(CCSSecretFinalizer))
				})
			})
		})

		Context("when the project reference exists", func() {
			It("there is no error and claim object is not deleted", func() {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *testStructs.NewProjectReferenceBuilder().GetProjectReference()).Times(2)
				mockClient.EXPECT().Delete(gomock.Any(), &testStructs.ProjectReferenceMatcher{}).Times(1)
				_, err := adapter.EnsureProjectReferenceExists()
				Expect(err).ToNot(HaveOccurred())
				crStatus, err := adapter.FinalizeProjectClaim()
				Expect(err).ToNot(HaveOccurred())
				Expect(crStatus).To(Equal(ObjectUnchanged))
			})
		})
	})

	Context("EnsureProjectClaimInitialized", func() {
		Context("When conditions are already existing", func() {
			BeforeEach(func() {
				projectClaim = testStructs.NewProjectClaimBuilder().Initialized().GetProjectClaim()
			})

			It("doesn't update ProjectClaim status", func() {
				result, err := adapter.EnsureProjectClaimInitialized()
				Expect(err).NotTo(HaveOccurred())
				Expect(result.CancelRequest).To(Equal(false))
			})
		})
		Context("When conditions are not set", func() {
			BeforeEach(func() {
				projectClaim.Status.Conditions = nil
			})
			It("Initializes them with an empty array", func() {
				matcher := testStructs.NewProjectClaimMatcher()
				mockClient.EXPECT().Status().Return(mockStatusWriter)
				mockStatusWriter.EXPECT().Update(gomock.Any(), matcher)
				result, err := adapter.EnsureProjectClaimInitialized()
				Expect(err).NotTo(HaveOccurred())
				Expect(result.CancelRequest).To(Equal(true))
				Expect(matcher.ActualProjectClaim.Status.Conditions).NotTo(Equal(nil))
				Expect(len(matcher.ActualProjectClaim.Status.Conditions)).To(Equal(0))
			})
		})
	})

	Context("EnsureProjectReferenceLink", func() {
		Context("when ProjectReferenceCRLink is not set", func() {
			It("sets the ProjectReferenceCRLink and returns ObjectModified", func() {
				matcher := testStructs.NewProjectClaimMatcher()
				mockClient.EXPECT().Update(gomock.Any(), matcher).Times(1)
				result, err := adapter.EnsureProjectReferenceLink()
				Expect(err).ToNot(HaveOccurred())
				Expect(matcher.ActualProjectClaim.Spec.ProjectReferenceCRLink.Name).To(Equal(projectClaim.GetNamespace() + "-" + projectClaim.GetName()))
				Expect(matcher.ActualProjectClaim.Spec.ProjectReferenceCRLink.Namespace).To(Equal(gcpv1alpha1.ProjectReferenceNamespace))
				Expect(result.CancelRequest).To(Equal(true))
			})
		})

		Context("when ProjectReferenceCRLink is set", func() {
			BeforeEach(func() {
				projectClaim.Spec.ProjectReferenceCRLink.Name = projectClaim.GetNamespace() + "-" + projectClaim.GetName()
				projectClaim.Spec.ProjectReferenceCRLink.Namespace = gcpv1alpha1.ProjectReferenceNamespace
			})

			It("doesn't update the ProjectClaim", func() {
				result, err := adapter.EnsureProjectReferenceLink()
				Expect(err).ToNot(HaveOccurred())
				Expect(result.CancelRequest).To(Equal(false))
			})
		})
	})

	Context("EnsureFinalizer", func() {
		Context("when finalizer is not set", func() {
			BeforeEach(func() {
				projectClaim.Finalizers = []string{}
			})
			It("sets the finalizer", func() {
				matcher := testStructs.NewProjectClaimMatcher()
				mockClient.EXPECT().Update(gomock.Any(), matcher).Times(1)
				result, err := adapter.EnsureFinalizer()
				Expect(err).NotTo(HaveOccurred())
				Expect(result.CancelRequest).To(Equal(true))
				Expect(projectClaim.Finalizers).To(ContainElement(ProjectClaimFinalizer))
			})
		})

		Context("when finalizer is set already", func() {
			BeforeEach(func() {
				projectClaim.Finalizers = []string{ProjectClaimFinalizer}
			})
			It("doesn't change the ProjectClaim", func() {
				result, err := adapter.EnsureFinalizer()
				Expect(err).NotTo(HaveOccurred())
				Expect(result.CancelRequest).To(Equal(false))
			})
		})
	})

	Context("IsProjectClaimFake", func() {
		BeforeEach(func() {
			projectClaim.Annotations = map[string]string{}
			projectClaim.Annotations[FakeProjectClaim] = "true"
		})
		Context("when ProjectClaim is marked for deletion", func() {
			BeforeEach(func() {
				secret := &corev1.Secret{}
				deletionTime := metav1.NewTime(time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC))
				projectClaim.SetDeletionTimestamp(&deletionTime)
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
				mockClient.EXPECT().Update(gomock.Any(), projectClaim).Times(1)
				mockClient.EXPECT().Delete(gomock.Any(), secret).Times(1)
				mockClient.EXPECT().Delete(gomock.Any(), &testStructs.ProjectReferenceMatcher{})
			})
			It("removes fake secret", func() {
				result, err := adapter.EnsureProjectClaimFakeProcessed()
				Expect(err).NotTo(HaveOccurred())
				Expect(result.CancelRequest).To(Equal(true))
			})
		})

		Context("when fake secret doesn't exist", func() {
			BeforeEach(func() {
				notFound := errors.NewNotFound(schema.GroupResource{}, "FakeSecret")
				matcher := testStructs.NewSecretMatcher()
				mockClient.EXPECT().Update(gomock.Any(), projectClaim).Times(2)
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(notFound)
				mockClient.EXPECT().Create(gomock.Any(), matcher)
			})
			It("creates fake secret", func() {
				_, err := adapter.EnsureProjectClaimFakeProcessed()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when ProjectClaim GCPProjectID is not fake", func() {
			BeforeEach(func() {
				projectClaim.Spec.GCPProjectID = ""
				mockClient.EXPECT().Update(gomock.Any(), projectClaim)
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, GCPCredentialSecret)

			})
			It("updates ProjectClaim with fake specs", func() {
				matcher := testStructs.NewProjectClaimMatcher()
				mockClient.EXPECT().Update(gomock.Any(), matcher).Times(1)
				result, err := adapter.EnsureProjectClaimFakeProcessed()
				Expect(err).NotTo(HaveOccurred())
				Expect(result.CancelRequest).To(Equal(true))
				Expect(matcher.ActualProjectClaim.Spec.GCPProjectID).To(Equal("fakeProjectClaim"))
				Expect(matcher.ActualProjectClaim.Spec.GCPCredentialSecret.Name).To(Equal(projectClaim.GetName()))
				Expect(matcher.ActualProjectClaim.Spec.GCPCredentialSecret.Namespace).To(Equal(projectClaim.GetNamespace()))
				Expect(matcher.ActualProjectClaim.Spec.Region).To(Equal("fakeRegion"))
				Expect(matcher.ActualProjectClaim.Spec.AvailabilityZones).To(Equal([]string{
					"fake-az-a",
					"fake-az-b",
					"fake-az-c",
				}))
			})
		})

		Context("when ProjectClaim status is not Ready", func() {
			BeforeEach(func() {
				projectClaim.Status.State = ""
				projectClaim.Spec.GCPProjectID = "fakeProjectClaim"
				mockClient.EXPECT().Update(gomock.Any(), projectClaim).Times(1)
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, GCPCredentialSecret)
			})
			It("updates ProjectClaim with Ready status", func() {
				matcher := testStructs.NewProjectClaimMatcher()
				mockClient.EXPECT().Status().Return(mockStatusWriter)
				mockStatusWriter.EXPECT().Update(gomock.Any(), matcher)
				result, err := adapter.EnsureProjectClaimFakeProcessed()
				Expect(err).NotTo(HaveOccurred())
				Expect(result.CancelRequest).To(Equal(true))
				Expect(matcher.ActualProjectClaim.Status.State).To(Equal(gcpv1alpha1.ClaimStatusReady))
			})
		})

		Context("when ProjectClaim status is Ready", func() {
			BeforeEach(func() {
				projectClaim.Status.State = "Ready"
				projectClaim.Spec.GCPProjectID = "fakeProjectClaim"
				mockClient.EXPECT().Update(gomock.Any(), projectClaim).Times(1)
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, GCPCredentialSecret)
			})
			It("doesn't change the ProjectClaim", func() {
				result, err := adapter.EnsureProjectClaimFakeProcessed()
				Expect(err).NotTo(HaveOccurred())
				Expect(result.CancelRequest).To(Equal(true))
			})
		})
	})

	Context("EnsureCCSSecretFinalizer", func() {
		Context("when it's not a CCS cluster", func() {
			BeforeEach(func() {
				projectClaim.Spec.CCS = false
			})
			It("doesn't cancel processing", func() {
				result, err := adapter.EnsureCCSSecretFinalizer()
				Expect(err).NotTo(HaveOccurred())
				Expect(result.CancelRequest).To(BeFalse())
			})

		})
		Context("when it's a CCS cluster", func() {
			BeforeEach(func() {
				projectClaim.Spec.CCS = true
			})
			JustBeforeEach(func() {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, ccsSecret).Times(1)
			})
			Context("when the finalizer of the ccs secret is not set", func() {
				BeforeEach(func() {
					projectClaim.Spec.CCSSecretRef = gcpv1alpha1.NamespacedName{
						Namespace: projectClaim.Namespace,
						Name:      "secret-name",
					}
				})
				It("sets the finalizer at the ccs secret", func() {

					matcher := testStructs.NewSecretMatcher()
					mockClient.EXPECT().Update(gomock.Any(), matcher).Times(1)

					result, err := adapter.EnsureCCSSecretFinalizer()
					Expect(matcher.ActualSecret.GetFinalizers()).To(ContainElement(CCSSecretFinalizer))
					Expect(err).NotTo(HaveOccurred())
					Expect(result.CancelRequest).To(Equal(false))
				})
			})

			Context("when the finalizer of the ccs secret is set already", func() {
				BeforeEach(func() {
					ccsSecret.Finalizers = []string{CCSSecretFinalizer}
				})

				It("doesn't cancel processing", func() {
					result, err := adapter.EnsureCCSSecretFinalizer()
					Expect(err).NotTo(HaveOccurred())
					Expect(result.CancelRequest).To(BeFalse())
				})
			})
		})
	})

	Context("EnsureProjectReferenceExists()", func() {
		Context("when matching ProjectReference doesn't exist", func() {
			BeforeEach(func() {
				notFound := errors.NewNotFound(schema.GroupResource{}, "FakeProjectReference")
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(notFound)
			})
			It("creates a ProjectReference", func() {
				matcher := testStructs.NewProjectReferenceMatcher()
				mockClient.EXPECT().Create(gomock.Any(), matcher)
				_, err := adapter.EnsureProjectReferenceExists()
				Expect(err).ToNot(HaveOccurred())
				Expect(matcher.ActualProjectReference.Name).To(Equal(projectClaim.GetNamespace() + "-" + projectClaim.GetName()))
				Expect(matcher.ActualProjectReference.Namespace).To(Equal(gcpv1alpha1.ProjectReferenceNamespace))
				Expect(matcher.ActualProjectReference.Spec.ProjectClaimCRLink.Name).To(Equal(projectClaim.Name))
				Expect(matcher.ActualProjectReference.Spec.ProjectClaimCRLink.Namespace).To(Equal(projectClaim.Namespace))
				Expect(matcher.ActualProjectReference.Spec.LegalEntity).To(Equal(projectClaim.Spec.LegalEntity))
			})
		})

		Context("when matching ProjectReference exists", func() {
			BeforeEach(func() {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *testStructs.NewProjectReferenceBuilder().GetProjectReference())
			})
			It("doesn't return an error", func() {
				_, err := adapter.EnsureProjectReferenceExists()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("EnsureProjectClaimState()", func() {
		var (
			requestedState gcpv1alpha1.ClaimStatus
			currentState   gcpv1alpha1.ClaimStatus
		)
		JustBeforeEach(func() {
			projectClaim.Status.State = currentState
		})

		Context("when requested state is Pending", func() {
			BeforeEach(func() {
				requestedState = gcpv1alpha1.ClaimStatusPending
			})

			Context("when ProjectClaim state is not empty", func() {
				BeforeEach(func() {
					currentState = gcpv1alpha1.ClaimStatusReady
				})
				It("doesn't change the ProjectClaim state", func() {
					_, _ = adapter.EnsureProjectClaimState(requestedState)
					Expect(projectClaim.Status.State).To(Equal(currentState))
				})
			})

			Context("when ProjectClaim state is empty", func() {
				BeforeEach(func() {
					currentState = ""
				})
				It("updates the state to Pending", func() {
					mockClient.EXPECT().Status().Times(1).Return(stubStatus{})
					_, _ = adapter.EnsureProjectClaimState(requestedState)
					Expect(projectClaim.Status.State).To(Equal(requestedState))
				})
			})
		})

		Context("when requested state is PendingProject", func() {
			BeforeEach(func() {
				requestedState = gcpv1alpha1.ClaimStatusPendingProject
			})

			Context("when ProjectClaim state is not Pending", func() {
				BeforeEach(func() {
					currentState = gcpv1alpha1.ClaimStatusReady
				})
				It("doesn't change the ProjectClaim state", func() {
					_, _ = adapter.EnsureProjectClaimState(requestedState)
					Expect(projectClaim.Status.State).To(Equal(currentState))
				})
			})

			Context("when ProjectClaim state is Pending", func() {
				BeforeEach(func() {
					currentState = gcpv1alpha1.ClaimStatusPending
				})
				It("updates the state to PendingProject", func() {
					mockClient.EXPECT().Status().Times(1).Return(stubStatus{})
					_, _ = adapter.EnsureProjectClaimState(requestedState)
					Expect(projectClaim.Status.State).To(Equal(requestedState))
				})
			})
		})

		Context("SetProjectClaimCondition()", func() {
			var (
				err           = er.New("fake reconcile")
				reason        = "ReconcileError"
				conditionType = gcpv1alpha1.ConditionError
			)
			Context("when no conditions defined before and the err is nil", func() {
				BeforeEach(func() {
					mockConditions.EXPECT().HasCondition(gomock.Any(), conditionType).Return(false)
				})
				It("It returns nil ", func() {
					_, errTemp := adapter.SetProjectClaimCondition(conditionType, reason, nil)
					Expect(errTemp).To(BeNil())
				})
			})
			Context("when the err comes from reconcileHandler", func() {
				It("should update the CR", func() {
					matcher := testStructs.NewProjectClaimMatcher()
					mockClient.EXPECT().Status().Return(mockStatusWriter)
					mockStatusWriter.EXPECT().Update(gomock.Any(), matcher)
					mockConditions.EXPECT().SetCondition(gomock.Any(), conditionType, corev1.ConditionTrue, reason, err.Error()).Times(1)
					res, err := adapter.SetProjectClaimCondition(conditionType, reason, err)
					Expect(res).To(Equal(util.StopOperationResult()))
					Expect(err).NotTo(HaveOccurred())
				})
			})
			Context("when the err has been resolved", func() {
				BeforeEach(func() {
					mockConditions.EXPECT().HasCondition(gomock.Any(), conditionType).Return(true)
					mockConditions.EXPECT().FindCondition(gomock.Any(), conditionType).Return(&gcpv1alpha1.Condition{}, true)
				})
				It("It should update the CR condition status as resolved", func() {
					matcher := testStructs.NewProjectClaimMatcher()
					conditions := &projectClaim.Status.Conditions
					*conditions = append(*conditions, gcpv1alpha1.Condition{})
					mockClient.EXPECT().Status().Return(mockStatusWriter)
					mockStatusWriter.EXPECT().Update(gomock.Any(), matcher)
					mockConditions.EXPECT().SetCondition(conditions, conditionType, corev1.ConditionFalse, "ReconcileErrorResolved", "").Times(1)
					res, err := adapter.SetProjectClaimCondition(conditionType, reason, nil)
					Expect(res).To(Equal(util.StopOperationResult()))
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})
})

type stubStatus struct{}

var _ client.StatusWriter = stubStatus{}

func (stubStatus) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}
func (stubStatus) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}
