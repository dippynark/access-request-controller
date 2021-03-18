package main

import (
	"fmt"
	"strings"
	"time"

	iamv1alpha1 "github.com/dippynark/access-request-controller/api/v1alpha1"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func mutateAccessRequest(ar v1.AdmissionReview) *v1.AdmissionResponse {
	klog.V(2).Infof("mutating %s", accessRequestResourceSingular)

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

	patches := []string{}

	// Ensure attributes object
	if accessRequest.Spec.Attributes == nil {
		patches = append(patches, `{"op":"add","path":"/spec/attributes","value":{}}`)
	}

	// Always patch createdBy attribute on create
	if ar.Request.Operation == v1.Create {
		patches = append(patches, fmt.Sprintf(`{"op":"add","path":"/spec/attributes/createdBy","value":"%s"}`, ar.Request.UserInfo.Username))
	}

	// Patch approval attributes
	if accessRequest.Spec.Approved {
		patches = append(patches, fmt.Sprintf(`{"op":"add","path":"/spec/attributes/approvedBy","value":"%s"}`, ar.Request.UserInfo.Username))
		patches = append(patches, fmt.Sprintf(`{"op":"add","path":"/spec/attributes/approvalTime","value":"%s"}`, metav1.Now().Format(time.RFC3339)))
	}

	admissionResponse := &v1.AdmissionResponse{Allowed: true}
	if len(patches) > 0 {
		admissionResponse.Patch = []byte(fmt.Sprintf(`[%s]`, strings.Join(patches, ",")))
		pt := v1.PatchTypeJSONPatch
		admissionResponse.PatchType = &pt
	}

	return admissionResponse
}
