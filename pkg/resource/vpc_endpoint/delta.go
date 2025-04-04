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

package vpc_endpoint

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

	if ackcompare.HasNilDifference(a.ko.Spec.DNSOptions, b.ko.Spec.DNSOptions) {
		delta.Add("Spec.DNSOptions", a.ko.Spec.DNSOptions, b.ko.Spec.DNSOptions)
	} else if a.ko.Spec.DNSOptions != nil && b.ko.Spec.DNSOptions != nil {
		if ackcompare.HasNilDifference(a.ko.Spec.DNSOptions.DNSRecordIPType, b.ko.Spec.DNSOptions.DNSRecordIPType) {
			delta.Add("Spec.DNSOptions.DNSRecordIPType", a.ko.Spec.DNSOptions.DNSRecordIPType, b.ko.Spec.DNSOptions.DNSRecordIPType)
		} else if a.ko.Spec.DNSOptions.DNSRecordIPType != nil && b.ko.Spec.DNSOptions.DNSRecordIPType != nil {
			if *a.ko.Spec.DNSOptions.DNSRecordIPType != *b.ko.Spec.DNSOptions.DNSRecordIPType {
				delta.Add("Spec.DNSOptions.DNSRecordIPType", a.ko.Spec.DNSOptions.DNSRecordIPType, b.ko.Spec.DNSOptions.DNSRecordIPType)
			}
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.IPAddressType, b.ko.Spec.IPAddressType) {
		delta.Add("Spec.IPAddressType", a.ko.Spec.IPAddressType, b.ko.Spec.IPAddressType)
	} else if a.ko.Spec.IPAddressType != nil && b.ko.Spec.IPAddressType != nil {
		if *a.ko.Spec.IPAddressType != *b.ko.Spec.IPAddressType {
			delta.Add("Spec.IPAddressType", a.ko.Spec.IPAddressType, b.ko.Spec.IPAddressType)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.PolicyDocument, b.ko.Spec.PolicyDocument) {
		delta.Add("Spec.PolicyDocument", a.ko.Spec.PolicyDocument, b.ko.Spec.PolicyDocument)
	} else if a.ko.Spec.PolicyDocument != nil && b.ko.Spec.PolicyDocument != nil {
		if *a.ko.Spec.PolicyDocument != *b.ko.Spec.PolicyDocument {
			delta.Add("Spec.PolicyDocument", a.ko.Spec.PolicyDocument, b.ko.Spec.PolicyDocument)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.PrivateDNSEnabled, b.ko.Spec.PrivateDNSEnabled) {
		delta.Add("Spec.PrivateDNSEnabled", a.ko.Spec.PrivateDNSEnabled, b.ko.Spec.PrivateDNSEnabled)
	} else if a.ko.Spec.PrivateDNSEnabled != nil && b.ko.Spec.PrivateDNSEnabled != nil {
		if *a.ko.Spec.PrivateDNSEnabled != *b.ko.Spec.PrivateDNSEnabled {
			delta.Add("Spec.PrivateDNSEnabled", a.ko.Spec.PrivateDNSEnabled, b.ko.Spec.PrivateDNSEnabled)
		}
	}
	if len(a.ko.Spec.RouteTableIDs) != len(b.ko.Spec.RouteTableIDs) {
		delta.Add("Spec.RouteTableIDs", a.ko.Spec.RouteTableIDs, b.ko.Spec.RouteTableIDs)
	} else if len(a.ko.Spec.RouteTableIDs) > 0 {
		if !ackcompare.SliceStringPEqual(a.ko.Spec.RouteTableIDs, b.ko.Spec.RouteTableIDs) {
			delta.Add("Spec.RouteTableIDs", a.ko.Spec.RouteTableIDs, b.ko.Spec.RouteTableIDs)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.RouteTableRefs, b.ko.Spec.RouteTableRefs) {
		delta.Add("Spec.RouteTableRefs", a.ko.Spec.RouteTableRefs, b.ko.Spec.RouteTableRefs)
	}
	if len(a.ko.Spec.SecurityGroupIDs) != len(b.ko.Spec.SecurityGroupIDs) {
		delta.Add("Spec.SecurityGroupIDs", a.ko.Spec.SecurityGroupIDs, b.ko.Spec.SecurityGroupIDs)
	} else if len(a.ko.Spec.SecurityGroupIDs) > 0 {
		if !ackcompare.SliceStringPEqual(a.ko.Spec.SecurityGroupIDs, b.ko.Spec.SecurityGroupIDs) {
			delta.Add("Spec.SecurityGroupIDs", a.ko.Spec.SecurityGroupIDs, b.ko.Spec.SecurityGroupIDs)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.SecurityGroupRefs, b.ko.Spec.SecurityGroupRefs) {
		delta.Add("Spec.SecurityGroupRefs", a.ko.Spec.SecurityGroupRefs, b.ko.Spec.SecurityGroupRefs)
	}
	if ackcompare.HasNilDifference(a.ko.Spec.ServiceName, b.ko.Spec.ServiceName) {
		delta.Add("Spec.ServiceName", a.ko.Spec.ServiceName, b.ko.Spec.ServiceName)
	} else if a.ko.Spec.ServiceName != nil && b.ko.Spec.ServiceName != nil {
		if *a.ko.Spec.ServiceName != *b.ko.Spec.ServiceName {
			delta.Add("Spec.ServiceName", a.ko.Spec.ServiceName, b.ko.Spec.ServiceName)
		}
	}
	if len(a.ko.Spec.SubnetIDs) != len(b.ko.Spec.SubnetIDs) {
		delta.Add("Spec.SubnetIDs", a.ko.Spec.SubnetIDs, b.ko.Spec.SubnetIDs)
	} else if len(a.ko.Spec.SubnetIDs) > 0 {
		if !ackcompare.SliceStringPEqual(a.ko.Spec.SubnetIDs, b.ko.Spec.SubnetIDs) {
			delta.Add("Spec.SubnetIDs", a.ko.Spec.SubnetIDs, b.ko.Spec.SubnetIDs)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.SubnetRefs, b.ko.Spec.SubnetRefs) {
		delta.Add("Spec.SubnetRefs", a.ko.Spec.SubnetRefs, b.ko.Spec.SubnetRefs)
	}
	desiredACKTags, _ := convertToOrderedACKTags(a.ko.Spec.Tags)
	latestACKTags, _ := convertToOrderedACKTags(b.ko.Spec.Tags)
	if !ackcompare.MapStringStringEqual(desiredACKTags, latestACKTags) {
		delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
	}
	if ackcompare.HasNilDifference(a.ko.Spec.VPCEndpointType, b.ko.Spec.VPCEndpointType) {
		delta.Add("Spec.VPCEndpointType", a.ko.Spec.VPCEndpointType, b.ko.Spec.VPCEndpointType)
	} else if a.ko.Spec.VPCEndpointType != nil && b.ko.Spec.VPCEndpointType != nil {
		if *a.ko.Spec.VPCEndpointType != *b.ko.Spec.VPCEndpointType {
			delta.Add("Spec.VPCEndpointType", a.ko.Spec.VPCEndpointType, b.ko.Spec.VPCEndpointType)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.VPCID, b.ko.Spec.VPCID) {
		delta.Add("Spec.VPCID", a.ko.Spec.VPCID, b.ko.Spec.VPCID)
	} else if a.ko.Spec.VPCID != nil && b.ko.Spec.VPCID != nil {
		if *a.ko.Spec.VPCID != *b.ko.Spec.VPCID {
			delta.Add("Spec.VPCID", a.ko.Spec.VPCID, b.ko.Spec.VPCID)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.VPCRef, b.ko.Spec.VPCRef) {
		delta.Add("Spec.VPCRef", a.ko.Spec.VPCRef, b.ko.Spec.VPCRef)
	}

	return delta
}
