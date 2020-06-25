package k8sclient

//go:generate mockgen -destination=../util/mocks/$GOPACKAGE/$GOFILE -package=$GOPACKAGE sigs.k8s.io/controller-runtime/pkg/client StatusWriter
