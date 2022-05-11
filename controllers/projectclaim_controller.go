/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gcpv1alpha1 "github.com/openshift/gcp-project-operator/api/v1alpha1"
	condition "github.com/openshift/gcp-project-operator/pkg/condition"
	gcputil "github.com/openshift/gcp-project-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

//go:generate mockgen -destination=../pkg/util/mocks/$GOPACKAGE/customeresourceadapter.go -package=$GOPACKAGE github.com/openshift/gcp-project-operator/controllers CustomResourceAdapter
type CustomResourceAdapter interface {
	EnsureProjectClaimFakeProcessed() (gcputil.OperationResult, error)
	EnsureProjectClaimDeletionProcessed() (gcputil.OperationResult, error)
	ProjectReferenceExists() (bool, error)
	EnsureProjectClaimInitialized() (gcputil.OperationResult, error)
	EnsureProjectClaimStatePending() (gcputil.OperationResult, error)
	EnsureProjectClaimStatePendingProject() (gcputil.OperationResult, error)
	EnsureRegionSupported() (gcputil.OperationResult, error)
	EnsureProjectReferenceExists() (gcputil.OperationResult, error)
	EnsureProjectReferenceLink() (gcputil.OperationResult, error)
	EnsureFinalizer() (gcputil.OperationResult, error)
	EnsureCCSSecretFinalizer() (gcputil.OperationResult, error)
	FinalizeProjectClaim() (ObjectState, error)
	SetProjectClaimCondition(gcpv1alpha1.ConditionType, string, error) (gcputil.OperationResult, error)
}

// ProjectClaimReconciler reconciles a ProjectClaim object
type ProjectClaimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=gcp.managed.openshift.io,resources=projectclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gcp.managed.openshift.io,resources=projectclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gcp.managed.openshift.io,resources=projectclaims/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ProjectClaim object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ProjectClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	// Fetch the ProjectClaim instance
	instance := &gcpv1alpha1.ProjectClaim{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	conditionManager := condition.NewConditionManager()
	adapter := NewProjectClaimAdapter(instance, reqLogger, r.Client, conditionManager)
	result, err := r.ReconcileHandler(adapter)
	reason := "ReconcileError"
	_, _ = adapter.SetProjectClaimCondition(gcpv1alpha1.ConditionError, reason, err)

	return result, err
}

type ReconcileOperation func() (gcputil.OperationResult, error)

// ReconcileHandler reads that state of the cluster for a ProjectClaim object and makes changes based on the state read
// and what is in the ProjectClaim.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ProjectClaimReconciler) ReconcileHandler(adapter CustomResourceAdapter) (ctrl.Result, error) {
	operations := []ReconcileOperation{
		adapter.EnsureProjectClaimFakeProcessed,
		adapter.EnsureProjectClaimDeletionProcessed,
		adapter.EnsureProjectClaimInitialized,
		adapter.EnsureRegionSupported,
		adapter.EnsureProjectClaimStatePending,
		adapter.EnsureProjectReferenceExists,
		adapter.EnsureProjectReferenceLink,
		adapter.EnsureFinalizer,
		adapter.EnsureCCSSecretFinalizer,
		adapter.EnsureProjectClaimStatePendingProject,
	}
	for _, operation := range operations {
		result, err := operation()
		if err != nil || result.RequeueRequest {
			return ctrl.Result{RequeueAfter: result.RequeueDelay}, err
		}
		if result.CancelRequest {
			return ctrl.Result{}, nil
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gcpv1alpha1.ProjectClaim{}).
		Complete(r)
}
