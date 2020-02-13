// +build !ignore_autogenerated

// Code generated by openapi-gen. DO NOT EDIT.

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectClaim":           schema_pkg_apis_gcp_v1alpha1_ProjectClaim(ref),
		"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectClaimSpec":       schema_pkg_apis_gcp_v1alpha1_ProjectClaimSpec(ref),
		"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectClaimStatus":     schema_pkg_apis_gcp_v1alpha1_ProjectClaimStatus(ref),
		"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectReference":       schema_pkg_apis_gcp_v1alpha1_ProjectReference(ref),
		"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectReferenceSpec":   schema_pkg_apis_gcp_v1alpha1_ProjectReferenceSpec(ref),
		"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectReferenceStatus": schema_pkg_apis_gcp_v1alpha1_ProjectReferenceStatus(ref),
	}
}

func schema_pkg_apis_gcp_v1alpha1_ProjectClaim(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ProjectClaim is the Schema for the projectclaims API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectClaimSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectClaimStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectClaimSpec", "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectClaimStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_gcp_v1alpha1_ProjectClaimSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ProjectClaimSpec defines the desired state of ProjectClaim",
				Properties: map[string]spec.Schema{
					"legalEntity": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.LegalEntity"),
						},
					},
					"gcpCredentialSecret": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.NamespacedName"),
						},
					},
					"region": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"gcpProjectID": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"projectReferenceCRLink": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.NamespacedName"),
						},
					},
				},
				Required: []string{"legalEntity", "gcpCredentialSecret", "region"},
			},
		},
		Dependencies: []string{
			"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.LegalEntity", "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.NamespacedName"},
	}
}

func schema_pkg_apis_gcp_v1alpha1_ProjectClaimStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ProjectClaimStatus defines the observed state of ProjectClaim",
				Properties: map[string]spec.Schema{
					"conditions": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectClaimCondition"),
									},
								},
							},
						},
					},
					"state": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
				Required: []string{"conditions", "state"},
			},
		},
		Dependencies: []string{
			"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectClaimCondition"},
	}
}

func schema_pkg_apis_gcp_v1alpha1_ProjectReference(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ProjectReference is the Schema for the ProjectReferences API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectReferenceSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectReferenceStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectReferenceSpec", "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectReferenceStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_gcp_v1alpha1_ProjectReferenceSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ProjectReferenceSpec defines the desired state of Project",
				Properties: map[string]spec.Schema{
					"gcpProjectID": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"projectClaimCRLink": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.NamespacedName"),
						},
					},
					"legalEntity": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.LegalEntity"),
						},
					},
				},
				Required: []string{"projectClaimCRLink", "legalEntity"},
			},
		},
		Dependencies: []string{
			"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.LegalEntity", "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.NamespacedName"},
	}
}

func schema_pkg_apis_gcp_v1alpha1_ProjectReferenceStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ProjectReferenceStatus defines the observed state of Project",
				Properties: map[string]spec.Schema{
					"conditions": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectReferenceCondition"),
									},
								},
							},
						},
					},
					"state": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1.ProjectReferenceCondition"},
	}
}
