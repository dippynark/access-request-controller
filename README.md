# Access Request Controller

```sh
kubebuilder init --domain dippynark.co.uk
kubebuilder create api --group iam --version v1alpha1 --kind AccessRequest
```

## Installation

```sh
make deploy
```

## Example

```sh
# Allow developer to create AccessRequests
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: access-request-creator
rules:
- apiGroups:
  - iam.dippynark.co.uk
  resources:
  - accessrequests
  verbs:
  - create
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: access-request-creator:developer
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: access-request-creator
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: developer
EOF

# Create AccessRequest as developer
kubectl create --as developer -f - <<EOF
apiVersion: iam.dippynark.co.uk/v1alpha1
kind: AccessRequest
metadata:
  name: developer
spec:
  subjects:
  - apiGroup: rbac.authorization.k8s.io
    kind: User
    name: developer
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: Role
    name: developer
EOF

# Allow access-request-controller to create RoleBinding for developer Role
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: developer-role-binder
rules:
- apiGroups:
  - rbac.authorization.k8s.io
  resources:
  - roles
  verbs:
  - bind
  resourceNames:
  - developer
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: developer-role-binder:access-request-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: developer-role-binder
subjects:
- kind: ServiceAccount
  name: default
  namespace: access-request-controller-system
EOF

# Allow manager to approve AccessRequests
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: access-request-approver
rules:
- apiGroups:
  - iam.dippynark.co.uk
  resources:
  - accessrequests
  verbs:
  - get
  - patch
  - approve
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: access-request-approver:manager
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: access-request-approver
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: manager
EOF

# Approve AccessRequest as manager
kubectl patch accessrequests.iam.dippynark.co.uk developer --as manager --type=merge -p '{"spec":{"approved":true}}'

# Cleanup
kubectl delete rolebinding access-request-approver:manager access-request-creator:developer developer-role-binder:access-request-controller
kubectl delete role access-request-approver access-request-creator developer-role-binder
kubectl delete accessrequest developer
```
