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

	iamv1alpha1 "github.com/dippynark/access-request-controller/api/v1alpha1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// AccessRequestReconciler reconciles a AccessRequest object
type AccessRequestReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=iam.dippynark.co.uk,resources=accessrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=iam.dippynark.co.uk,resources=accessrequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create

func (r *AccessRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	log := r.Log.WithValues("accessrequest", req.NamespacedName)
	log.Info("Reconciling")

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
		// Always attempt to patch the object and status after each reconciliation.
		// Patch ObservedGeneration only if the reconciliation completed successfully
		patchOpts := []patch.Option{}
		if rerr == nil {
			patchOpts = append(patchOpts, patch.WithStatusObservedGeneration{})
		}
		if err := patchHelper.Patch(ctx, accessRequest, patchOpts...); err != nil {
			rerr = kerrors.NewAggregate([]error{rerr, err})
		}
	}()

	return r.reconcile(ctx, accessRequest)
}

func (r *AccessRequestReconciler) approvalAllowed(ctx context.Context, accessRequest *iamv1alpha1.AccessRequest) (bool, error) {

	// Verify approval permissions
	sar, err := r.checkAccess(ctx, accessRequest)
	if err != nil {
		return false, err
	}

	if !sar.Status.Allowed || sar.Status.Denied {
		return false, nil
	}

	return true, nil
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
	log := r.Log.WithValues("accessrequest", fmt.Sprintf("%s/%s", accessRequest.Namespace, accessRequest.Name))

	// Default all conditions to unknown
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
	accessRequest.Status.Conditions = setConditionStatus(accessRequest.Status.Conditions, iamv1alpha1.AccessRequestApproved, v1.ConditionUnknown, "", "")
	accessRequest.Status.Conditions = setConditionStatus(accessRequest.Status.Conditions, iamv1alpha1.AccessRequestComplete, v1.ConditionUnknown, "", "")

	// Check approval
	if !accessRequest.Spec.Approved {
		accessRequest.Status.Conditions = setConditionStatus(accessRequest.Status.Conditions, iamv1alpha1.AccessRequestApproved, v1.ConditionFalse, "WaitingForApproval", "AccessRequest has not been approved")
		return ctrl.Result{}, nil
	}

	// TODO: This situation should be ensured by the mutating admission webhook and verified by the
	// validating admission webhook
	if accessRequest.Spec.Attributes == nil || accessRequest.Spec.Attributes.ApprovedBy == "" {
		return ctrl.Result{}, errors.New("accessrequest has been approved but the approvedBy attribute is not set")
	}
	accessRequest.Status.Conditions = setConditionStatus(accessRequest.Status.Conditions, iamv1alpha1.AccessRequestApproved, v1.ConditionTrue, "AccessRequestApproved", fmt.Sprintf("AccessRequest approved by %s", accessRequest.Spec.Attributes.ApprovedBy))

	// Verify whether the user who approved the accessrequest is allowed to approve it. This should be
	// validated by the validating webhook but we verify again to here to avoid TOCTOU race conditions
	approvalAllowed, err := r.approvalAllowed(ctx, accessRequest)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !approvalAllowed {
		message := fmt.Sprintf("%s is not allowed to approve AccessRequest", accessRequest.Spec.Attributes.ApprovedBy)
		accessRequest.Status.Conditions = setConditionStatus(accessRequest.Status.Conditions, iamv1alpha1.AccessRequestComplete, v1.ConditionFalse, "ApproverDenied", message)
		log.Info(message)
		return ctrl.Result{}, nil
	}

	// Get or create rolebinding
	roleBinding := &rbacv1.RoleBinding{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: accessRequest.Namespace,
		Name:      accessRequest.Name,
	}, roleBinding)
	if k8serrors.IsNotFound(err) {
		return r.createRoleBinding(ctx, accessRequest)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check rolebinding is controlled by accessrequest
	ref := metav1.GetControllerOf(roleBinding)
	if ref == nil || ref.UID != accessRequest.UID {
		accessRequest.Status.Conditions = setConditionStatus(accessRequest.Status.Conditions, iamv1alpha1.AccessRequestComplete, v1.ConditionFalse, "RoleBindingExists", fmt.Sprintf("RoleBinding %s exists but is not controlled by AccessRequest", roleBinding.Name))
		return ctrl.Result{}, nil
	}

	// TODO: check rolebinding matches accessrequest specification

	// Set completion time
	if accessRequest.Status.CompletionTime.IsZero() {
		currentTime := metav1.Now()
		accessRequest.Status.CompletionTime = &currentTime
	}

	accessRequest.Status.Conditions = setConditionStatus(accessRequest.Status.Conditions, iamv1alpha1.AccessRequestComplete, v1.ConditionTrue, "RoleBindingCreated", fmt.Sprintf("RoleBinding %s created", accessRequest.Name))

	return ctrl.Result{}, nil
}

func (r *AccessRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iamv1alpha1.AccessRequest{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
	// TODO: watch for roles and rolebindings in case approver becomes able to approve
}
