package main

import (
	"context"
	"fmt"

	iamv1alpha1 "github.com/dippynark/access-request-controller/api/v1alpha1"
	v1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const (
	accessRequestResourceSingular = "accessrequest"
	accessRequestResourcePlural   = "accessrequests"
	approveVerb                   = "approve"
)

func (h *serveMutateAccessRequestHandler) mutateAccessRequest(ar v1.AdmissionReview) *v1.AdmissionResponse {
	shouldPatchAccessRequest := func(accessRequest *iamv1alpha1.AccessRequest) bool {
		if accessRequest.Spec.Approved && accessRequest.Status.ApprovalTime.IsZero() {
			return true
		}
		return false
	}
	return h.applyAccessRequestPatch(ar, shouldPatchAccessRequest)
}

func (h *serveMutateAccessRequestHandler) applyAccessRequestPatch(ar v1.AdmissionReview, shouldPatchAccessRequest func(accessRequest *iamv1alpha1.AccessRequest) bool) *v1.AdmissionResponse {
	klog.V(2).Infof("mutating %s", accessRequestResourceSingular)
	accessRequestResource := metav1.GroupVersionResource{
		Group:    iamv1alpha1.GroupVersion.Group,
		Version:  iamv1alpha1.GroupVersion.Version,
		Resource: accessRequestResourcePlural,
	}
	if ar.Request.Resource != accessRequestResource {
		klog.Errorf("expect resource to be %s", accessRequestResource)
		return nil
	}

	raw := ar.Request.Object.Raw
	accessRequest := iamv1alpha1.AccessRequest{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &accessRequest); err != nil {
		klog.Error(err)
		return toV1AdmissionResponse(err)
	}
	reviewResponse := v1.AdmissionResponse{}
	reviewResponse.Allowed = true
	if shouldPatchAccessRequest(&accessRequest) {
		sar, err := h.checkAccess(ar, &accessRequest)
		if err != nil {
			klog.Errorf("failed to check access %s", err)
			return nil
		}

		if !sar.Status.Allowed {
			klog.Errorf("user %s is not allowed to approve access request %s/%s: %s", sar.Spec.User, accessRequest.Namespace, accessRequest.Name, sar.Status.Reason)
			return nil
		}

		reviewResponse.Patch = []byte(fmt.Sprintf(`[
	{"op":"add","path":"/status/approvedBy","value":"%s"},
	{"op":"add","path":"/status/approvalTime","value":"%s"}
]`, ar.Request.UserInfo.Username, metav1.Now().Time))
		pt := v1.PatchTypeJSONPatch
		reviewResponse.PatchType = &pt
	}
	return &reviewResponse
}

func (h *serveMutateAccessRequestHandler) checkAccess(ar v1.AdmissionReview, accessRequest *iamv1alpha1.AccessRequest) (*authv1.SubjectAccessReview, error) {
	sar := &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			User: ar.Request.UserInfo.Username,
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: "",
				Verb:      approveVerb,
				Group:     iamv1alpha1.GroupVersion.Group,
				Version:   iamv1alpha1.GroupVersion.Version,
				Resource:  accessRequestResourcePlural,
				Name:      accessRequest.Name,
			},
		},
	}
	sar, err := h.clientset.AuthorizationV1().SubjectAccessReviews().Create(context.TODO(), sar, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return sar, nil
}
