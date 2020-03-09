package projectreference

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang/mock/gomock"
	api "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	mocks "github.com/openshift/gcp-project-operator/pkg/util/mocks"
	mockGCP "github.com/openshift/gcp-project-operator/pkg/util/mocks/gcpclient"
	testStructs "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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

type updaterNoErr struct {
}

func (u updaterNoErr) Update(ctx context.Context, obj runtime.Object) error {
	return nil
}

type updaterWithErr struct {
}

func (u updaterWithErr) Update(ctx context.Context, obj runtime.Object) error {
	return errors.New("Fake update Error")
}

var _ = Describe("ProjectReference controller reconcilation", func() {
	var (
		projectReference     *api.ProjectReference
		mockKubeClient       *mocks.MockClient
		projectReferenceName types.NamespacedName
		reconciler           *ReconcileProjectReference
		mockGCPClient        *mockGCP.MockClient
		projectClaim         *api.ProjectClaim
	)

	BeforeEach(func() {
		projectReferenceName = types.NamespacedName{
			Name:      testProjectReferenceName,
			Namespace: testNamespace,
		}
		projectReference = testStructs.NewProjectReferenceBuilder().GetProjectReference()
		projectClaim = testStructs.NewProjectClaimBuilder().GetProjectClaim()
		ctrl := gomock.NewController(GinkgoT())
		mockKubeClient = mocks.NewMockClient(ctrl)
		mockGCPClient = mockGCP.NewMockClient(ctrl)

		gcpBuilder := func(projectName string, authJSON []byte) (gcpclient.Client, error) {
			return mockGCPClient, nil
		}

		reconciler = &ReconcileProjectReference{
			mockKubeClient,
			scheme.Scheme,
			gcpBuilder,
		}
	})
	Context("When project reference CR does not exist", func() {
		JustBeforeEach(func() {
			notFound := k8serrs.NewNotFound(schema.GroupResource{}, projectReferenceName.Name)
			mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Return(notFound)
		})
		It("Returns without error", func() {
			_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When project reference can not be fetched", func() {
		var someError error
		JustBeforeEach(func() {
			someError = k8serrs.NewInternalError(fmt.Errorf("Fake err"))
			mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Return(someError)
		})
		It("Returns the error", func() {
			_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
			Expect(err).To(Equal(someError))
		})
	})

	Context("When Project Reference state is Error", func() {
		BeforeEach(func() {
			projectReference.Status.State = api.ProjectReferenceStatusError
		})
		It("Does not requeue", func() {
			mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1)
			_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When you cannot get credenitals", func() {
		It("Requeues with error", func() {
			gomock.InOrder(
				mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1),
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{}).Times(1),
			)
			_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When reference adapter cannot be created", func() {
		It("Requeues with error", func() {
			gomock.InOrder(
				mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1),
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{
					Data: map[string][]byte{"osServiceAccount.json": []byte("fakedata"), "key.json": []byte("fakedata")},
				}).Times(1),
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("Fake Error")).Times(1),
			)
			_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Project Reference State", func() {
		JustBeforeEach(func() {
			projectReference.Spec.GCPProjectID = "Project-ID-already-set"
			gomock.InOrder(
				mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1),
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{
					Data: map[string][]byte{"osServiceAccount.json": []byte("fakedata"), "key.json": []byte("fakedata")},
				}).Times(1),
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *projectClaim).Times(1),
			)
		})

		Context("When Reference State is Ready and Project Claim is Ready", func() {
			BeforeEach(func() {
				projectReference.Status.State = api.ProjectReferenceStatusReady
				projectClaim.Status.State = api.ClaimStatusReady
			})

			It("Does not reconcile", func() {
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When Reference State is Ready", func() {
			BeforeEach(func() {
				projectReference.Status.State = api.ProjectReferenceStatusReady
			})

			It("It does not reconcile", func() {
				mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
				mockKubeClient.EXPECT().Status().Return(updaterNoErr{})
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When Reference State is Ready and it fails to update", func() {
			BeforeEach(func() {
				projectReference.Status.State = api.ProjectReferenceStatusReady
			})

			It("It reconciles with error", func() {
				mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
				mockKubeClient.EXPECT().Status().Return(updaterWithErr{})
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When Reference State is empty and it failes to update", func() {
			BeforeEach(func() {
				projectReference.Status.State = ""
			})

			It("It reconciles with error", func() {
				mockKubeClient.EXPECT().Status().Return(updaterWithErr{})
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When Reference State is empty and it failes to requirement check and update", func() {
			BeforeEach(func() {
				projectReference.Status.State = ""
				projectClaim.Spec.Region = "bad region"
			})

			It("It reconciles with error", func() {
				mockKubeClient.EXPECT().Status().Return(updaterWithErr{})
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When Reference State is empty and it failes to requirement check", func() {
			BeforeEach(func() {
				projectReference.Status.State = ""
				projectClaim.Spec.Region = "bad region"
			})

			It("It does not reconcile", func() {
				mockKubeClient.EXPECT().Status().Return(updaterNoErr{})
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})

	Context("Project id generation", func() {
		JustBeforeEach(func() {
			gomock.InOrder(
				mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1),
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{
					Data: map[string][]byte{"osServiceAccount.json": []byte("fakedata"), "key.json": []byte("fakedata")},
				}).Times(1),
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *testStructs.NewProjectClaimBuilder().GetProjectClaim()).Times(1),
				mockKubeClient.EXPECT().Status().Return(updaterNoErr{}),
			)
		})

		Context("When project id is not set", func() {
			It("Updates the project id", func() {
				matcher := testStructs.NewProjectReferenceMatcher()
				mockKubeClient.EXPECT().Update(gomock.Any(), matcher).Times(1)
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).NotTo(HaveOccurred())
				Expect(matcher.ActualProjectReference.Spec.GCPProjectID).NotTo(Equal(""))
			})
		})

		// FContext("When the project id is set already", func() {
		// 	BeforeEach(func() {
		// 		projectReference.Spec.GCPProjectID = "Project-ID-already-set"
		// 	})
		// 	It("Doesn't change the project id", func() {
		// 		_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
		// 		mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any()).MaxTimes(0)
		// 		Expect(err).NotTo(HaveOccurred())
		// 	})
		// })

		Context("When gcpBuilder Fails", func() {
			JustBeforeEach(func() {
				gcpBuilder := func(projectName string, authJSON []byte) (gcpclient.Client, error) {
					return mockGCPClient, errors.New("Fakeerror")
				}
				reconciler.gcpClientBuilder = gcpBuilder
			})
			It("Requeues with error", func() {
				mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1)
				gomock.InOrder(
					mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1),
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{
						Data: map[string][]byte{"osServiceAccount.json": []byte("fakedata"), "key.json": []byte("fakedata")},
					}).Times(1),
				)
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).To(HaveOccurred())
			})
		})

	})

	Context("When processing Project", func() {
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
			gomock.InOrder(
				mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1),
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{
					Data: map[string][]byte{"osServiceAccount.json": []byte("fakedata"), "key.json": []byte("fakedata")},
				}).Times(1),
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *testStructs.NewProjectClaimBuilder().GetProjectClaim()).Times(1),
			)
		})

		Context("When it fails to get Parent Folder ID", func() {
			It("requeues with error", func() {
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
					Data: map[string]string{},
				})
				mockKubeClient.EXPECT().Status().Return(updaterNoErr{})
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
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
					mockKubeClient.EXPECT().Status().Return(updaterNoErr{})
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the project is Inactive and fails to update", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "DELETE_REQUESTED", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					mockKubeClient.EXPECT().Status().Return(updaterWithErr{})
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the project is Inactive and fails to update", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "DELETE_REQUESTED", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					mockKubeClient.EXPECT().Status().Return(updaterWithErr{})
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the project is Inactive and fails to update", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
						Data: map[string]string{"orgParentFolderID": "Fake Folder ID"},
					})
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "DELETE_REQUESTED", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					mockKubeClient.EXPECT().Status().Return(updaterWithErr{})
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
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
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
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
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
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
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When the failing to update Status to Ready", func() {
				It("It requeues with error", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap).Times(2)
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					mockGCPClient.EXPECT().EnableAPI(gomock.Any(), gomock.Any()).AnyTimes()
					mockGCPClient.EXPECT().CreateCloudBillingAccount(gomock.Any(), gomock.Any()).Return(nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "Some Email"}, nil).Times(2)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "dGVzdAo="}, nil)
					mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
					mockKubeClient.EXPECT().Status().Return(updaterWithErr{})
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
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
					mockKubeClient.EXPECT().Status().Return(updaterNoErr{})
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).ToNot(HaveOccurred())
				})
			})

		})

	})

})
