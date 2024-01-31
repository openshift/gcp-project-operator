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

package projectreference

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	goruntime "runtime"

	"github.com/go-logr/logr"
	"github.com/openshift/gcp-project-operator/pkg/configmap"
	"github.com/openshift/gcp-project-operator/pkg/gcpclient"
	"github.com/openshift/gcp-project-operator/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gcpv1alpha1 "github.com/openshift/gcp-project-operator/api/v1alpha1"
	"github.com/openshift/gcp-project-operator/pkg/condition"
	operrors "github.com/openshift/gcp-project-operator/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// Operator config
	operatorNamespace = "gcp-project-operator"

	// secret information
	orgGcpSecretName = "gcp-project-operator-credentials" //#nosec G101 -- not a secret, just name for it
)

// ProjectReferenceReconciler reconciles a ProjectReference object
type ProjectReferenceReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	GcpClientBuilder func(projectName string, authJSON []byte) (gcpclient.Client, error)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ProjectReference object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ProjectReferenceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	projectReference := &gcpv1alpha1.ProjectReference{}
	err := r.Get(context.TODO(), req.NamespacedName, projectReference)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	gcpClient, err := r.getGcpClient(projectReference, reqLogger)
	if err != nil {
		return ctrl.Result{}, err
	}

	cm, err := r.getConfigMap()
	if err != nil {
		return ctrl.Result{}, err
	}

	conditionManager := condition.NewConditionManager()
	adapter, err := NewReferenceAdapter(projectReference, reqLogger, r.Client, gcpClient, conditionManager, cm)
	if err != nil {
		err = operrors.Wrap(err, "could not create ReferenceAdapter")
		return ctrl.Result{}, err
	}

	result, err := r.ReconcileHandler(adapter, reqLogger)
	reason := "ReconcileError"
	_ = adapter.SetProjectReferenceCondition(reason, err)

	reqLogger.V(1).Info(fmt.Sprintf("Finished Reconcile. Error occured: %t, Requeing: %t, Delay: %d", err != nil, result.Requeue, result.RequeueAfter))
	return result, err
}

type ReferenceReconcileOperation func(*ReferenceAdapter) (util.OperationResult, error)

// ReconcileHandler reads that state of the cluster for a ProjectReference object and makes changes based on the state read
// and what is in the ProjectReference.Spec
func (r *ProjectReferenceReconciler) ReconcileHandler(adapter *ReferenceAdapter, reqLogger logr.Logger) (ctrl.Result, error) {
	operations := []ReferenceReconcileOperation{
		EnsureProjectReferenceInitialized, //Set conditions
		EnsureDeletionProcessed,           // Cleanup
		EnsureProjectClaimReady,           // Make projectReference  be processed based on state of ProjectClaim and Project Reference
		VerifyProjectClaimPending,         //only make changes to ProjectReference if ProjectClaim is pending
		EnsureProjectReferenceStatusCreating,
		EnsureProjectID,
		EnsureServiceAccountName,
		EnsureFinalizerAdded,
		EnsureProjectCreated,
		EnsureProjectConfigured,
		EnsureStateReady,
	}
	for _, operation := range operations {
		if log.Log.V(3).Enabled() {
			log.Log.V(3).Info("func", "name", strings.Split(goruntime.FuncForPC(reflect.ValueOf(operation).Pointer()).Name(), ".")[2])
		}
		result, err := operation(adapter)
		if err != nil || result.RequeueRequest {
			return ctrl.Result{RequeueAfter: result.RequeueDelay}, err
		}
		if result.CancelRequest {
			return ctrl.Result{}, nil
		}
	}
	return ctrl.Result{}, nil
}

// Returns a gcpClient, that uses the access credential Secret in the CCS project namespace or the operator namespace
func (r *ProjectReferenceReconciler) getGcpClient(projectReference *gcpv1alpha1.ProjectReference, logger logr.Logger) (gcpclient.Client, error) {
	credSecretNamespace := operatorNamespace
	credSecretName := orgGcpSecretName
	if projectReference.Spec.CCS {
		credSecretNamespace = projectReference.Spec.CCSSecretRef.Namespace
		credSecretName = projectReference.Spec.CCSSecretRef.Name
	}
	// Get org creds from secret
	creds, err := util.GetGCPCredentialsFromSecret(r.Client, credSecretNamespace, credSecretName)
	if err != nil {
		err = operrors.Wrap(err, fmt.Sprintf("could not get Creds from secret: %s, for namespace %s", credSecretName, credSecretNamespace))
		return nil, err
	}

	// Get gcpclient with creds
	gcpClient, err := r.GcpClientBuilder(projectReference.Spec.GCPProjectID, creds)
	if err != nil {
		return nil, operrors.Wrap(err, fmt.Sprintf("could not get gcp client with secret: %s, for namespace %s", credSecretName, credSecretNamespace))
	}

	return gcpClient, nil
}

func (r *ProjectReferenceReconciler) getConfigMap() (configmap.OperatorConfigMap, error) {
	operatorConfigMap, err := configmap.GetOperatorConfigMap(r.Client)
	if err != nil {
		return operatorConfigMap, operrors.Wrap(err, "could not find the OperatorConfigMap")
	}

	if err := configmap.ValidateOperatorConfigMap(operatorConfigMap); err != nil {
		return operatorConfigMap, operrors.Wrap(err, "configmap didn't get filled properly")
	}

	return operatorConfigMap, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReferenceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gcpv1alpha1.ProjectReference{}).
		Complete(r)
}
