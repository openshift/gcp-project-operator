package projectreference

import (
	"errors"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	api "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	gcpv1alpha1 "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	"github.com/openshift/gcp-project-operator/pkg/configmap"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	"github.com/openshift/gcp-project-operator/pkg/util/mocks"
	mockGCP "github.com/openshift/gcp-project-operator/pkg/util/mocks/gcpclient"
	testStructs "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
)

var (
	fakeError = errors.New("fakeError")
)

const (
	testProjectReferenceName = "testProjectReference"
	testNamespace            = "namespace"
)

var _ = Describe("ProjectReference controller reconcilation", func() {
	var (
		projectReference     *api.ProjectReference
		mockKubeClient       *mocks.MockClient
		projectReferenceName types.NamespacedName
		reconciler           *ReconcileProjectReference
		mockGCPClient        *mockGCP.MockClient
		projectClaim         *api.ProjectClaim
		configMap            corev1.ConfigMap
		mockCtrl             *gomock.Controller
		mockUpdater          *mocks.MockStatusWriter
	)

	BeforeEach(func() {
		projectReferenceName = types.NamespacedName{
			Name:      testProjectReferenceName,
			Namespace: testNamespace,
		}
		projectReference = testStructs.NewProjectReferenceBuilder().GetProjectReference()
		projectClaim = testStructs.NewProjectClaimBuilder().GetProjectClaim()
		mockCtrl = gomock.NewController(GinkgoT())
		mockKubeClient = mocks.NewMockClient(mockCtrl)
		mockGCPClient = mockGCP.NewMockClient(mockCtrl)
		mockUpdater = mocks.NewMockStatusWriter(mockCtrl)

		gcpBuilder := func(projectName string, authJSON []byte) (gcpclient.Client, error) {
			return mockGCPClient, nil
		}

		reconciler = &ReconcileProjectReference{
			mockKubeClient,
			scheme.Scheme,
			gcpBuilder,
		}
		configMap = corev1.ConfigMap{
			Data: map[string]string{
				configmap.OperatorConfigMapKey: `
billingAccount: fake-account
parentFolderID: fake-folder
`,
			},
		}
	})
	AfterEach(func() {
		mockCtrl.Finish()
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
			someError = k8serrs.NewInternalError(fakeError)
			mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Return(someError)
		})
		It("Returns the error", func() {
			_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
			Expect(err).To(Equal(someError))
		})
	})

	Context("When you cannot get credentials", func() {
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
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeError).Times(1),
			)
			_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Project Reference State", func() {
		JustBeforeEach(func() {
			projectReference.Spec.GCPProjectID = "Project-ID-already-set"
			projectReference.Spec.ServiceAccountName = "osd-managed-admin-abcdedgh"
			projectReference.Status.Conditions = []gcpv1alpha1.Condition{}
			gomock.InOrder(
				mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1),
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{
					Data: map[string][]byte{"osServiceAccount.json": []byte("fakedata"), "key.json": []byte("fakedata")},
				}).Times(1),
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap).Times(1),
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *projectClaim).Times(1),
			)
		})

		Context("When Reference State is Ready and Project Claim is Ready", func() {
			BeforeEach(func() {
				projectReference.Status.State = api.ProjectReferenceStatusReady
				projectClaim.Status.State = api.ClaimStatusReady
			})

			It("Does not reconcile", func() {
				mockGCPClient.EXPECT().GetProject(gomock.Any()).Return(&cloudresourcemanager.Project{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}, nil)
				mockGCPClient.EXPECT().CreateProjectLabels(gomock.Any(), gomock.Any()).Return(nil)
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When Reference State is Ready", func() {
			BeforeEach(func() {
				projectReference.Status.State = api.ProjectReferenceStatusReady
				projectClaim.Spec.AvailabilityZones = []string{"zone1", "zone2", "zone3"}
			})

			Context("When ProjectClaim GCPProjectID is empty", func() {
				BeforeEach(func() {
					projectClaim.Spec.AvailabilityZones = []string{}
					mockGCPClient.EXPECT().ListAvailabilityZones(gomock.Any(), gomock.Any()).Return([]string{"zone1", "zone2", "zone3"}, nil)
				})
				It("Updates ProjectClaim GCPPRojectID", func() {
					matcher := testStructs.NewProjectClaimMatcher()
					mockKubeClient.EXPECT().Update(gomock.Any(), matcher).Return(nil)
					mockKubeClient.EXPECT().Status().Return(mockUpdater).AnyTimes()
					mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes()
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).NotTo(HaveOccurred())
					Expect(matcher.ActualProjectClaim.Spec.AvailabilityZones).To(Equal([]string{"zone1", "zone2", "zone3"}))
				})
			})
			Context("When ProjectClaim GCPProjectID is empty", func() {
				It("Updates ProjectClaim GCPPRojectID", func() {
					matcher := testStructs.NewProjectClaimMatcher()
					mockKubeClient.EXPECT().Update(gomock.Any(), matcher).Return(nil)
					mockKubeClient.EXPECT().Status().Return(mockUpdater).AnyTimes()
					mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes()
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).NotTo(HaveOccurred())
					Expect(matcher.ActualProjectClaim.Spec.GCPProjectID).ToNot(Equal(""))
				})
			})

			Context("When ProjectClaim GCPProjectID is empty and it fails to Update ProjectClaim", func() {
				It("Reconciles with error", func() {
					mockKubeClient.EXPECT().Status().Return(mockUpdater)
					mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any())
					mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(fakeError)
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When ProjectClaim GCPProjectID is not empty", func() {
				BeforeEach(func() {
					projectClaim.Spec.GCPProjectID = "Not Empty"
					projectClaim.Spec.AvailabilityZones = []string{}
					mockGCPClient.EXPECT().ListAvailabilityZones(gomock.Any(), gomock.Any()).Return([]string{"zone1", "zone2", "zone3"}, nil)
				})

				It("It updates az and does not reconcile", func() {
					mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
					mockKubeClient.EXPECT().Status().Return(mockUpdater).AnyTimes()
					mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes()
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		Context("When Reference State is Ready and it fails to update", func() {
			BeforeEach(func() {
				projectReference.Status.State = api.ProjectReferenceStatusReady
				projectClaim.Spec.GCPProjectID = "fake-id"
				projectClaim.Spec.AvailabilityZones = []string{"zone1", "zone2", "zone3"}
			})

			It("It reconciles with error", func() {
				mockKubeClient.EXPECT().Status().Return(mockUpdater).Times(2)
				mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).Return(fakeError).Times(2)
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When Reference State is empty and it fails to update", func() {
			BeforeEach(func() {
				projectReference.Status.State = ""
				projectClaim.Spec.GCPProjectID = "fake-id"
				projectClaim.Spec.AvailabilityZones = []string{"zone1", "zone2", "zone3"}
			})

			It("It reconciles with error", func() {
				mockKubeClient.EXPECT().Status().Return(mockUpdater).Times(2)
				mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).Return(fakeError).Times(2)
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("Project id generation", func() {
		BeforeEach(func() {
			projectReference.Status.Conditions = []gcpv1alpha1.Condition{}
			projectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusCreating
			mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1)
			mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{
				Data: map[string][]byte{"osServiceAccount.json": []byte("fakedata"), "key.json": []byte("fakedata")},
			}).Times(1)
		})

		Context("When project id is not set", func() {
			It("Updates the project id", func() {
				matcher := testStructs.NewProjectReferenceMatcher()
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap)
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *projectClaim).AnyTimes()
				mockKubeClient.EXPECT().Update(gomock.Any(), matcher).Times(1)
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).NotTo(HaveOccurred())
				Expect(matcher.ActualProjectReference.Spec.GCPProjectID).NotTo(Equal(""))
			})
		})

		Context("When gcpBuilder Fails", func() {
			JustBeforeEach(func() {
				gcpBuilder := func(projectName string, authJSON []byte) (gcpclient.Client, error) {
					return mockGCPClient, fakeError
				}
				reconciler.gcpClientBuilder = gcpBuilder
			})
			It("Requeues with error", func() {
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When it fails to get operator configMap", func() {
			It("Requeues with error", func() {
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeError)
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When it fails to validate operator configMap", func() {
			It("Requeues with error", func() {
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.ConfigMap{
					Data: map[string]string{},
				})
				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("When project claim CR is not PendingProject", func() {
		BeforeEach(func() {
			projectClaim.Status.State = v1alpha1.ClaimStatusPending
			projectReference.Status.Conditions = []gcpv1alpha1.Condition{}
			mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1)
			mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{
				Data: map[string][]byte{"osServiceAccount.json": []byte("fakedata"), "key.json": []byte("fakedata")},
			}).Times(1)
			mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap)
			mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *projectClaim).Times(1)
		})
		It("Gets requeued after 5 seconds", func() {
			result, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(5 * time.Second))
		})
	})

	Context("When processing Project", func() {
		Context("When it is a CCS project", func() {
			JustBeforeEach(func() {
				projectReference.Spec.CCS = true
				projectReference.Spec.GCPProjectID = "Some fake id"
				projectReference.Spec.ServiceAccountName = "osd-managed-admin-abcdedgh"
				projectReference.Status.State = api.ProjectReferenceStatusCreating
				projectReference.Status.Conditions = []gcpv1alpha1.Condition{}
				projectReference.SetFinalizers([]string{FinalizerName})
				gomock.InOrder(
					mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1),
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
						SetArg(2, corev1.Secret{Data: map[string][]byte{"osServiceAccount.json": []byte("fakedata"), "key.json": []byte("fakedata")}}).
						Times(1),
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap),
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *testStructs.NewProjectClaimBuilder().GetProjectClaim()).Times(1),
				)
			})

			Context("When the failing to update Status to Ready", func() {
				It("It requeues with error", func() {
					mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "Some Email"}, nil).Times(2)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "dGVzdAo="}, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeError).Times(1)
					mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

					gomock.InOrder(
						mockKubeClient.EXPECT().Status().Return(mockUpdater),
						mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).Return(fakeError),

						// these are for the SetProjectReferenceCondition segment
						mockKubeClient.EXPECT().Status().Return(mockUpdater),
						mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil),
					)

					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When processes the project reference correctly", func() {
				It("It does not requeue", func() {
					mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "Some Email"}, nil).Times(2)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "dGVzdAo="}, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeError).Times(1)
					mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
					mockKubeClient.EXPECT().Status().Return(mockUpdater)
					mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("When it is a non-CCS project", func() {

			JustBeforeEach(func() {
				projectReference.Spec.GCPProjectID = "Some fake id"
				projectReference.Spec.ServiceAccountName = "osd-managed-admin-abcdedgh"
				projectReference.Status.State = api.ProjectReferenceStatusCreating
				projectReference.Status.Conditions = []gcpv1alpha1.Condition{}
				projectReference.SetFinalizers([]string{FinalizerName})
				gomock.InOrder(
					mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1),
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{
						Data: map[string][]byte{"osServiceAccount.json": []byte("fakedata"), "key.json": []byte("fakedata")},
					}).Times(1),
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap),
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, *testStructs.NewProjectClaimBuilder().GetProjectClaim()).Times(1),
				)
			})

			Context("When the failing to update Status to Ready", func() {
				It("It requeues with error", func() {
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					mockGCPClient.EXPECT().ListAPIs(gomock.Any())
					mockGCPClient.EXPECT().EnableAPI(gomock.Any(), gomock.Any()).AnyTimes()
					mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
					mockGCPClient.EXPECT().CreateCloudBillingAccount(gomock.Any(), gomock.Any()).Return(nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "Some Email"}, nil).Times(2)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "dGVzdAo="}, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeError).Times(1)
					mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

					gomock.InOrder(
						mockKubeClient.EXPECT().Status().Return(mockUpdater),
						mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).Return(fakeError),

						// these are for the SetProjectReferenceCondition segment
						mockKubeClient.EXPECT().Status().Return(mockUpdater),
						mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil),
					)
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When processes the project reference correctly", func() {
				It("It does not requeue", func() {
					mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
					mockGCPClient.EXPECT().ListAPIs(gomock.Any())
					mockGCPClient.EXPECT().EnableAPI(gomock.Any(), gomock.Any()).AnyTimes()
					mockGCPClient.EXPECT().CreateCloudBillingAccount(gomock.Any(), gomock.Any()).Return(nil)
					mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "Some Email"}, nil).Times(2)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "dGVzdAo="}, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeError).Times(1)
					mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
					mockKubeClient.EXPECT().Status().Return(mockUpdater)
					mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any())
					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

	})
	Context("Given there is a ProjectReference deletion request", func() {
		var (
			projects []*cloudresourcemanager.Project
		)

		BeforeEach(func() {
			projectReference.Spec.GCPProjectID = "fake-id"
			projectReference.Spec.ServiceAccountName = "osd-managed-admin-abcdedgh"
			projectReference.Status.Conditions = []gcpv1alpha1.Condition{}
			projects = []*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}
			deletionTime := metav1.NewTime(time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC))
			projectReference.SetDeletionTimestamp(&deletionTime)
		})

		JustBeforeEach(func() {
			mockKubeClient.EXPECT().Get(gomock.Any(), projectReferenceName, gomock.Any()).SetArg(2, *projectReference).Times(1)
			mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{
				Data: map[string][]byte{"osServiceAccount.json": []byte("fakedata"), "key.json": []byte("fakedata")},
			}).Times(1)
			mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, configMap)
			mockGCPClient.EXPECT().ListProjects().Return(projects, nil)
			mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		})

		Context("When GCP project ID is set", func() {
			BeforeEach(func() {
				projectReference.Spec.GCPProjectID = "fake-id"
			})

			Context("When cleanup succeeds", func() {
				It("does not requeue", func() {
					mockGCPClient.EXPECT().DeleteProject(gomock.Any()).Times(1)
					email := "Some Email"
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: email}, nil).Times(1)
					mockGCPClient.EXPECT().DeleteServiceAccount(gomock.Eq(email)).Return(nil).Times(1)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{}).Times(2)
					mockKubeClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("When project is in Error state", func() {
				BeforeEach(func() {
					projectReference.Status.State = "Error"
				})
				It("Does not requeue", func() {
					mockGCPClient.EXPECT().DeleteProject(gomock.Any()).Times(1)
					email := "Some Email"
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: email}, nil).Times(1)
					mockGCPClient.EXPECT().DeleteServiceAccount(gomock.Eq(email)).Return(nil).Times(1)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{}).Times(2)
					mockKubeClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

					_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("When cleanup fails", func() {
				It("Gets requeued after 5 seconds", func() {
					mockKubeClient.EXPECT().Status().Return(mockUpdater)
					mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any())
					mockGCPClient.EXPECT().DeleteProject(gomock.Any()).Times(1)
					email := "Some Email"
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: email}, nil).Times(1)
					mockGCPClient.EXPECT().DeleteServiceAccount(gomock.Eq(email)).Return(nil).Times(1)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{}).Times(2)
					mockKubeClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(fakeError)
					result, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
					Expect(err).To(HaveOccurred())
					Expect(result.RequeueAfter).To(Equal(5 * time.Second))
				})
			})
		})

		Context("When GCP project ID is empty", func() {
			BeforeEach(func() {
				projectReference.Spec.GCPProjectID = ""
				projects = []*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: "another-project-id"}}
			})

			It("Does not requeue", func() {
				email := "Some Email"
				mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: email}, nil).Times(1)
				mockGCPClient.EXPECT().DeleteServiceAccount(gomock.Eq(email)).Return(nil).Times(1)
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{}).Times(2)
				mockKubeClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

				_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: projectReferenceName})
				Expect(err).ToNot(HaveOccurred())

			})
		})
	})

})
