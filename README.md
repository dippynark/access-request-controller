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
# Apply RBAC to allow access-request-controller to bypass privilege escalation when creating RoleBindings
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: cluster-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: default
  namespace: access-request-controller-system
EOF

# Apply RBAC to allow developer to create AccessRequests
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
  - get
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
kubectl apply --as developer -f - <<EOF
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

# Apply RBAC to allow manager to approve AccessRequests
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
```
