module github.com/dippynark/access-request-controller

go 1.15

require (
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.15.2
	github.com/onsi/gomega v1.11.0
	github.com/spf13/cobra v1.1.3
	k8s.io/api v0.21.0-beta.1
	k8s.io/apimachinery v0.21.0-beta.1
	k8s.io/client-go v0.21.0-beta.1
	k8s.io/klog/v2 v2.8.0
	sigs.k8s.io/cluster-api v0.3.11-0.20210316165436-c08044671121
	sigs.k8s.io/controller-runtime v0.8.2-0.20210314174504-df2c43d8896d
)
