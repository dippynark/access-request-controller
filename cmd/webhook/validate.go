package main

import (
	"context"
	"fmt"

	iamv1alpha1 "github.com/dippynark/access-request-controller/api/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func (h *serveValidateAccessRequestHandler) validateAccessRequest(ar v1.AdmissionReview) *v1.AdmissionResponse {
	klog.V(2).Infof("validating %s", accessRequestResourceSingular)

	accessRequestResource := metav1.GroupVersionResource{
		Group:    iamv1alpha1.GroupVersion.Group,
		Version:  iamv1alpha1.GroupVersion.Version,
		Resource: accessRequestResourcePlural,
	}
	if ar.Request.Resource != accessRequestResource {
		err := fmt.Errorf("expect resource to be %s", accessRequestResource)
		klog.Error(err)
		return toV1AdmissionResponse(err)
	}

	accessRequest := &iamv1alpha1.AccessRequest{}
	raw := ar.Request.Object.Raw
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, accessRequest); err != nil {
		klog.Error(err)
		return toV1AdmissionResponse(err)
	}

	oldAccessRequest := &iamv1alpha1.AccessRequest{}
	if ar.Request.Operation == v1.Update || ar.Request.Operation == v1.Delete {
		oldRaw := ar.Request.OldObject.Raw
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(oldRaw, nil, oldAccessRequest); err != nil {
			klog.Error(err)
			return toV1AdmissionResponse(err)
		}
	}

	// Ensure createdBy attribute is immutable
	if ar.Request.Operation == v1.Update || ar.Request.Operation == v1.Delete {
		if accessRequest.Spec.Attributes == nil ||
			oldAccessRequest.Spec.Attributes == nil ||
			(accessRequest.Spec.Attributes.CreatedBy != oldAccessRequest.Spec.Attributes.CreatedBy) {
			err := errors.New("spec.attributes.createdBy is immutable")
			klog.Error(err)
			return toV1AdmissionResponse(err)
		}
	}

	// Validate approver
	if accessRequest.Spec.Approved {
		if accessRequest.Spec.Attributes == nil || accessRequest.Spec.Attributes.ApprovedBy == "" {
			err := fmt.Errorf("AccessRequest %s/%s has been approved but the approvedBy attribute is not set", accessRequest.Namespace, accessRequest.Name)
			klog.Error(err)
			return toV1AdmissionResponse(err)
		}

		sar, err := h.checkAccess(accessRequest.Spec.Attributes.ApprovedBy, accessRequest)
		if err != nil {
			klog.Error(err)
			return toV1AdmissionResponse(err)
		}

		if !sar.Status.Allowed || sar.Status.Denied {
			err := fmt.Errorf("%s is not allowed to approve AccessRequest %s/%s", ar.Request.UserInfo.Username, accessRequest.Namespace, accessRequest.Name)
			klog.Error(err)
			return toV1AdmissionResponse(err)
		}
	}

	return &v1.AdmissionResponse{Allowed: true}
}

func (h *serveValidateAccessRequestHandler) checkAccess(user string, accessRequest *iamv1alpha1.AccessRequest) (*authv1.SubjectAccessReview, error) {
	sar := &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			User: user,
			ResourceAttributes: &authv1.ResourceAttributes{
				Name:      accessRequest.Name,
				Namespace: accessRequest.Namespace,
				Verb:      approveVerb,
				Group:     iamv1alpha1.GroupVersion.Group,
				Version:   iamv1alpha1.GroupVersion.Version,
				Resource:  accessRequestResourcePlural,
			},
		},
	}

	sar, err := h.clientset.AuthorizationV1().SubjectAccessReviews().Create(context.TODO(), sar, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return sar, nil
}
