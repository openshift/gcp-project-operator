package projectclaim_test

import (
	"context"
	er "errors"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	"github.com/openshift/gcp-project-operator/pkg/controller/projectclaim"
	. "github.com/openshift/gcp-project-operator/pkg/controller/projectclaim"
	mockconditions "github.com/openshift/gcp-project-operator/pkg/util/mocks/condition"
	k8sclient "github.com/openshift/gcp-project-operator/pkg/util/mocks/k8sclient"
	testStructs "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var _ = Describe("Customresourceadapter", func() {
	var (
		adapter          CustomResourceAdapter
		mockCtrl         *gomock.Controller
		mockClient       *k8sclient.MockClient
		mockStatusWriter *k8sclient.MockStatusWriter
		projectClaim     *gcpv1alpha1.ProjectClaim
		mockConditions   *mockconditions.MockConditions
	)

	BeforeEach(func() {
		projectClaim = testStructs.NewProjectClaimBuilder().Initialized().GetProjectClaim()
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = k8sclient.NewMockClient(mockCtrl)
		mockConditions = mockconditions.NewMockConditions(mockCtrl)
		mockStatusWriter = k8sclient.NewMockStatusWriter(mockCtrl)
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
		Context("if the projectclaim has a supported region", func() {
			BeforeEach(func() {
				projectClaim.Spec.Region = "us-east1"
			})
			It("should return nil", func() {
				err := adapter.EnsureRegionSupported()
				Expect(err).To(BeNil())
			})
		})
		Context("if the projectclaim has an unsupported region", func() {
			BeforeEach(func() {
				matcher := testStructs.NewProjectClaimMatcher()
				mockClient.EXPECT().Status().Return(mockStatusWriter)
				mockStatusWriter.EXPECT().Update(gomock.Any(), matcher)
				projectClaim.Spec.Region = "fake-region"
			})
			It("should return err", func() {
				err := adapter.EnsureRegionSupported()
				Expect(err.Error()).Should(ContainSubstring("gcp-project-operator/pkg/controller/projectclaim/projectclaimadapter.go"))
				Expect(err.Error()).Should(ContainSubstring("Line:"))
				Expect(err.Error()).Should(ContainSubstring("gcp-project-operator/pkg/controller/projectclaim.(*ProjectClaimAdapter).EnsureRegionSupported"))
				Expect(err.Error()).Should(ContainSubstring("RegionNotSupported"))
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
			BeforeEach(func() {
				notFound := errors.NewNotFound(schema.GroupResource{}, "FakeProjectReference")
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

		Context("when the project reference exists", func() {
			It("there is no error and claim object is not deleted", func() {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *testStructs.NewProjectReferenceBuilder().GetProjectReference()).Times(2)
				mockClient.EXPECT().Delete(gomock.Any(), &testStructs.ProjectReferenceMatcher{}).Times(1)
				err := adapter.EnsureProjectReferenceExists()
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
				crState, err := adapter.EnsureProjectClaimInitialized()
				Expect(err).NotTo(HaveOccurred())
				Expect(crState).To(Equal(projectclaim.ObjectUnchanged))
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
				crState, err := adapter.EnsureProjectClaimInitialized()
				Expect(err).NotTo(HaveOccurred())
				Expect(crState).To(Equal(projectclaim.ObjectModified))
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
				crState, err := adapter.EnsureProjectReferenceLink()
				Expect(err).ToNot(HaveOccurred())
				Expect(matcher.ActualProjectClaim.Spec.ProjectReferenceCRLink.Name).To(Equal(projectClaim.GetNamespace() + "-" + projectClaim.GetName()))
				Expect(matcher.ActualProjectClaim.Spec.ProjectReferenceCRLink.Namespace).To(Equal(gcpv1alpha1.ProjectReferenceNamespace))
				Expect(crState).To(Equal(projectclaim.ObjectModified))
			})
		})

		Context("when ProjectReferenceCRLink is set", func() {
			BeforeEach(func() {
				projectClaim.Spec.ProjectReferenceCRLink.Name = projectClaim.GetNamespace() + "-" + projectClaim.GetName()
				projectClaim.Spec.ProjectReferenceCRLink.Namespace = gcpv1alpha1.ProjectReferenceNamespace
			})

			It("doesn't update the ProjectClaim", func() {
				crState, err := adapter.EnsureProjectReferenceLink()
				Expect(err).ToNot(HaveOccurred())
				Expect(crState).To(Equal(projectclaim.ObjectUnchanged))
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
				crState, err := adapter.EnsureFinalizer()
				Expect(err).NotTo(HaveOccurred())
				Expect(crState).To(Equal(projectclaim.ObjectModified))
				Expect(projectClaim.Finalizers).To(ContainElement(projectclaim.ProjectClaimFinalizer))
			})
		})

		Context("when finalizer is set already", func() {
			BeforeEach(func() {
				projectClaim.Finalizers = []string{projectclaim.ProjectClaimFinalizer}
			})
			It("doesn't change the ProjectClaim", func() {
				crState, err := adapter.EnsureFinalizer()
				Expect(err).NotTo(HaveOccurred())
				Expect(crState).To(Equal(projectclaim.ObjectUnchanged))
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
				err := adapter.EnsureProjectReferenceExists()
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
				err := adapter.EnsureProjectReferenceExists()
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
					adapter.EnsureProjectClaimState(requestedState)
					Expect(projectClaim.Status.State).To(Equal(currentState))
				})
			})

			Context("when ProjectClaim state is empty", func() {
				BeforeEach(func() {
					currentState = ""
				})
				It("updates the state to Pending", func() {
					mockClient.EXPECT().Status().Times(1).Return(stubStatus{})
					adapter.EnsureProjectClaimState(requestedState)
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
					adapter.EnsureProjectClaimState(requestedState)
					Expect(projectClaim.Status.State).To(Equal(currentState))
				})
			})

			Context("when ProjectClaim state is Pending", func() {
				BeforeEach(func() {
					currentState = gcpv1alpha1.ClaimStatusPending
				})
				It("updates the state to PendingProject", func() {
					mockClient.EXPECT().Status().Times(1).Return(stubStatus{})
					adapter.EnsureProjectClaimState(requestedState)
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
				It("It returns nil ", func() {
					errTemp := adapter.SetProjectClaimCondition(reason, nil)
					Expect(errTemp).To(BeNil())
				})
			})
			Context("when the err comes from reconcileHandler", func() {
				It("should update the CRD", func() {
					matcher := testStructs.NewProjectClaimMatcher()
					mockClient.EXPECT().Status().Return(mockStatusWriter)
					mockStatusWriter.EXPECT().Update(gomock.Any(), matcher)
					mockConditions.EXPECT().SetCondition(gomock.Any(), conditionType, corev1.ConditionTrue, reason, err.Error()).Times(1)
					adapter.SetProjectClaimCondition(reason, err)
				})
			})
			Context("when the err has been resolved", func() {
				It("It should update the CRD condition status as resolved", func() {
					matcher := testStructs.NewProjectClaimMatcher()
					conditions := &projectClaim.Status.Conditions
					*conditions = append(*conditions, gcpv1alpha1.Condition{})
					mockClient.EXPECT().Status().Return(mockStatusWriter)
					mockStatusWriter.EXPECT().Update(gomock.Any(), matcher)
					mockConditions.EXPECT().SetCondition(conditions, conditionType, corev1.ConditionFalse, "ReconcileErrorResolved", "").Times(1)
					adapter.SetProjectClaimCondition(reason, nil)
				})
			})
		})
	})
})

type stubStatus struct{}

func (stubStatus) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	return nil
}
func (stubStatus) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}
