package structs

import (
	//"errors"
	//"github.com/stretchr/testify/assert"

	"fmt"

	api "github.com/openshift/gcp-project-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestProjectClaimName string = "fakeProjectClaim"
	TestNamespace        string = "fakeNamespace"
)

type testProjectClaimBuilder struct {
	p api.ProjectClaim
}

func (t *testProjectClaimBuilder) GetProjectClaim() *api.ProjectClaim {
	return &t.p
}

func NewProjectClaimBuilder() *testProjectClaimBuilder {
	return &testProjectClaimBuilder{
		p: api.ProjectClaim{

			ObjectMeta: metav1.ObjectMeta{
				Name:      TestProjectClaimName,
				Namespace: TestNamespace,
			},
			Spec: api.ProjectClaimSpec{
				LegalEntity: api.LegalEntity{
					Name: "fakeLegalEntityName",
					ID:   "fakeLegalEntityID",
				},
				Region: "us-east1",
			},
			Status: api.ProjectClaimStatus{
				State: api.ClaimStatusPendingProject,
			},
		},
	}
}

func (t *testProjectClaimBuilder) Initialized() *testProjectClaimBuilder {
	t.p.Status = api.ProjectClaimStatus{
		Conditions: []api.Condition{},
	}
	return t
}

func (t *testProjectClaimBuilder) WithFinalizer(finalizers []string) *testProjectClaimBuilder {
	t.p.Finalizers = finalizers
	return t
}

type ProjectClaimMatcher struct {
	ActualProjectClaim *api.ProjectClaim
	FailReason         string
}

func NewProjectClaimMatcher() *ProjectClaimMatcher {
	return &ProjectClaimMatcher{&api.ProjectClaim{}, ""}
}

func (m *ProjectClaimMatcher) Matches(x interface{}) bool {
	ref, isCorrectType := x.(*api.ProjectClaim)
	if !isCorrectType {
		m.FailReason = fmt.Sprintf("Unexpected type passed: want '%T', got '%T'", api.ProjectClaim{}, x)
		return false
	}
	m.ActualProjectClaim = ref
	return true
}

func (m *ProjectClaimMatcher) String() string {
	return "Fail reason: " + m.FailReason
}
