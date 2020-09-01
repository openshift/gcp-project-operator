/*


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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gcpmanagedopenshiftiov1alpha1 "github.com/openshift/gcp-project-operator/api/v1alpha1"
)

// ProjectReferenceReconciler reconciles a ProjectReference object
type ProjectReferenceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gcp.managed.openshift.io.my.domain,resources=projectreferences,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gcp.managed.openshift.io.my.domain,resources=projectreferences/status,verbs=get;update;patch

func (r *ProjectReferenceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("projectreference", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *ProjectReferenceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gcpmanagedopenshiftiov1alpha1.ProjectReference{}).
		Complete(r)
}
