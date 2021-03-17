# Access Request Controller

```sh
kubebuilder init --domain dippynark.co.uk
kubebuilder create api --group iam --version v1alpha1 --kind AccessRequest
kubebuilder create webhook --group iam --version v1alpha1 --kind AccessRequest --defaulting --programmatic-validation
```

## API

```yaml
apiVersion: iam.dippynark.co.uk/v1alpha1
kind: AccessRequest
metadata:
  name: developer@org.com:pod-reader
spec:
  subjects:
  - kind: User
    name: developer@org.com
    apiGroup: rbac.authorization.k8s.io
  roleRef:
    kind: Role
    name: pod-reader
    apiGroup: rbac.authorization.k8s.io
status:
  createdBy: developer@org.com
  approvedBy: admin@org.com
  approvalTime: "2021-02-19T10:40:17Z"
  completionTime: "2021-02-19T10:40:17Z"
  conditions:
  - lastProbeTime: "2021-02-19T10:40:17Z"
    lastTransitionTime: "2021-02-19T10:40:17Z"
    status: "True"
    type: Approved
  - lastProbeTime: "2021-02-19T10:40:17Z"
    lastTransitionTime: "2021-02-19T10:40:17Z"
    status: "True"
    type: Complete
```

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test
subjects:
- kind: User
  name: developer
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```
