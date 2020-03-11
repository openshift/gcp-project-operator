package projectclaim

import (
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testStructs "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	api "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	"github.com/openshift/gcp-project-operator/pkg/util/mocks"
)

var _ = Describe("ProjectclaimController", func() {
	var (
		projectClaimName        types.NamespacedName
		projectReferenceName    types.NamespacedName
		reconciler              *ReconcileProjectClaim
		mockClient              *mocks.MockClient
		mockStatusWriter        *mocks.MockStatusWriter
		projectClaim            *api.ProjectClaim
		projectReferenceMatcher testStructs.ProjectReferenceMatcher
		projectClaimMatcher     testStructs.ProjectClaimMatcher
		mockCtrl                *gomock.Controller
	)
	BeforeEach(func() {
		projectClaimName = types.NamespacedName{
			Name:      testStructs.TestProjectClaimName,
			Namespace: testStructs.TestNamespace,
		}
		projectReferenceName = types.NamespacedName{
			Name:      projectClaimName.Namespace + "-" + projectClaimName.Name,
			Namespace: "gcp-project-operator",
		}
		projectClaim = testStructs.NewProjectClaimBuilder().GetProjectClaim()
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(mockCtrl)
		mockStatusWriter = mocks.NewMockStatusWriter(mockCtrl)

		reconciler = &ReconcileProjectClaim{
			mockClient,
			scheme.Scheme,
		}

	})
	AfterEach(func() {
		mockCtrl.Finish()
	})
	Context("When a new ProjectClaim is reconciled", func() {
		Context("When the ProjectClaim does not exist", func() {
			JustBeforeEach(func() {
				notFound := errors.NewNotFound(schema.GroupResource{}, projectReferenceName.Name)
				mockClient.EXPECT().Get(gomock.Any(), projectClaimName, gomock.Any()).Return(notFound)
			})
			It("Returns without error", func() {
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectClaimName})
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("When the ProjectReference does not exist", func() {
			JustBeforeEach(func() {
				notFound := errors.NewNotFound(schema.GroupResource{}, projectReferenceName.Name)
				mockClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).Return(notFound)
				mockClient.EXPECT().Get(gomock.Any(), projectClaimName, gomock.Any()).SetArg(2, *projectClaim)
				mockClient.EXPECT().Update(gomock.Any(), &projectClaimMatcher)
				mockClient.EXPECT().Create(gomock.Any(), &projectReferenceMatcher).Times(1)
				mockClient.EXPECT().Status().Return(mockStatusWriter)
				mockStatusWriter.EXPECT().Update(gomock.Any(), &projectClaimMatcher)
			})

			It("Creates a ProjectReference with the correct name", func() {
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectClaimName})
				Expect(err).ToNot(HaveOccurred())
				Expect(projectReferenceMatcher.ActualProjectReference.GetName()).To(Equal(projectClaimName.Namespace + "-" + projectClaimName.Name))
			})

			It("Creates a ProjectReference that links to the ProjectClaim", func() {
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectClaimName})
				Expect(err).ToNot(HaveOccurred())
				Expect(projectReferenceMatcher.ActualProjectReference.Spec.ProjectClaimCRLink.Name).To(Equal(projectClaimName.Name))
				Expect(projectReferenceMatcher.ActualProjectReference.Spec.ProjectClaimCRLink.Namespace).To(Equal(projectClaimName.Namespace))
			})

			It("Creates a ProjectReference that contains the same legal entity as the ProjectClaim", func() {
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectClaimName})
				Expect(err).ToNot(HaveOccurred())
				Expect(projectReferenceMatcher.ActualProjectReference.Spec.LegalEntity).To(Equal(projectClaim.Spec.LegalEntity))
			})

			It("Creates a ProjectReferenceCRLink linking to the created ProjectReference", func() {
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectClaimName})
				Expect(err).ToNot(HaveOccurred())
				Expect(projectClaimMatcher.ActualProjectClaim.Spec.ProjectReferenceCRLink.Name).To(Equal(projectReferenceMatcher.ActualProjectReference.GetName()))
				Expect(projectClaimMatcher.ActualProjectClaim.Spec.ProjectReferenceCRLink.Namespace).To(Equal(projectReferenceMatcher.ActualProjectReference.GetNamespace()))
			})
			It("Updates the ProjectClaim status to Pending", func() {
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectClaimName})
				Expect(err).ToNot(HaveOccurred())
				Expect(projectClaimMatcher.ActualProjectClaim.Status.State).To(Equal(api.ClaimStatusPending))
			})
		})

		Context("When the ProjectReference exists", func() {
			JustBeforeEach(func() {
				projectReference := testStructs.NewProjectReferenceBuilder().WithNamespacedName(projectReferenceName).GetProjectReference()
				projectClaim.Spec.ProjectReferenceCRLink.Name = projectReferenceName.Name
				projectClaim.Spec.ProjectReferenceCRLink.Namespace = projectReferenceName.Namespace
				projectClaim.Status.State = api.ClaimStatusPending
				projectClaim.SetFinalizers([]string{ProjectClaimFinalizer})
				mockClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference)
				mockClient.EXPECT().Get(gomock.Any(), projectClaimName, gomock.Any()).SetArg(2, *projectClaim)
				mockClient.EXPECT().Status().Return(mockStatusWriter)
			})

			It("Updates the ProjectClaim status to PendingProject", func() {
				mockStatusWriter.EXPECT().Update(gomock.Any(), &projectClaimMatcher)
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectClaimName})
				Expect(err).ToNot(HaveOccurred())
				Expect(projectClaimMatcher.ActualProjectClaim.Status.State).To(Equal(api.ClaimStatusPendingProject))
			})
		})

		Context("When the ProjectClaim gets deleted", func() {
			BeforeEach(func() {
				deletionTime := metav1.NewTime(time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC))
				projectClaim.SetDeletionTimestamp(&deletionTime)
				projectClaim.Spec.ProjectReferenceCRLink.Name = projectReferenceName.Name
				projectClaim.Spec.ProjectReferenceCRLink.Namespace = projectReferenceName.Namespace
				mockClient.EXPECT().Get(gomock.Any(), projectClaimName, gomock.Any()).SetArg(2, *projectClaim)
				projectReference := testStructs.NewProjectReferenceBuilder().WithNamespacedName(projectReferenceName).GetProjectReference()
				mockClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference)
			})

			It("will also delete the linked ProjectReference", func() {
				mockClient.EXPECT().Delete(gomock.Any(), &projectReferenceMatcher).Times(1)
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectClaimName})
				Expect(err).ToNot(HaveOccurred())
				Expect(projectReferenceMatcher.ActualProjectReference.GetName()).To(Equal(projectClaim.Spec.ProjectReferenceCRLink.Name))
				Expect(projectReferenceMatcher.ActualProjectReference.GetNamespace()).To(Equal(projectClaim.Spec.ProjectReferenceCRLink.Namespace))
			})
		})
	})
})
