package types

// ComponentLogLevel type is used in the logging function to assert the log level for components
type ComponentLogLevel int

const (
	// ProjectReference is v1 verbose level
	ProjectReference ComponentLogLevel = 1
	// ProjectClaim is v2 verbose level
	ProjectClaim = 2
	// GCPClient is v3 verbose level
	GCPClient = 3
	//OperatorSDK is v4 verbose level
	OperatorSDK = 4
)
