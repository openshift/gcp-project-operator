package projectclaim_test

import (
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/openshift/gcp-project-operator/pkg/controller/projectclaim"
	testStructs "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	api "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	"github.com/openshift/gcp-project-operator/pkg/util/mocks"
	mockclaim "github.com/openshift/gcp-project-operator/pkg/util/mocks/projectclaim"
)

var _ = Describe("ProjectclaimController", func() {
	var (
		reconciler *ReconcileProjectClaim
		mockClient *mocks.MockClient
		mockCtrl   *gomock.Controller
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(mockCtrl)

		reconciler = NewReconcileProjectClaim(
			mockClient,
			scheme.Scheme,
		)

	})
	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("Reconcile", func() {
		var (
			projectClaimName     types.NamespacedName
			projectReferenceName types.NamespacedName
		)
		Context("When the ProjectClaim does not exist", func() {
			BeforeEach(func() {
				projectClaimName = types.NamespacedName{
					Name:      testStructs.TestProjectClaimName,
					Namespace: testStructs.TestNamespace,
				}
				projectReferenceName = types.NamespacedName{
					Name:      projectClaimName.Namespace + "-" + projectClaimName.Name,
					Namespace: "gcp-project-operator",
				}
				notFound := errors.NewNotFound(schema.GroupResource{}, projectReferenceName.Name)
				mockClient.EXPECT().Get(gomock.Any(), projectClaimName, gomock.Any()).Return(notFound)
			})
			It("Returns without error", func() {
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectClaimName})
				Expect(err).NotTo(HaveOccurred())
			})
		})

	})

	Context("ReconcileHandler", func() {
		var (
			mockAdapter *mockclaim.MockCustomResourceAdapter
		)
		Context("When the ProjectClaim is newly created", func() {
			BeforeEach(func() {
				mockAdapter = mockclaim.NewMockCustomResourceAdapter(mockCtrl)
				mockAdapter.EXPECT().EnsureProjectReferenceExists().Return(nil)
				mockAdapter.EXPECT().IsProjectClaimDeletion().Return(false)
				mockAdapter.EXPECT().EnsureProjectClaimInitialized().Return(ObjectUnchanged, nil)
				mockAdapter.EXPECT().EnsureProjectClaimState(api.ClaimStatusPending).Return(nil)
			})

			Context("When the ProjectReferenceLink does not exist", func() {
				It("Creates a ProjectReference, Links reference, sets status to Pending, and does not requeue", func() {
					mockAdapter.EXPECT().EnsureProjectReferenceLink().Return(ObjectModified, nil)
					res, err := reconciler.ReconcileHandler(mockAdapter)
					Expect(err).ToNot(HaveOccurred())
					Expect(res.Requeue).To(Equal(false))
					Expect(res.RequeueAfter).To(Equal(0 * time.Second))
				})
			})

			Context("When the ProjectReferenceLink exists", func() {
				BeforeEach(func() {
					mockAdapter.EXPECT().EnsureProjectReferenceLink().Return(ObjectUnchanged, nil)

				})
				Context("When the Finalizer does not exist", func() {
					It("Adds the finalizer and does not requeue", func() {
						mockAdapter.EXPECT().EnsureFinalizer().Return(ObjectModified, nil)
						res, err := reconciler.ReconcileHandler(mockAdapter)
						Expect(err).ToNot(HaveOccurred())
						Expect(res.Requeue).To(Equal(false))
						Expect(res.RequeueAfter).To(Equal(0 * time.Second))
					})
				})

				Context("When the finalizer exists", func() {
					BeforeEach(func() {
						mockAdapter.EXPECT().EnsureFinalizer().Return(ObjectUnchanged, nil)
					})

					It("Sets the state to PendingProject", func() {
						mockAdapter.EXPECT().EnsureProjectClaimState(api.ClaimStatusPendingProject)
						res, err := reconciler.ReconcileHandler(mockAdapter)
						Expect(err).ToNot(HaveOccurred())
						Expect(res.Requeue).To(Equal(false))
						Expect(res.RequeueAfter).To(Equal(0 * time.Second))
					})
				})
			})
		})

		Context("When the ProjectClaim gets deleted", func() {
			BeforeEach(func() {
				mockAdapter.EXPECT().IsProjectClaimDeletion().Return(true)
			})

			It("finalizes the projectclaim", func() {
				mockAdapter.EXPECT().FinalizeProjectClaim().Return(ObjectModified, nil)
				_, err := reconciler.ReconcileHandler(mockAdapter)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
