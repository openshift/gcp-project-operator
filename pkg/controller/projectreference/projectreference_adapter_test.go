package projectreference

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
})
