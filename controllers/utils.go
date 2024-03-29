package controllers

import (
	"context"
	"errors"

	iamv1alpha1 "github.com/dippynark/access-request-controller/api/v1alpha1"
	authv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	approveVerb                 = "approve"
	accessRequestResourcePlural = "accessrequests"
)

// ensureAccessRequestConditionStatus appends or updates an existing accessrequest condition of the
// given type with the given status value. Note that this function will not append to the conditions
// list if the new condition's status is false (because going from nothing to false is meaningless);
// it can, however, update the status condition to false.
func setConditionStatus(list []iamv1alpha1.AccessRequestCondition, cType iamv1alpha1.AccessRequestConditionType, status v1.ConditionStatus, reason, message string) []iamv1alpha1.AccessRequestCondition {
	for i := range list {
		if list[i].Type == cType {
			if list[i].Status != status || list[i].Reason != reason || list[i].Message != message {
				list[i].Status = status
				list[i].LastTransitionTime = metav1.Now()
				list[i].Reason = reason
				list[i].Message = message
				return list
			}
			return list
		}
	}
	// A condition with that type doesn't exist in the list.
	if status != v1.ConditionFalse {
		return append(list, newCondition(cType, status, reason, message))
	}
	return list
}

func newCondition(conditionType iamv1alpha1.AccessRequestConditionType, status v1.ConditionStatus, reason, message string) iamv1alpha1.AccessRequestCondition {
	return iamv1alpha1.AccessRequestCondition{
		Type:               conditionType,
		Status:             status,
		LastProbeTime:      metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

func (r *AccessRequestReconciler) checkAccess(ctx context.Context, accessRequest *iamv1alpha1.AccessRequest) (*authv1.SubjectAccessReview, error) {
	if accessRequest.Spec.Attributes == nil {
		return nil, errors.New("spec.attributes.approvedBy is nil")
	}
	sar := &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			User: accessRequest.Spec.Attributes.ApprovedBy,
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: accessRequest.Namespace,
				Name:      accessRequest.Name,
				Verb:      approveVerb,
				Group:     iamv1alpha1.GroupVersion.Group,
				Version:   iamv1alpha1.GroupVersion.Version,
				Resource:  accessRequestResourcePlural,
			},
		},
	}
	err := r.Create(ctx, sar)
	if err != nil {
		return nil, err
	}
	return sar, nil
}
