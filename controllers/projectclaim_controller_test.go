package controllers_test

import (
	"context"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/openshift/gcp-project-operator/pkg/util/mocks"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/openshift/gcp-project-operator/controllers"
	gcputil "github.com/openshift/gcp-project-operator/pkg/util"
	mockclaim "github.com/openshift/gcp-project-operator/pkg/util/mocks/projectclaim"
	testStructs "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
)

var _ = Describe("ProjectclaimController", func() {
	var (
		reconciler *ProjectClaimReconciler
		mockClient *mocks.MockClient
		mockCtrl   *gomock.Controller
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(mockCtrl)

		reconciler = &ProjectClaimReconciler{
			Client: mockClient,
			Scheme: scheme.Scheme,
		}

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
				_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: projectClaimName})
				Expect(err).NotTo(HaveOccurred())
			})
		})

	})

	Context("ReconcileHandler", func() {
		var (
			mockAdapter *mockclaim.MockCustomResourceAdapter
		)
		BeforeEach(func() {
			mockAdapter = mockclaim.NewMockCustomResourceAdapter(mockCtrl)
		})
		Context("When the ProjectClaim is newly created", func() {
			Context("When the ProjectClaim is fake", func() {
				It("Creates a Fake Secret, updates ProjectClaim with fake specs, sets status to Ready, and does not requeue", func() {
					mockAdapter.EXPECT().EnsureProjectClaimFakeProcessed().Return(gcputil.StopProcessing())
					res, err := reconciler.ReconcileHandler(mockAdapter)
					Expect(err).ToNot(HaveOccurred())
					Expect(res.Requeue).To(Equal(false))
					Expect(res.RequeueAfter).To(Equal(0 * time.Second))
				})
			})
			Context("When the ProjectClaim is not fake", func() {
				BeforeEach(func() {
					mockAdapter.EXPECT().EnsureProjectClaimFakeProcessed().Return(gcputil.ContinueProcessing())
					mockAdapter.EXPECT().EnsureProjectClaimDeletionProcessed().Return(gcputil.ContinueProcessing())
					mockAdapter.EXPECT().EnsureRegionSupported().Return(gcputil.ContinueProcessing())
					mockAdapter.EXPECT().EnsureProjectReferenceExists().Return(gcputil.ContinueProcessing())
					mockAdapter.EXPECT().EnsureProjectClaimInitialized().Return(gcputil.ContinueProcessing())
					mockAdapter.EXPECT().EnsureProjectClaimStatePending().Return(gcputil.ContinueProcessing())
				})

				Context("When the ProjectReferenceLink does not exist", func() {
					It("Creates a ProjectReference, Links reference, sets status to Pending, and does not requeue", func() {
						mockAdapter.EXPECT().EnsureProjectReferenceLink().Return(gcputil.StopProcessing())
						res, err := reconciler.ReconcileHandler(mockAdapter)
						Expect(err).ToNot(HaveOccurred())
						Expect(res.Requeue).To(Equal(false))
						Expect(res.RequeueAfter).To(Equal(0 * time.Second))
					})
				})

				Context("When the ProjectReferenceLink exists", func() {
					BeforeEach(func() {
						mockAdapter.EXPECT().EnsureProjectReferenceLink().Return(gcputil.ContinueProcessing())

					})
					Context("When the Finalizer does not exist", func() {
						It("Adds the finalizer and does not requeue", func() {
							mockAdapter.EXPECT().EnsureFinalizer().Return(gcputil.StopProcessing())
							res, err := reconciler.ReconcileHandler(mockAdapter)
							Expect(err).ToNot(HaveOccurred())
							Expect(res.Requeue).To(Equal(false))
							Expect(res.RequeueAfter).To(Equal(0 * time.Second))
						})
					})

					Context("When the finalizer exists", func() {
						BeforeEach(func() {
							mockAdapter.EXPECT().EnsureFinalizer().Return(gcputil.ContinueProcessing())
						})

						Context("When it's a CCS cluster", func() {
							It("Sets finalizer at the ccs secret", func() {
								mockAdapter.EXPECT().EnsureCCSSecretFinalizer().Return(gcputil.StopProcessing())
								res, err := reconciler.ReconcileHandler(mockAdapter)
								Expect(err).ToNot(HaveOccurred())
								Expect(res.Requeue).To(Equal(false))
								Expect(res.RequeueAfter).To(Equal(0 * time.Second))
							})

						})

						Context("When it's not a CCS cluster or finalizer is set", func() {
							BeforeEach(func() {
								mockAdapter.EXPECT().EnsureCCSSecretFinalizer().Return(gcputil.ContinueProcessing())
							})
							It("Sets the state to PendingProject", func() {
								mockAdapter.EXPECT().EnsureProjectClaimStatePendingProject()
								res, err := reconciler.ReconcileHandler(mockAdapter)
								Expect(err).ToNot(HaveOccurred())
								Expect(res.Requeue).To(Equal(false))
								Expect(res.RequeueAfter).To(Equal(0 * time.Second))
							})
						})

					})
				})
			})
		})

		Context("When the ProjectClaim gets deleted", func() {
			Context("When the ProjectClaim is fake", func() {
				It("finalizes the projectclaim", func() {
					mockAdapter.EXPECT().EnsureProjectClaimFakeProcessed().Return(gcputil.StopProcessing())
					_, err := reconciler.ReconcileHandler(mockAdapter)
					Expect(err).ToNot(HaveOccurred())
				})
			})
			Context("When the ProjectClaim is not fake", func() {
				It("finalizes the projectclaim", func() {
					mockAdapter.EXPECT().EnsureProjectClaimFakeProcessed().Return(gcputil.ContinueProcessing())
					mockAdapter.EXPECT().EnsureProjectClaimDeletionProcessed().Return(gcputil.StopProcessing())
					_, err := reconciler.ReconcileHandler(mockAdapter)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	})
})
