package structs

import (
	//"errors"
	//"github.com/stretchr/testify/assert"

	"fmt"

	api "github.com/openshift/gcp-project-operator/pkg/apis/gcp/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testProjectReferenceBuilder struct {
	p api.ProjectReference
}

func (t *testProjectReferenceBuilder) GetProjectReference() *api.ProjectReference {
	return &t.p
}

func NewProjectReferenceBuilder() *testProjectReferenceBuilder {
	return &testProjectReferenceBuilder{
		p: api.ProjectReference{

			ObjectMeta: metav1.ObjectMeta{
				Name:      "fakeProjectReference",
				Namespace: "fakeNamespace",
			},
			Spec: api.ProjectReferenceSpec{
				GCPProjectID: "",
				ProjectClaimCRLink: api.NamespacedName{
					Namespace: "fakeNamespace",
					Name:      "fakeName",
				},

				LegalEntity: api.LegalEntity{
					Name: "fakeLegalEntityName",
					ID:   "fakeLegalEntityID",
				},
			},
		},
	}
}

type projectIdMatcher struct {
	ActualProjectId string
	FailReason      string
}

func NewProjectIdMatcher() *projectIdMatcher {
	return &projectIdMatcher{}
}

func (m *projectIdMatcher) Matches(x interface{}) bool {
	ref, isCorrectType := x.(*api.ProjectReference)
	if !isCorrectType {
		m.FailReason = fmt.Sprintf("Unexpected type passed: want '%T', got '%T'", api.ProjectReference{}, x)
		return false
	}
	m.ActualProjectId = ref.Spec.GCPProjectID
	return true
}

func (m *projectIdMatcher) String() string {
	return "Fail reason: " + m.FailReason
}
