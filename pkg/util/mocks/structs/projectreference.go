package structs

import (
	//"errors"
	//"github.com/stretchr/testify/assert"

	"fmt"

	api "github.com/openshift/gcp-project-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
				ServiceAccountName: "",
			},
		},
	}
}

func (t *testProjectReferenceBuilder) WithNamespacedName(namespacedName types.NamespacedName) *testProjectReferenceBuilder {
	t.p.Name = namespacedName.Name
	t.p.Namespace = namespacedName.Namespace
	return t
}

type ProjectReferenceMatcher struct {
	ActualProjectReference api.ProjectReference
	FailReason             string
}

func NewProjectReferenceMatcher() *ProjectReferenceMatcher {
	return &ProjectReferenceMatcher{}
}

func (m *ProjectReferenceMatcher) Matches(x interface{}) bool {
	ref, isCorrectType := x.(*api.ProjectReference)
	if !isCorrectType {
		m.FailReason = fmt.Sprintf("Unexpected type passed: want '%T', got '%T'", api.ProjectReference{}, x)
		return false
	}
	m.ActualProjectReference = *ref.DeepCopy()
	return true
}

func (m *ProjectReferenceMatcher) String() string {
	return "Fail reason: " + m.FailReason
}
