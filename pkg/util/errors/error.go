package errors

import "errors"

func New(err string) error {
	return errors.New(err)
}

// ErrRegionNotSupported indicates the region is not supported by OSD on GCP.
var ErrRegionNotSupported = errors.New("RegionNotSupported")

// ErrNotGCPCluster indicates that the cluster is not a gcp cluster
var ErrNotGCPCluster = errors.New("NotGCPCluster")

// ErrNotManagedCluster indicates this is not an OSD managed cluster
var ErrNotManagedCluster = errors.New("NotManagedCluster")

// ErrClusterInstalled indicates the cluster is already installed
var ErrClusterInstalled = errors.New("ClusterInstalled")

// ErrMissingProjectID indicates that the cluster deployment is missing the field ProjectID
var ErrMissingProjectID = errors.New("MissingProjectID")

// ErrMissingRegion indicates that the cluster deployment is missing the field Region
var ErrMissingRegion = errors.New("MissingRegion")
