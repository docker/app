// +build !ignore_autogenerated

/*
Copyright 2018 The Kubernetes Authors.

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

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package v1beta2

import (
	core_v1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	reflect "reflect"
)

func init() {
	SchemeBuilder.Register(RegisterDeepCopies)
}

// RegisterDeepCopies adds deep-copy functions to the given scheme. Public
// to allow building arbitrary schemes.
//
// Deprecated: deepcopy registration will go away when static deepcopy is fully implemented.
func RegisterDeepCopies(scheme *runtime.Scheme) error {
	return scheme.AddGeneratedDeepCopyFuncs(
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*ControllerRevision).DeepCopyInto(out.(*ControllerRevision))
			return nil
		}, InType: reflect.TypeOf(&ControllerRevision{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*ControllerRevisionList).DeepCopyInto(out.(*ControllerRevisionList))
			return nil
		}, InType: reflect.TypeOf(&ControllerRevisionList{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*DaemonSet).DeepCopyInto(out.(*DaemonSet))
			return nil
		}, InType: reflect.TypeOf(&DaemonSet{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*DaemonSetList).DeepCopyInto(out.(*DaemonSetList))
			return nil
		}, InType: reflect.TypeOf(&DaemonSetList{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*DaemonSetSpec).DeepCopyInto(out.(*DaemonSetSpec))
			return nil
		}, InType: reflect.TypeOf(&DaemonSetSpec{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*DaemonSetStatus).DeepCopyInto(out.(*DaemonSetStatus))
			return nil
		}, InType: reflect.TypeOf(&DaemonSetStatus{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*DaemonSetUpdateStrategy).DeepCopyInto(out.(*DaemonSetUpdateStrategy))
			return nil
		}, InType: reflect.TypeOf(&DaemonSetUpdateStrategy{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*Deployment).DeepCopyInto(out.(*Deployment))
			return nil
		}, InType: reflect.TypeOf(&Deployment{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*DeploymentCondition).DeepCopyInto(out.(*DeploymentCondition))
			return nil
		}, InType: reflect.TypeOf(&DeploymentCondition{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*DeploymentList).DeepCopyInto(out.(*DeploymentList))
			return nil
		}, InType: reflect.TypeOf(&DeploymentList{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*DeploymentSpec).DeepCopyInto(out.(*DeploymentSpec))
			return nil
		}, InType: reflect.TypeOf(&DeploymentSpec{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*DeploymentStatus).DeepCopyInto(out.(*DeploymentStatus))
			return nil
		}, InType: reflect.TypeOf(&DeploymentStatus{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*DeploymentStrategy).DeepCopyInto(out.(*DeploymentStrategy))
			return nil
		}, InType: reflect.TypeOf(&DeploymentStrategy{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*ReplicaSet).DeepCopyInto(out.(*ReplicaSet))
			return nil
		}, InType: reflect.TypeOf(&ReplicaSet{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*ReplicaSetCondition).DeepCopyInto(out.(*ReplicaSetCondition))
			return nil
		}, InType: reflect.TypeOf(&ReplicaSetCondition{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*ReplicaSetList).DeepCopyInto(out.(*ReplicaSetList))
			return nil
		}, InType: reflect.TypeOf(&ReplicaSetList{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*ReplicaSetSpec).DeepCopyInto(out.(*ReplicaSetSpec))
			return nil
		}, InType: reflect.TypeOf(&ReplicaSetSpec{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*ReplicaSetStatus).DeepCopyInto(out.(*ReplicaSetStatus))
			return nil
		}, InType: reflect.TypeOf(&ReplicaSetStatus{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*RollingUpdateDaemonSet).DeepCopyInto(out.(*RollingUpdateDaemonSet))
			return nil
		}, InType: reflect.TypeOf(&RollingUpdateDaemonSet{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*RollingUpdateDeployment).DeepCopyInto(out.(*RollingUpdateDeployment))
			return nil
		}, InType: reflect.TypeOf(&RollingUpdateDeployment{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*RollingUpdateStatefulSetStrategy).DeepCopyInto(out.(*RollingUpdateStatefulSetStrategy))
			return nil
		}, InType: reflect.TypeOf(&RollingUpdateStatefulSetStrategy{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*Scale).DeepCopyInto(out.(*Scale))
			return nil
		}, InType: reflect.TypeOf(&Scale{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*ScaleSpec).DeepCopyInto(out.(*ScaleSpec))
			return nil
		}, InType: reflect.TypeOf(&ScaleSpec{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*ScaleStatus).DeepCopyInto(out.(*ScaleStatus))
			return nil
		}, InType: reflect.TypeOf(&ScaleStatus{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*StatefulSet).DeepCopyInto(out.(*StatefulSet))
			return nil
		}, InType: reflect.TypeOf(&StatefulSet{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*StatefulSetList).DeepCopyInto(out.(*StatefulSetList))
			return nil
		}, InType: reflect.TypeOf(&StatefulSetList{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*StatefulSetSpec).DeepCopyInto(out.(*StatefulSetSpec))
			return nil
		}, InType: reflect.TypeOf(&StatefulSetSpec{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*StatefulSetStatus).DeepCopyInto(out.(*StatefulSetStatus))
			return nil
		}, InType: reflect.TypeOf(&StatefulSetStatus{})},
		conversion.GeneratedDeepCopyFunc{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*StatefulSetUpdateStrategy).DeepCopyInto(out.(*StatefulSetUpdateStrategy))
			return nil
		}, InType: reflect.TypeOf(&StatefulSetUpdateStrategy{})},
	)
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerRevision) DeepCopyInto(out *ControllerRevision) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Data.DeepCopyInto(&out.Data)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerRevision.
func (in *ControllerRevision) DeepCopy() *ControllerRevision {
	if in == nil {
		return nil
	}
	out := new(ControllerRevision)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ControllerRevision) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerRevisionList) DeepCopyInto(out *ControllerRevisionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ControllerRevision, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerRevisionList.
func (in *ControllerRevisionList) DeepCopy() *ControllerRevisionList {
	if in == nil {
		return nil
	}
	out := new(ControllerRevisionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ControllerRevisionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DaemonSet) DeepCopyInto(out *DaemonSet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DaemonSet.
func (in *DaemonSet) DeepCopy() *DaemonSet {
	if in == nil {
		return nil
	}
	out := new(DaemonSet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DaemonSet) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DaemonSetList) DeepCopyInto(out *DaemonSetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DaemonSet, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DaemonSetList.
func (in *DaemonSetList) DeepCopy() *DaemonSetList {
	if in == nil {
		return nil
	}
	out := new(DaemonSetList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DaemonSetList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DaemonSetSpec) DeepCopyInto(out *DaemonSetSpec) {
	*out = *in
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		if *in == nil {
			*out = nil
		} else {
			*out = new(v1.LabelSelector)
			(*in).DeepCopyInto(*out)
		}
	}
	in.Template.DeepCopyInto(&out.Template)
	in.UpdateStrategy.DeepCopyInto(&out.UpdateStrategy)
	if in.RevisionHistoryLimit != nil {
		in, out := &in.RevisionHistoryLimit, &out.RevisionHistoryLimit
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DaemonSetSpec.
func (in *DaemonSetSpec) DeepCopy() *DaemonSetSpec {
	if in == nil {
		return nil
	}
	out := new(DaemonSetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DaemonSetStatus) DeepCopyInto(out *DaemonSetStatus) {
	*out = *in
	if in.CollisionCount != nil {
		in, out := &in.CollisionCount, &out.CollisionCount
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DaemonSetStatus.
func (in *DaemonSetStatus) DeepCopy() *DaemonSetStatus {
	if in == nil {
		return nil
	}
	out := new(DaemonSetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DaemonSetUpdateStrategy) DeepCopyInto(out *DaemonSetUpdateStrategy) {
	*out = *in
	if in.RollingUpdate != nil {
		in, out := &in.RollingUpdate, &out.RollingUpdate
		if *in == nil {
			*out = nil
		} else {
			*out = new(RollingUpdateDaemonSet)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DaemonSetUpdateStrategy.
func (in *DaemonSetUpdateStrategy) DeepCopy() *DaemonSetUpdateStrategy {
	if in == nil {
		return nil
	}
	out := new(DaemonSetUpdateStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Deployment) DeepCopyInto(out *Deployment) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Deployment.
func (in *Deployment) DeepCopy() *Deployment {
	if in == nil {
		return nil
	}
	out := new(Deployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Deployment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentCondition) DeepCopyInto(out *DeploymentCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentCondition.
func (in *DeploymentCondition) DeepCopy() *DeploymentCondition {
	if in == nil {
		return nil
	}
	out := new(DeploymentCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentList) DeepCopyInto(out *DeploymentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Deployment, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentList.
func (in *DeploymentList) DeepCopy() *DeploymentList {
	if in == nil {
		return nil
	}
	out := new(DeploymentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DeploymentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentSpec) DeepCopyInto(out *DeploymentSpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		if *in == nil {
			*out = nil
		} else {
			*out = new(v1.LabelSelector)
			(*in).DeepCopyInto(*out)
		}
	}
	in.Template.DeepCopyInto(&out.Template)
	in.Strategy.DeepCopyInto(&out.Strategy)
	if in.RevisionHistoryLimit != nil {
		in, out := &in.RevisionHistoryLimit, &out.RevisionHistoryLimit
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	if in.ProgressDeadlineSeconds != nil {
		in, out := &in.ProgressDeadlineSeconds, &out.ProgressDeadlineSeconds
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentSpec.
func (in *DeploymentSpec) DeepCopy() *DeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(DeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentStatus) DeepCopyInto(out *DeploymentStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]DeploymentCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.CollisionCount != nil {
		in, out := &in.CollisionCount, &out.CollisionCount
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentStatus.
func (in *DeploymentStatus) DeepCopy() *DeploymentStatus {
	if in == nil {
		return nil
	}
	out := new(DeploymentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentStrategy) DeepCopyInto(out *DeploymentStrategy) {
	*out = *in
	if in.RollingUpdate != nil {
		in, out := &in.RollingUpdate, &out.RollingUpdate
		if *in == nil {
			*out = nil
		} else {
			*out = new(RollingUpdateDeployment)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentStrategy.
func (in *DeploymentStrategy) DeepCopy() *DeploymentStrategy {
	if in == nil {
		return nil
	}
	out := new(DeploymentStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicaSet) DeepCopyInto(out *ReplicaSet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaSet.
func (in *ReplicaSet) DeepCopy() *ReplicaSet {
	if in == nil {
		return nil
	}
	out := new(ReplicaSet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ReplicaSet) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicaSetCondition) DeepCopyInto(out *ReplicaSetCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaSetCondition.
func (in *ReplicaSetCondition) DeepCopy() *ReplicaSetCondition {
	if in == nil {
		return nil
	}
	out := new(ReplicaSetCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicaSetList) DeepCopyInto(out *ReplicaSetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ReplicaSet, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaSetList.
func (in *ReplicaSetList) DeepCopy() *ReplicaSetList {
	if in == nil {
		return nil
	}
	out := new(ReplicaSetList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ReplicaSetList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicaSetSpec) DeepCopyInto(out *ReplicaSetSpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		if *in == nil {
			*out = nil
		} else {
			*out = new(v1.LabelSelector)
			(*in).DeepCopyInto(*out)
		}
	}
	in.Template.DeepCopyInto(&out.Template)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaSetSpec.
func (in *ReplicaSetSpec) DeepCopy() *ReplicaSetSpec {
	if in == nil {
		return nil
	}
	out := new(ReplicaSetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicaSetStatus) DeepCopyInto(out *ReplicaSetStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]ReplicaSetCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaSetStatus.
func (in *ReplicaSetStatus) DeepCopy() *ReplicaSetStatus {
	if in == nil {
		return nil
	}
	out := new(ReplicaSetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RollingUpdateDaemonSet) DeepCopyInto(out *RollingUpdateDaemonSet) {
	*out = *in
	if in.MaxUnavailable != nil {
		in, out := &in.MaxUnavailable, &out.MaxUnavailable
		if *in == nil {
			*out = nil
		} else {
			*out = new(intstr.IntOrString)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RollingUpdateDaemonSet.
func (in *RollingUpdateDaemonSet) DeepCopy() *RollingUpdateDaemonSet {
	if in == nil {
		return nil
	}
	out := new(RollingUpdateDaemonSet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RollingUpdateDeployment) DeepCopyInto(out *RollingUpdateDeployment) {
	*out = *in
	if in.MaxUnavailable != nil {
		in, out := &in.MaxUnavailable, &out.MaxUnavailable
		if *in == nil {
			*out = nil
		} else {
			*out = new(intstr.IntOrString)
			**out = **in
		}
	}
	if in.MaxSurge != nil {
		in, out := &in.MaxSurge, &out.MaxSurge
		if *in == nil {
			*out = nil
		} else {
			*out = new(intstr.IntOrString)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RollingUpdateDeployment.
func (in *RollingUpdateDeployment) DeepCopy() *RollingUpdateDeployment {
	if in == nil {
		return nil
	}
	out := new(RollingUpdateDeployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RollingUpdateStatefulSetStrategy) DeepCopyInto(out *RollingUpdateStatefulSetStrategy) {
	*out = *in
	if in.Partition != nil {
		in, out := &in.Partition, &out.Partition
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RollingUpdateStatefulSetStrategy.
func (in *RollingUpdateStatefulSetStrategy) DeepCopy() *RollingUpdateStatefulSetStrategy {
	if in == nil {
		return nil
	}
	out := new(RollingUpdateStatefulSetStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Scale) DeepCopyInto(out *Scale) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Scale.
func (in *Scale) DeepCopy() *Scale {
	if in == nil {
		return nil
	}
	out := new(Scale)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Scale) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScaleSpec) DeepCopyInto(out *ScaleSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScaleSpec.
func (in *ScaleSpec) DeepCopy() *ScaleSpec {
	if in == nil {
		return nil
	}
	out := new(ScaleSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScaleStatus) DeepCopyInto(out *ScaleStatus) {
	*out = *in
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScaleStatus.
func (in *ScaleStatus) DeepCopy() *ScaleStatus {
	if in == nil {
		return nil
	}
	out := new(ScaleStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StatefulSet) DeepCopyInto(out *StatefulSet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StatefulSet.
func (in *StatefulSet) DeepCopy() *StatefulSet {
	if in == nil {
		return nil
	}
	out := new(StatefulSet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *StatefulSet) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StatefulSetList) DeepCopyInto(out *StatefulSetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]StatefulSet, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StatefulSetList.
func (in *StatefulSetList) DeepCopy() *StatefulSetList {
	if in == nil {
		return nil
	}
	out := new(StatefulSetList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *StatefulSetList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StatefulSetSpec) DeepCopyInto(out *StatefulSetSpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		if *in == nil {
			*out = nil
		} else {
			*out = new(v1.LabelSelector)
			(*in).DeepCopyInto(*out)
		}
	}
	in.Template.DeepCopyInto(&out.Template)
	if in.VolumeClaimTemplates != nil {
		in, out := &in.VolumeClaimTemplates, &out.VolumeClaimTemplates
		*out = make([]core_v1.PersistentVolumeClaim, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.UpdateStrategy.DeepCopyInto(&out.UpdateStrategy)
	if in.RevisionHistoryLimit != nil {
		in, out := &in.RevisionHistoryLimit, &out.RevisionHistoryLimit
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StatefulSetSpec.
func (in *StatefulSetSpec) DeepCopy() *StatefulSetSpec {
	if in == nil {
		return nil
	}
	out := new(StatefulSetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StatefulSetStatus) DeepCopyInto(out *StatefulSetStatus) {
	*out = *in
	if in.CollisionCount != nil {
		in, out := &in.CollisionCount, &out.CollisionCount
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StatefulSetStatus.
func (in *StatefulSetStatus) DeepCopy() *StatefulSetStatus {
	if in == nil {
		return nil
	}
	out := new(StatefulSetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StatefulSetUpdateStrategy) DeepCopyInto(out *StatefulSetUpdateStrategy) {
	*out = *in
	if in.RollingUpdate != nil {
		in, out := &in.RollingUpdate, &out.RollingUpdate
		if *in == nil {
			*out = nil
		} else {
			*out = new(RollingUpdateStatefulSetStrategy)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StatefulSetUpdateStrategy.
func (in *StatefulSetUpdateStrategy) DeepCopy() *StatefulSetUpdateStrategy {
	if in == nil {
		return nil
	}
	out := new(StatefulSetUpdateStrategy)
	in.DeepCopyInto(out)
	return out
}
