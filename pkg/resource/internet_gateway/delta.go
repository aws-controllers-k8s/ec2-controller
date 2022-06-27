// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Code generated by ack-generate. DO NOT EDIT.

package internet_gateway

import (
	"bytes"
	"reflect"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
)

// Hack to avoid import errors during build...
var (
	_ = &bytes.Buffer{}
	_ = &reflect.Method{}
)

// newResourceDelta returns a new `ackcompare.Delta` used to compare two
// resources
func newResourceDelta(
	a *resource,
	b *resource,
) *ackcompare.Delta {
	delta := ackcompare.NewDelta()
	if (a == nil && b != nil) ||
		(a != nil && b == nil) {
		delta.Add("", a, b)
		return delta
	}

	if !reflect.DeepEqual(a.ko.Spec.Tags, b.ko.Spec.Tags) {
		delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
	}
	if ackcompare.HasNilDifference(a.ko.Spec.VPC, b.ko.Spec.VPC) {
		delta.Add("Spec.VPC", a.ko.Spec.VPC, b.ko.Spec.VPC)
	} else if a.ko.Spec.VPC != nil && b.ko.Spec.VPC != nil {
		if *a.ko.Spec.VPC != *b.ko.Spec.VPC {
			delta.Add("Spec.VPC", a.ko.Spec.VPC, b.ko.Spec.VPC)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.VPCRef, b.ko.Spec.VPCRef) {
		delta.Add("Spec.VPCRef", a.ko.Spec.VPCRef, b.ko.Spec.VPCRef)
	}

	return delta
}
