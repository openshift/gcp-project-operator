package projectreference_test

import (
	"errors"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/openshift/gcp-project-operator/pkg/configmap"
	"github.com/openshift/gcp-project-operator/pkg/util"
	"github.com/openshift/gcp-project-operator/pkg/util/mocks"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/openshift/gcp-project-operator/controllers/projectreference"

	gcpv1alpha1 "github.com/openshift/gcp-project-operator/api/v1alpha1"
	mockconditions "github.com/openshift/gcp-project-operator/pkg/util/mocks/condition"
	mockGCP "github.com/openshift/gcp-project-operator/pkg/util/mocks/gcpclient"
	testStructs "github.com/openshift/gcp-project-operator/pkg/util/mocks/structs"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	fakeError            = errors.New("fakeError")
	stopProcessingResult = util.OperationResult{
		RequeueDelay:   0,
		RequeueRequest: false,
		CancelRequest:  true,
	}

	continueProcessingResult = util.OperationResult{
		RequeueDelay:   0,
		RequeueRequest: false,
		CancelRequest:  false,
	}
)

var _ = Describe("ProjectreferenceAdapter", func() {
	var (
		adapter          *ReferenceAdapter
		projectReference *gcpv1alpha1.ProjectReference
		mockKubeClient   *mocks.MockClient
		mockGCPClient    *mockGCP.MockClient
		mockConditions   *mockconditions.MockConditions
		mockStatusWriter *mocks.MockStatusWriter
		projectClaim     *gcpv1alpha1.ProjectClaim
		err              error
		mockCtrl         *gomock.Controller
		configMap        configmap.OperatorConfigMap
	)

	BeforeEach(func() {
		projectReference = testStructs.NewProjectReferenceBuilder().GetProjectReference()
		projectClaim = testStructs.NewProjectClaimBuilder().GetProjectClaim()
		mockCtrl = gomock.NewController(GinkgoT())
		mockStatusWriter = mocks.NewMockStatusWriter(mockCtrl)
		mockKubeClient = mocks.NewMockClient(mockCtrl)
		mockGCPClient = mockGCP.NewMockClient(mockCtrl)
		mockConditions = mockconditions.NewMockConditions(mockCtrl)
		configMap = configmap.OperatorConfigMap{
			BillingAccount: "fake-account",
			ParentFolderID: "fake-folderID",
		}
	})
	AfterEach(func() {
		mockCtrl.Finish()
	})
	JustBeforeEach(func() {
		claimLink := types.NamespacedName{Name: projectReference.Spec.ProjectClaimCRLink.Name, Namespace: projectReference.Spec.ProjectClaimCRLink.Namespace}
		mockKubeClient.EXPECT().Get(gomock.Any(), claimLink, gomock.Any()).SetArg(2, *projectClaim)
		adapter, err = NewReferenceAdapter(projectReference, logf.Log.WithName("Test Logger"), mockKubeClient, mockGCPClient, mockConditions, configMap)
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
				projectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusCreating
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
				projectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusReady
			})

			Context("When ProjectClaim is in Ready state", func() {
				BeforeEach(func() {
					projectClaim.Status.State = gcpv1alpha1.ClaimStatusReady
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
						projectClaim.Status.State = gcpv1alpha1.ClaimStatusPending
						projectClaim.Spec.GCPProjectID = ""
						projectReference.Spec.GCPProjectID = "fake-gcp-project"

					})

					Context("When availability zones are not set", func() {
						BeforeEach(func() {
							mockGCPClient.EXPECT().ListAvailabilityZones(gomock.Any(), gomock.Any()).Return([]string{"zone1", "zone2", "zone3"}, nil)
							mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1)
							mockConditions.EXPECT().SetCondition(gomock.Any(), gcpv1alpha1.ConditionComputeApiReady, corev1.ConditionTrue, "QueryAvailabilityZonesSucceeded", "ComputeAPI ready, successfully queried availability zones").Times(1)
						})
						It("updates the ProjectClaim with availability zones", func() {
							_, err := EnsureProjectClaimReady(adapter)
							Expect(err).NotTo(HaveOccurred())
							Expect(adapter.ProjectClaim.Spec.AvailabilityZones).To(Equal([]string{"zone1", "zone2", "zone3"}))
						})

					})

					Context("When availability zones are set but GCPProjectID are not", func() {
						BeforeEach(func() {
							mockGCPClient.EXPECT().ListAvailabilityZones(gomock.Any(), gomock.Any()).Return([]string{"zone1", "zone2", "zone3"}, nil)
							mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any())
							mockConditions.EXPECT().SetCondition(gomock.Any(), gcpv1alpha1.ConditionComputeApiReady, corev1.ConditionTrue, "QueryAvailabilityZonesSucceeded", "ComputeAPI ready, successfully queried availability zones").Times(1)
						})
						It("updates the ProjectClaim with availability zones", func() {
							_, err := EnsureProjectClaimReady(adapter)
							Expect(err).NotTo(HaveOccurred())
							Expect(adapter.ProjectClaim.Spec.AvailabilityZones).To(Equal([]string{"zone1", "zone2", "zone3"}))
						})

					})
					Context("When availability zones are set already", func() {
						BeforeEach(func() {
							projectClaim.Spec.AvailabilityZones = []string{"zone1", "zone2", "zone3"}
							projectClaim.Spec.GCPProjectID = "fake-id"
							mockKubeClient.EXPECT().Status().Return(mockStatusWriter).Times(1)
							mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1)
						})

						It("sets state to Ready", func() {
							_, err := EnsureProjectClaimReady(adapter)
							Expect(err).NotTo(HaveOccurred())
							Expect(adapter.ProjectClaim.Status.State).To(Equal(gcpv1alpha1.ClaimStatusReady))

						})
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
						projectClaim.Status.State = gcpv1alpha1.ClaimStatusPending
						projectClaim.Spec.GCPProjectID = ""
						projectReference.Spec.GCPProjectID = "fake-gcp-project"

						conditionFound = false
						mockConditions.EXPECT().SetCondition(gomock.Any(), gcpv1alpha1.ConditionComputeApiReady, corev1.ConditionFalse, "QueryAvailabilityZonesFailed", "ComputeAPI not yet ready, couldn't query availability zones").Times(1)
						mockGCPClient.EXPECT().ListAvailabilityZones(gomock.Any(), gomock.Any()).Return([]string{}, errors.New("googleapi: Error 403: Access Not Configured. Compute Engine API has not been used in project"))
						mockKubeClient.EXPECT().Status().Return(mockStatusWriter)
						mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
					})

					Context("When compute API is not ready and no condition is set, yet", func() {
						It("does not return an error", func() {
							_, err := EnsureProjectClaimReady(adapter)
							Expect(err).NotTo(HaveOccurred())
							Expect(adapter.ProjectClaim.Status.State).NotTo(Equal(gcpv1alpha1.ClaimStatusReady))
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
						})
						It("does not return an error", func() {
							_, err := EnsureProjectClaimReady(adapter)
							Expect(err).NotTo(HaveOccurred())
							Expect(adapter.ProjectClaim.Status.State).NotTo(Equal(gcpv1alpha1.ClaimStatusReady))
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
							Expect(adapter.ProjectClaim.Status.State).NotTo(Equal(gcpv1alpha1.ClaimStatusReady))
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

	Context("EnsureProjectCreated", func() {

		Context("When CCS project", func() {
			JustBeforeEach(func() {
				projectReference.Spec.CCS = true
			})

			It("it continues processing", func() {
				result, err := EnsureProjectCreated(adapter)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(continueProcessingResult))
			})
		})

		Context("When non-CCS project", func() {
			Context("When it fails to create Project", func() {

				Context("When it fails to get project", func() {
					It("It requeues with error", func() {
						mockGCPClient.EXPECT().ListProjects().Return(nil, fakeError)
						_, err := EnsureProjectCreated(adapter)
						Expect(err).To(HaveOccurred())
					})
				})

				Context("When the lifecycleStatus is LIFECYCLE_STATE_UNSPECIFIED", func() {
					It("It requeues with error", func() {
						mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "foo", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
						_, err := EnsureProjectCreated(adapter)
						Expect(err).To(HaveOccurred())
					})
				})

				Context("When the lifecycleStatus is DELETE_REQUESTED and fails to update projectReference status", func() {
					It("It requeues with error", func() {
						mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "DELETE_REQUESTED", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
						mockKubeClient.EXPECT().Status().Return(mockStatusWriter)
						mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Return(fakeError)
						_, err := EnsureProjectCreated(adapter)
						Expect(err).To(HaveOccurred())
					})
				})

				Context("When the project is inactive and update projectReference status successfully", func() {
					It("It requeues with error", func() {
						mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "DELETE_REQUESTED", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
						mockKubeClient.EXPECT().Status().Return(mockStatusWriter)
						mockStatusWriter.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
						result, err := EnsureProjectCreated(adapter)
						Expect(err).ToNot(HaveOccurred())
						Expect(result).To(Equal(stopProcessingResult))
					})
				})

				Context("When the project doesn't exist and fails to create one", func() {

					Context("When fails to clear projectID", func() {
						It("It requeues with error", func() {
							mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: "foo"}}, nil)
							mockGCPClient.EXPECT().CreateProject(gomock.Any(), gomock.Any()).Return(nil, fakeError)
							mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(fakeError)
							_, err := EnsureProjectCreated(adapter)
							Expect(err).To(HaveOccurred())
							Expect(strings.Contains(err.Error(), "could not clear project ID")).To(BeTrue())
						})
					})

					Context("When it clears projectID successfully", func() {
						It("It requeues with error", func() {
							mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: "foo"}}, nil)
							mockGCPClient.EXPECT().CreateProject(gomock.Any(), gomock.Any()).Return(nil, fakeError)
							mockKubeClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
							_, err := EnsureProjectCreated(adapter)
							Expect(err).To(HaveOccurred())
							Expect(strings.Contains(err.Error(), "could not clear project ID")).To(BeFalse())
							Expect(strings.Contains(err.Error(), "could not create project. Parent Folder ID")).To(BeTrue())
						})
					})
				})
			})

			Context("When it fails to configure Billing API", func() {
				Context("When it fails to list APIs", func() {
					It("It requeues with error", func() {
						mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
						mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(nil, fakeError)
						_, err := EnsureProjectCreated(adapter)
						Expect(err).To(HaveOccurred())
					})
				})

				Context("When it fails to enable Billing API", func() {
					It("It requeues with error", func() {
						mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
						mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return([]string{"foo"}, nil)
						mockGCPClient.EXPECT().EnableAPI(gomock.Any(), gomock.Any()).Return(fakeError)
						_, err := EnsureProjectCreated(adapter)
						Expect(err).To(HaveOccurred())
						Expect(strings.Contains(err.Error(), "Error enabling cloudbilling.googleapis.com api for project")).To(BeTrue())
					})
				})

				Context("When it fails to create Cloud Billing account", func() {
					It("It requeues with error", func() {
						mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: "ACTIVE", ProjectId: projectReference.Spec.GCPProjectID}}, nil)
						mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return([]string{"cloudbilling.googleapis.com"}, nil)
						mockGCPClient.EXPECT().CreateCloudBillingAccount(gomock.Any(), gomock.Any()).Return(fakeError)
						_, err := EnsureProjectCreated(adapter)
						Expect(err).To(HaveOccurred())
						Expect(strings.Contains(err.Error(), "error creating CloudBilling")).To(BeTrue())
					})
				})

			})
		})

	})

	Context("EnsureProjectConfigured", func() {
		JustBeforeEach(func() {
			projectReference.Spec.GCPProjectID = "Some fake id"
			projectReference.Status.State = gcpv1alpha1.ProjectReferenceStatusCreating
		})

		Context("When it fails to configure APIS", func() {
			Context("When it fails to list APIs", func() {
				It("It requeues with error", func() {
					mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return([]string{}, fakeError)
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When it fails to enable APIs", func() {
				It("It requeues with error", func() {
					mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return([]string{}, nil)
					mockGCPClient.EXPECT().EnableAPI(gomock.Any(), gomock.Any()).Return(fakeError)
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("When it fails to configure Service Accounts", func() {
			Context("When it fails to get Service Accounts", func() {
				Context("When it fails to create Service Account", func() {
					Context("When it fails to create Service Account with fakeError", func() {
						It("It requeues with error", func() {
							mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
							mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(nil, fakeError)
							mockGCPClient.EXPECT().CreateServiceAccount(gomock.Any(), gomock.Any()).Return(nil, fakeError)
							_, err := EnsureProjectConfigured(adapter)
							Expect(err).To(HaveOccurred())
						})
					})

					Context("When it fails to create Service Account with matchesAlreadyExistsError", func() {
						It("It requeues with delay", func() {
							mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
							mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(nil, fakeError)
							mockGCPClient.EXPECT().CreateServiceAccount(gomock.Any(), gomock.Any()).Return(nil, errors.New("googleapi: Error 409:foo"))
							result, err := EnsureProjectConfigured(adapter)
							Expect(err).ToNot(HaveOccurred())
							Expect(result).To(Equal(util.OperationResult{
								RequeueDelay:   30 * time.Second,
								RequeueRequest: true,
								CancelRequest:  false,
							}))
						})
					})
				})
			})

			Context("When it fails to configure IAM policy", func() {
				Context("When it fails to get IAM Policy", func() {
					It("It requeues with error", func() {
						mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
						mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
						mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(nil, fakeError)
						_, err := EnsureProjectConfigured(adapter)
						Expect(err).To(HaveOccurred())
					})
				})

				Context("When it fails to set IAM Policy", func() {
					It("It requeues with error", func() {
						mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
						mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
						mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
						mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(nil, fakeError)
						_, err := EnsureProjectConfigured(adapter)
						Expect(err).To(HaveOccurred())
					})
				})
			})
		})

		Context("When it fails to create credentials", func() {
			Context("When it fails to get Service Account", func() {
				It("It requeues with error", func() {
					mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(nil, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeError)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(nil, fakeError)
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When it fails to create Service Account Key", func() {
				It("It requeues with error", func() {
					mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(nil, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeError)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
					mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(nil, fakeError)
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When it fails to create secret", func() {
				It("It requeues with error", func() {
					mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(nil, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeError)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
					mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "YWRtaW4="}, nil)
					mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(fakeError)
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("When it create credentials successfully", func() {
			Context("Credential Secret already exists", func() {
				It("Continue execute", func() {
					mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(nil, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("Create a secret successfully", func() {
				It("Continue execute", func() {
					mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(nil, nil)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeError)
					mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
					mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "YWRtaW4="}, nil)
					mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("When ccsConsoleAccess configured", func() {
			JustBeforeEach(func() {
				mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
				mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
				mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
				mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(nil, nil)
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeError)
				mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
				mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "YWRtaW4="}, nil)
				mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

				adapter.OperatorConfig.CCSConsoleAccess = []string{"example-group@xxx.com"}
			})

			Context("When it is a non CCS project", func() {
				It("nothing to do", func() {
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("When it is a CCS project", func() {
				JustBeforeEach(func() {
					projectReference.Spec.CCS = true
				})

				Context("When only one ccsConsoleAccessAccount are configured", func() {
					It("It doesn't need to create a service account", func() {
						mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
						mockGCPClient.EXPECT().SetIamPolicy(gomock.Any())
						_, err := EnsureProjectConfigured(adapter)
						Expect(err).ToNot(HaveOccurred())
					})
				})

				Context("When multiple ccsConsoleAccessAccount are configured", func() {
					JustBeforeEach(func() {
						adapter.OperatorConfig.CCSConsoleAccess = []string{"foo", "bar"}
					})
					It("repeat the process", func() {
						mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
						mockGCPClient.EXPECT().SetIamPolicy(gomock.Any())
						mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
						mockGCPClient.EXPECT().SetIamPolicy(gomock.Any())
						_, err := EnsureProjectConfigured(adapter)
						Expect(err).ToNot(HaveOccurred())
					})
				})
			})
		})

		Context("When ccsReadOnlyConsoleAccess configured", func() {
			JustBeforeEach(func() {
				mockGCPClient.EXPECT().ListAPIs(gomock.Any()).Return(OSDRequiredAPIS, nil)
				mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
				mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
				mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Return(nil, nil)
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeError)
				mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: "foo"}, nil)
				mockGCPClient.EXPECT().CreateServiceAccountKey(gomock.Any()).Return(&iam.ServiceAccountKey{PrivateKeyData: "YWRtaW4="}, nil)
				mockKubeClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

				adapter.OperatorConfig.CCSReadOnlyConsoleAccess = []string{"example-group@xxx.com"}
			})

			Context("When it is a non CCS project", func() {
				It("nothing to do", func() {
					_, err := EnsureProjectConfigured(adapter)
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("When it is a CCS project", func() {
				JustBeforeEach(func() {
					projectReference.Spec.CCS = true
				})

				Context("When only one ccsReadOnlyConsoleAccessAccount are configured", func() {
					It("It doesn't need to create a service account", func() {
						mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
						mockGCPClient.EXPECT().SetIamPolicy(gomock.Any())
						_, err := EnsureProjectConfigured(adapter)
						Expect(err).ToNot(HaveOccurred())
					})
				})

				Context("When multiple ccsReadOnlyConsoleAccessAccount are configured", func() {
					JustBeforeEach(func() {
						adapter.OperatorConfig.CCSReadOnlyConsoleAccess = []string{"foo", "bar"}
					})
					It("repeat the process", func() {
						mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
						mockGCPClient.EXPECT().SetIamPolicy(gomock.Any())
						mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
						mockGCPClient.EXPECT().SetIamPolicy(gomock.Any())
						_, err := EnsureProjectConfigured(adapter)
						Expect(err).ToNot(HaveOccurred())
					})
				})
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
				projectReference.SetFinalizers(util.Filter(projectReference.GetFinalizers(), FinalizerName))
				err := adapter.EnsureFinalizerDeleted()
				Expect(err).ToNot(HaveOccurred())
				Expect(projectReference.Finalizers).ToNot(ContainElement(FinalizerName))
			})
		})
	})

	Context("EnsureFinalizerAdded", func() {
		Context("When the finalizer does not exist", func() {
			It("adds the finalizer and updates the instance", func() {
				projectReference.SetFinalizers(util.Filter(projectReference.GetFinalizers(), FinalizerName))
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
			email        string
			testPolicy   cloudresourcemanager.Policy
		)
		BeforeEach(func() {
			projectReference.Spec.GCPProjectID = "fake-id"
			projectState = "ACTIVE"
			email = "Some Email"
			mockGCPClient.EXPECT().GetServiceAccount(gomock.Any()).Return(&iam.ServiceAccount{Email: email}, nil).Times(1)
			mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&cloudresourcemanager.Policy{}, nil)
			mockGCPClient.EXPECT().DeleteServiceAccount(gomock.Eq(email)).Return(nil).Times(1)
		})
		Context("When it's a non-CCS Project", func() {
			JustBeforeEach(func() {
				mockGCPClient.EXPECT().ListProjects().Return([]*cloudresourcemanager.Project{{LifecycleState: projectState, ProjectId: projectReference.Spec.GCPProjectID}}, nil)
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
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{}).Times(2)
					mockKubeClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1)
					err := adapter.EnsureProjectCleanedUp()
					Expect(err).NotTo(HaveOccurred())
				})
			})
			Context("When the lifecycleStatus is ACTIVE", func() {
				It("deletes the project", func() {
					mockGCPClient.EXPECT().DeleteProject(gomock.Any()).Times(1)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{}).Times(2)
					mockKubeClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(1)
					err := adapter.EnsureProjectCleanedUp()
					Expect(err).NotTo(HaveOccurred())
				})
			})
			Context("When it cannot delete the project", func() {
				It("returns an error", func() {
					mockGCPClient.EXPECT().DeleteProject(gomock.Any()).Times(1)
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{}).Times(2)
					mockKubeClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errors.New("Cannot delete the project"))
					err := adapter.EnsureProjectCleanedUp()
					Expect(err).To(HaveOccurred())
				})
			})

		})
		Context("When it's a CCS project", func() {
			BeforeEach(func() {
				projectReference.Spec.CCS = true
			})
			It("doesn't delete the project", func() {
				mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{}).Times(2)
				mockKubeClient.EXPECT().Delete(gomock.Any(), gomock.Any())
				err := adapter.EnsureProjectCleanedUp()
				Expect(err).NotTo(HaveOccurred())
			})
			Context("When SRE console access has been configured", func() {
				JustBeforeEach(func() {
					adapter.OperatorConfig.CCSConsoleAccess = []string{"SREAcc"}
					adapter.OperatorConfig.CCSReadOnlyConsoleAccess = []string{"SREViewOnly"}

					bindings := []*cloudresourcemanager.Binding{
						{Role: "role/admin", Members: []string{"group:SREAcc", "customerAcc"}},
						{Role: "role/viewer", Members: []string{"group:SREViewOnly", "customerAcc"}},
					}
					testPolicy = cloudresourcemanager.Policy{Bindings: bindings}

				})
				It("removes the Policies for that access", func() {
					mockKubeClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, corev1.Secret{}).Times(2)
					mockKubeClient.EXPECT().Delete(gomock.Any(), gomock.Any())
					// delete CCSConsoleAccess and CCSReadOnlyConsoleAccess
					mockGCPClient.EXPECT().GetIamPolicy(gomock.Any()).Return(&testPolicy, nil).Times(2)
					mockGCPClient.EXPECT().SetIamPolicy(gomock.Any()).Times(2)
					err := adapter.EnsureProjectCleanedUp()
					Expect(err).NotTo(HaveOccurred())
				})
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
