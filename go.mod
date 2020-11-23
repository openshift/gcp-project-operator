module github.com/openshift/gcp-project-operator

require (
	github.com/emicklei/go-restful v2.11.2+incompatible // indirect
	github.com/fatih/color v1.10.0 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.4
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/jpoles1/gopherbadger v2.4.0+incompatible // indirect
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/openshift/cluster-api v0.0.0-20191129101638-b09907ac6668
	github.com/operator-framework/operator-sdk v0.17.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	go.opencensus.io v0.22.3 // indirect
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20200523222454-059865788121 // indirect
	golang.org/x/tools v0.0.0-20201008184944-d01b322e6f06 // indirect
	google.golang.org/api v0.25.0
	google.golang.org/genproto v0.0.0-20200527145253-8367513e4ece // indirect
	google.golang.org/grpc v1.29.1 // indirect
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.17.4
	k8s.io/gengo v0.0.0-20191010091904-7fa3014cb28f
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

go 1.13
