module github.com/openshift/gcp-project-operator/pkg/apis

go 1.14

require (
	github.com/emicklei/go-restful v2.11.2+incompatible // indirect
	github.com/go-openapi/spec v0.19.4
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/onsi/gomega v1.9.0 // indirect
	github.com/stretchr/testify v1.6.1 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sys v0.0.0-20210119212857-b64e53b001e4 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/protobuf v1.24.0 // indirect
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)

// Pin google api to v0.11.0
replace google.golang.org/api => google.golang.org/api v0.11.0

// Pin hive dep
replace github.com/openshift/cluster-network-operator => github.com/openshift/cluster-network-operator v0.0.0-20190207145423-c226dcab667e
