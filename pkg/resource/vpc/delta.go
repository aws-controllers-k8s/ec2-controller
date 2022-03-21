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

package vpc

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

	if ackcompare.HasNilDifference(a.ko.Spec.AmazonProvidedIPv6CIDRBlock, b.ko.Spec.AmazonProvidedIPv6CIDRBlock) {
		delta.Add("Spec.AmazonProvidedIPv6CIDRBlock", a.ko.Spec.AmazonProvidedIPv6CIDRBlock, b.ko.Spec.AmazonProvidedIPv6CIDRBlock)
	} else if a.ko.Spec.AmazonProvidedIPv6CIDRBlock != nil && b.ko.Spec.AmazonProvidedIPv6CIDRBlock != nil {
		if *a.ko.Spec.AmazonProvidedIPv6CIDRBlock != *b.ko.Spec.AmazonProvidedIPv6CIDRBlock {
			delta.Add("Spec.AmazonProvidedIPv6CIDRBlock", a.ko.Spec.AmazonProvidedIPv6CIDRBlock, b.ko.Spec.AmazonProvidedIPv6CIDRBlock)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.CIDRBlock, b.ko.Spec.CIDRBlock) {
		delta.Add("Spec.CIDRBlock", a.ko.Spec.CIDRBlock, b.ko.Spec.CIDRBlock)
	} else if a.ko.Spec.CIDRBlock != nil && b.ko.Spec.CIDRBlock != nil {
		if *a.ko.Spec.CIDRBlock != *b.ko.Spec.CIDRBlock {
			delta.Add("Spec.CIDRBlock", a.ko.Spec.CIDRBlock, b.ko.Spec.CIDRBlock)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.EnableDNSHostnames, b.ko.Spec.EnableDNSHostnames) {
		delta.Add("Spec.EnableDNSHostnames", a.ko.Spec.EnableDNSHostnames, b.ko.Spec.EnableDNSHostnames)
	} else if a.ko.Spec.EnableDNSHostnames != nil && b.ko.Spec.EnableDNSHostnames != nil {
		if *a.ko.Spec.EnableDNSHostnames != *b.ko.Spec.EnableDNSHostnames {
			delta.Add("Spec.EnableDNSHostnames", a.ko.Spec.EnableDNSHostnames, b.ko.Spec.EnableDNSHostnames)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.EnableDNSSupport, b.ko.Spec.EnableDNSSupport) {
		delta.Add("Spec.EnableDNSSupport", a.ko.Spec.EnableDNSSupport, b.ko.Spec.EnableDNSSupport)
	} else if a.ko.Spec.EnableDNSSupport != nil && b.ko.Spec.EnableDNSSupport != nil {
		if *a.ko.Spec.EnableDNSSupport != *b.ko.Spec.EnableDNSSupport {
			delta.Add("Spec.EnableDNSSupport", a.ko.Spec.EnableDNSSupport, b.ko.Spec.EnableDNSSupport)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.InstanceTenancy, b.ko.Spec.InstanceTenancy) {
		delta.Add("Spec.InstanceTenancy", a.ko.Spec.InstanceTenancy, b.ko.Spec.InstanceTenancy)
	} else if a.ko.Spec.InstanceTenancy != nil && b.ko.Spec.InstanceTenancy != nil {
		if *a.ko.Spec.InstanceTenancy != *b.ko.Spec.InstanceTenancy {
			delta.Add("Spec.InstanceTenancy", a.ko.Spec.InstanceTenancy, b.ko.Spec.InstanceTenancy)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.IPv6CIDRBlock, b.ko.Spec.IPv6CIDRBlock) {
		delta.Add("Spec.IPv6CIDRBlock", a.ko.Spec.IPv6CIDRBlock, b.ko.Spec.IPv6CIDRBlock)
	} else if a.ko.Spec.IPv6CIDRBlock != nil && b.ko.Spec.IPv6CIDRBlock != nil {
		if *a.ko.Spec.IPv6CIDRBlock != *b.ko.Spec.IPv6CIDRBlock {
			delta.Add("Spec.IPv6CIDRBlock", a.ko.Spec.IPv6CIDRBlock, b.ko.Spec.IPv6CIDRBlock)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.IPv6CIDRBlockNetworkBorderGroup, b.ko.Spec.IPv6CIDRBlockNetworkBorderGroup) {
		delta.Add("Spec.IPv6CIDRBlockNetworkBorderGroup", a.ko.Spec.IPv6CIDRBlockNetworkBorderGroup, b.ko.Spec.IPv6CIDRBlockNetworkBorderGroup)
	} else if a.ko.Spec.IPv6CIDRBlockNetworkBorderGroup != nil && b.ko.Spec.IPv6CIDRBlockNetworkBorderGroup != nil {
		if *a.ko.Spec.IPv6CIDRBlockNetworkBorderGroup != *b.ko.Spec.IPv6CIDRBlockNetworkBorderGroup {
			delta.Add("Spec.IPv6CIDRBlockNetworkBorderGroup", a.ko.Spec.IPv6CIDRBlockNetworkBorderGroup, b.ko.Spec.IPv6CIDRBlockNetworkBorderGroup)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.IPv6Pool, b.ko.Spec.IPv6Pool) {
		delta.Add("Spec.IPv6Pool", a.ko.Spec.IPv6Pool, b.ko.Spec.IPv6Pool)
	} else if a.ko.Spec.IPv6Pool != nil && b.ko.Spec.IPv6Pool != nil {
		if *a.ko.Spec.IPv6Pool != *b.ko.Spec.IPv6Pool {
			delta.Add("Spec.IPv6Pool", a.ko.Spec.IPv6Pool, b.ko.Spec.IPv6Pool)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.TagSpecifications, b.ko.Spec.TagSpecifications) {
		delta.Add("Spec.TagSpecifications", a.ko.Spec.TagSpecifications, b.ko.Spec.TagSpecifications)
	}

	return delta
}
