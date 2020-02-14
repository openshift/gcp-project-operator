package projectreference

import (
	"fmt"

	"github.com/golang/mock/gomock"
	api "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	mocks "github.com/openshift/gcp-project-operator/pkg/util/mocks"
	testStructs "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	testProjectReferenceName = "testProjectReference"
	testNamespace            = "namespace"
)

var _ = Describe("ProjectReference controller reconcilation", func() {
	var (
		projectReference     *api.ProjectReference
		mockClient           *mocks.MockClient
		projectReferenceName types.NamespacedName
		reconciler           *ReconcileProjectReference
	)

	BeforeEach(func() {
		projectReferenceName = types.NamespacedName{
			Name:      testProjectReferenceName,
			Namespace: testNamespace,
		}
		projectReference = testStructs.NewProjectReferenceBuilder().GetProjectReference()
		ctrl := gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(ctrl)

		reconciler = &ReconcileProjectReference{
			mockClient,
			scheme.Scheme,
		}
	})
	Context("When project reference CR does not exist", func() {
		JustBeforeEach(func() {
			notFound := errors.NewNotFound(schema.GroupResource{}, projectReferenceName.Name)
			mockClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Return(notFound)
		})
		It("Returns without error", func() {
			_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When project reference can not be fetched", func() {
		var someError error
		JustBeforeEach(func() {
			someError = errors.NewInternalError(fmt.Errorf("Fake err"))
			mockClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Return(someError)
		})
		It("Returns the error", func() {
			_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
			Expect(err).To(Equal(someError))
		})
	})

	Context("Project id generation", func() {
		JustBeforeEach(func() {
			mockClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference)
		})

		Context("When project id is not set", func() {
			It("Updates the project id", func() {
				matcher := testStructs.NewProjectIdMatcher()
				mockClient.EXPECT().Update(gomock.Any(), matcher)

				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).NotTo(HaveOccurred())
				Expect(matcher.ActualProjectId).NotTo(Equal(""))
			})
		})

		Context("When the project id is set already", func() {
			BeforeEach(func() {
				projectReference.Spec.GCPProjectID = "Project-ID-already-set"
			})
			It("Doesn't change the project id", func() {
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).MaxTimes(0)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
