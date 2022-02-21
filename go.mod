module github.com/openshift/gcp-project-operator

go 1.14

require (
	github.com/cenkalti/backoff/v4 v4.1.2
	github.com/go-logr/logr v1.2.2
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.3.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.18.1
	github.com/openshift/cluster-api v0.0.0-20191129101638-b09907ac6668
	github.com/openshift/gcp-project-operator/pkg/apis v0.0.0-00010101000000-000000000000
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	google.golang.org/api v0.69.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.23.0
	k8s.io/apimachinery v0.23.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.23.0
	k8s.io/gengo v0.0.0-20211129171323-c02415ce4185
	k8s.io/kube-openapi v0.0.0-20220124234850-424119656bbf
	sigs.k8s.io/controller-runtime v0.11.1
)

// Get the APIs from the sub-module in this same repository:
replace github.com/openshift/gcp-project-operator/pkg/apis => ./pkg/apis

// Required to avoid the incorrect v1.* tags of this project and force
// selection of the v.0.* tags that match the Kubernetes version:
replace k8s.io/client-go => k8s.io/client-go v0.23.0
