// +build !ignore_autogenerated

/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProjectClaim) DeepCopyInto(out *ProjectClaim) {
*out = *in
out.TypeMeta = in.TypeMeta
in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
in.Spec.DeepCopyInto(&out.Spec)
in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProjectClaim.
func (in *ProjectClaim) DeepCopy() *ProjectClaim {
	if in == nil { return nil }
	out := new(ProjectClaim)
	in.DeepCopyInto(out)
	return out
}


// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ProjectClaim) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProjectClaimList) DeepCopyInto(out *ProjectClaimList) {
*out = *in
out.TypeMeta = in.TypeMeta
in.ListMeta.DeepCopyInto(&out.ListMeta)
if in.Items != nil {
in, out := &in.Items, &out.Items
*out = make([]ProjectClaim, len(*in))
for i := range *in {
(*in)[i].DeepCopyInto(&(*out)[i])
}
}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProjectClaimList.
func (in *ProjectClaimList) DeepCopy() *ProjectClaimList {
	if in == nil { return nil }
	out := new(ProjectClaimList)
	in.DeepCopyInto(out)
	return out
}


// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ProjectClaimList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProjectClaimSpec) DeepCopyInto(out *ProjectClaimSpec) {
*out = *in
if in.AvailabilityZones != nil {
in, out := &in.AvailabilityZones, &out.AvailabilityZones
*out = make([]string, len(*in))
copy(*out, *in)
}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProjectClaimSpec.
func (in *ProjectClaimSpec) DeepCopy() *ProjectClaimSpec {
	if in == nil { return nil }
	out := new(ProjectClaimSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProjectClaimStatus) DeepCopyInto(out *ProjectClaimStatus) {
*out = *in
if in.Conditions != nil {
in, out := &in.Conditions, &out.Conditions
*out = make([]invalid type, len(*in))
copy(*out, *in)
}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProjectClaimStatus.
func (in *ProjectClaimStatus) DeepCopy() *ProjectClaimStatus {
	if in == nil { return nil }
	out := new(ProjectClaimStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProjectReference) DeepCopyInto(out *ProjectReference) {
*out = *in
out.TypeMeta = in.TypeMeta
in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
out.Spec = in.Spec
in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProjectReference.
func (in *ProjectReference) DeepCopy() *ProjectReference {
	if in == nil { return nil }
	out := new(ProjectReference)
	in.DeepCopyInto(out)
	return out
}


// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ProjectReference) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProjectReferenceList) DeepCopyInto(out *ProjectReferenceList) {
*out = *in
out.TypeMeta = in.TypeMeta
in.ListMeta.DeepCopyInto(&out.ListMeta)
if in.Items != nil {
in, out := &in.Items, &out.Items
*out = make([]ProjectReference, len(*in))
for i := range *in {
(*in)[i].DeepCopyInto(&(*out)[i])
}
}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProjectReferenceList.
func (in *ProjectReferenceList) DeepCopy() *ProjectReferenceList {
	if in == nil { return nil }
	out := new(ProjectReferenceList)
	in.DeepCopyInto(out)
	return out
}


// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ProjectReferenceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProjectReferenceSpec) DeepCopyInto(out *ProjectReferenceSpec) {
*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProjectReferenceSpec.
func (in *ProjectReferenceSpec) DeepCopy() *ProjectReferenceSpec {
	if in == nil { return nil }
	out := new(ProjectReferenceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProjectReferenceStatus) DeepCopyInto(out *ProjectReferenceStatus) {
*out = *in
if in.Conditions != nil {
in, out := &in.Conditions, &out.Conditions
*out = make([]invalid type, len(*in))
copy(*out, *in)
}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProjectReferenceStatus.
func (in *ProjectReferenceStatus) DeepCopy() *ProjectReferenceStatus {
	if in == nil { return nil }
	out := new(ProjectReferenceStatus)
	in.DeepCopyInto(out)
	return out
}

