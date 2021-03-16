package controllers

import (
	"strings"

	iamv1alpha1 "github.com/dippynark/access-request-controller/api/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func getRoleBindingName(accessRequest *iamv1alpha1.AccessRequest, subject rbacv1.Subject) string {
	roleBindingName := strings.ToLower(subject.Kind)

	if subject.Namespace != "" {
		roleBindingName += ":" + subject.Namespace
	}
	roleBindingName += ":" + subject.Name

	roleRef := accessRequest.Spec.RoleRef
	roleBindingName += ":" + strings.ToLower(roleRef.Kind) + ":" + roleRef.Name

	return roleBindingName
}
