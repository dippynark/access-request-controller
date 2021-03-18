# Access Request Controller

```sh
kubebuilder init --domain dippynark.co.uk
kubebuilder create api --group iam --version v1alpha1 --kind AccessRequest
```

## Example

```sh
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

kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: manager
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
  name: manager
subjects:
- kind: User
  name: manager
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: manager
  apiGroup: rbac.authorization.k8s.io
---
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

kubectl patch accessrequests.iam.dippynark.co.uk developer --type=merge -p '{"spec":{"approved":true}}' --as manager
```
