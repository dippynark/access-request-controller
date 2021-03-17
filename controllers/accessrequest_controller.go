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
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	iamv1alpha1 "github.com/dippynark/access-request-controller/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// AccessRequestReconciler reconciles a AccessRequest object
type AccessRequestReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=iam.dippynark.co.uk,resources=accessrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=iam.dippynark.co.uk,resources=accessrequests/status,verbs=get;update;patch

func (r *AccessRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	log := r.Log.WithValues("accessrequest", req.NamespacedName)

	// Fetch the accessrequest instance
	accessRequest := &iamv1alpha1.AccessRequest{}
	if err := r.Client.Get(ctx, req.NamespacedName, accessRequest); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(accessRequest, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Always attempt to patch the kubernetes cluster object and status after each reconciliation
	defer func() {
		if err := patchHelper.Patch(ctx, accessRequest); err != nil {
			log.Error(err, "failed to patch AccessRequest")
			if rerr == nil {
				rerr = err
			}
		}
	}()

	return r.reconcile(ctx, accessRequest)
}

func (r *AccessRequestReconciler) createRoleBinding(ctx context.Context, accessRequest *iamv1alpha1.AccessRequest) (ctrl.Result, error) {

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      accessRequest.Name,
			Namespace: accessRequest.Namespace,
		},
		Subjects: accessRequest.Spec.Subjects,
		RoleRef:  accessRequest.Spec.RoleRef,
	}
	if err := controllerutil.SetControllerReference(accessRequest, roleBinding, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.Create(ctx, roleBinding)
}

func (r *AccessRequestReconciler) reconcile(ctx context.Context, accessRequest *iamv1alpha1.AccessRequest) (ctrl.Result, error) {

	// Default all conditions to unknown
	accessRequest.Status.Conditions = ensureAccessRequestConditionStatus(accessRequest.Status.Conditions, iamv1alpha1.AccessRequestApproved, v1.ConditionUnknown, "", "")
	accessRequest.Status.Conditions = ensureAccessRequestConditionStatus(accessRequest.Status.Conditions, iamv1alpha1.AccessRequestComplete, v1.ConditionUnknown, "", "")

	// Check approval
	if !accessRequest.Spec.Approved {
		return ctrl.Result{}, nil
	}

	// This situation should be ensured by the mutating admission webhook and checked by the
	// validating admission webhook
	if accessRequest.Status.ApprovedBy == "" || accessRequest.Status.ApprovalTime.IsZero() {
		return ctrl.Result{}, errors.New("accessrequest is approved but approvedBy and approvalTime status fields are not set")
	}
	accessRequest.Status.Conditions = ensureAccessRequestConditionStatus(accessRequest.Status.Conditions, iamv1alpha1.AccessRequestApproved, v1.ConditionTrue, "AccessRequestApproved", fmt.Sprintf("AccessRequest approved by %s", accessRequest.Status.ApprovedBy))

	roleBinding := &rbacv1.RoleBinding{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: accessRequest.Namespace,
		Name:      accessRequest.Name,
	}, roleBinding)
	if k8serrors.IsNotFound(err) {
		return r.createRoleBinding(ctx, accessRequest)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	currentTime := metav1.Now()
	accessRequest.Status.CompletionTime = &currentTime
	accessRequest.Status.Conditions = ensureAccessRequestConditionStatus(accessRequest.Status.Conditions, iamv1alpha1.AccessRequestComplete, v1.ConditionTrue, "RoleBindingCreated", fmt.Sprintf("RoleBinding %s created", accessRequest.Name))

	return ctrl.Result{}, nil
}

func (r *AccessRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iamv1alpha1.AccessRequest{}).
		Complete(r)
}
