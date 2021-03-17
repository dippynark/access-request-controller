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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AccessRequestSpec defines the desired state of AccessRequest
type AccessRequestSpec struct {
	// Approved specifies whether the accessrequest has been approved
	Approved bool `json:"approved,omitempty"`
	// Subjects holds references to the objects the role applies to.
	// +optional
	Subjects []rbacv1.Subject `json:"subjects,omitempty"`
	// RoleRef can reference a Role in the current namespace or a ClusterRole in the global namespace.
	RoleRef rbacv1.RoleRef `json:"roleRef"`
}

// AccessRequestStatus defines the observed state of AccessRequest
type AccessRequestStatus struct {

	// Signifies who created the accessrequest
	CreatedBy string `json:"createdBy,omitempty"`

	// Signifies who approved the accessrequest
	// +optional
	ApprovedBy string `json:"approvedBy,omitempty"`

	// Represents time when the accessrequest was approved.
	// +optional
	ApprovalTime *metav1.Time `json:"approvalTime,omitempty"`

	// Represents time when the accessrequest was completed. The completion time is only set when the
	// accessrequest is rejected or is approved and the corresponding binding created.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// The latest available observations of an object's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []AccessRequestCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true

// AccessRequest is the Schema for the accessrequests API
type AccessRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessRequestSpec   `json:"spec,omitempty"`
	Status AccessRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AccessRequestList contains a list of AccessRequest
type AccessRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AccessRequest `json:"items"`
}

type AccessRequestConditionType string

// These are valid conditions of an accessrequest.
const (
	// AccessRequestApproved means the accessrequest has been approved.
	AccessRequestApproved AccessRequestConditionType = "Approved"
	// AccessRequestComplete means the accessrequest has completed its lifecycle.
	AccessRequestComplete AccessRequestConditionType = "Complete"
)

type AccessRequestCondition struct {
	// Type of accessrequest condition, Approved or Complete.
	Type AccessRequestConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// Last time the condition was checked.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

func init() {
	SchemeBuilder.Register(&AccessRequest{}, &AccessRequestList{})
}
