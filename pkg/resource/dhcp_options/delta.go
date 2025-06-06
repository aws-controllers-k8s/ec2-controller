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

package dhcp_options

import (
	"bytes"
	"reflect"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	acktags "github.com/aws-controllers-k8s/runtime/pkg/tags"
)

// Hack to avoid import errors during build...
var (
	_ = &bytes.Buffer{}
	_ = &reflect.Method{}
	_ = &acktags.Tags{}
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

	if len(a.ko.Spec.DHCPConfigurations) != len(b.ko.Spec.DHCPConfigurations) {
		delta.Add("Spec.DHCPConfigurations", a.ko.Spec.DHCPConfigurations, b.ko.Spec.DHCPConfigurations)
	} else if len(a.ko.Spec.DHCPConfigurations) > 0 {
		if !reflect.DeepEqual(a.ko.Spec.DHCPConfigurations, b.ko.Spec.DHCPConfigurations) {
			delta.Add("Spec.DHCPConfigurations", a.ko.Spec.DHCPConfigurations, b.ko.Spec.DHCPConfigurations)
		}
	}
	desiredACKTags, _ := convertToOrderedACKTags(a.ko.Spec.Tags)
	latestACKTags, _ := convertToOrderedACKTags(b.ko.Spec.Tags)
	if !ackcompare.MapStringStringEqual(desiredACKTags, latestACKTags) {
		delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
	}
	if len(a.ko.Spec.VPC) != len(b.ko.Spec.VPC) {
		delta.Add("Spec.VPC", a.ko.Spec.VPC, b.ko.Spec.VPC)
	} else if len(a.ko.Spec.VPC) > 0 {
		if !ackcompare.SliceStringPEqual(a.ko.Spec.VPC, b.ko.Spec.VPC) {
			delta.Add("Spec.VPC", a.ko.Spec.VPC, b.ko.Spec.VPC)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.VPCRefs, b.ko.Spec.VPCRefs) {
		delta.Add("Spec.VPCRefs", a.ko.Spec.VPCRefs, b.ko.Spec.VPCRefs)
	}

	return delta
}
