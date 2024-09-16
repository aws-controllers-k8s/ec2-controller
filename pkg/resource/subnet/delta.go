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

package subnet

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

	if ackcompare.HasNilDifference(a.ko.Spec.AssignIPv6AddressOnCreation, b.ko.Spec.AssignIPv6AddressOnCreation) {
		delta.Add("Spec.AssignIPv6AddressOnCreation", a.ko.Spec.AssignIPv6AddressOnCreation, b.ko.Spec.AssignIPv6AddressOnCreation)
	} else if a.ko.Spec.AssignIPv6AddressOnCreation != nil && b.ko.Spec.AssignIPv6AddressOnCreation != nil {
		if *a.ko.Spec.AssignIPv6AddressOnCreation != *b.ko.Spec.AssignIPv6AddressOnCreation {
			delta.Add("Spec.AssignIPv6AddressOnCreation", a.ko.Spec.AssignIPv6AddressOnCreation, b.ko.Spec.AssignIPv6AddressOnCreation)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.AvailabilityZone, b.ko.Spec.AvailabilityZone) {
		delta.Add("Spec.AvailabilityZone", a.ko.Spec.AvailabilityZone, b.ko.Spec.AvailabilityZone)
	} else if a.ko.Spec.AvailabilityZone != nil && b.ko.Spec.AvailabilityZone != nil {
		if *a.ko.Spec.AvailabilityZone != *b.ko.Spec.AvailabilityZone {
			delta.Add("Spec.AvailabilityZone", a.ko.Spec.AvailabilityZone, b.ko.Spec.AvailabilityZone)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.AvailabilityZoneID, b.ko.Spec.AvailabilityZoneID) {
		delta.Add("Spec.AvailabilityZoneID", a.ko.Spec.AvailabilityZoneID, b.ko.Spec.AvailabilityZoneID)
	} else if a.ko.Spec.AvailabilityZoneID != nil && b.ko.Spec.AvailabilityZoneID != nil {
		if *a.ko.Spec.AvailabilityZoneID != *b.ko.Spec.AvailabilityZoneID {
			delta.Add("Spec.AvailabilityZoneID", a.ko.Spec.AvailabilityZoneID, b.ko.Spec.AvailabilityZoneID)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.CIDRBlock, b.ko.Spec.CIDRBlock) {
		delta.Add("Spec.CIDRBlock", a.ko.Spec.CIDRBlock, b.ko.Spec.CIDRBlock)
	} else if a.ko.Spec.CIDRBlock != nil && b.ko.Spec.CIDRBlock != nil {
		if *a.ko.Spec.CIDRBlock != *b.ko.Spec.CIDRBlock {
			delta.Add("Spec.CIDRBlock", a.ko.Spec.CIDRBlock, b.ko.Spec.CIDRBlock)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.CustomerOwnedIPv4Pool, b.ko.Spec.CustomerOwnedIPv4Pool) {
		delta.Add("Spec.CustomerOwnedIPv4Pool", a.ko.Spec.CustomerOwnedIPv4Pool, b.ko.Spec.CustomerOwnedIPv4Pool)
	} else if a.ko.Spec.CustomerOwnedIPv4Pool != nil && b.ko.Spec.CustomerOwnedIPv4Pool != nil {
		if *a.ko.Spec.CustomerOwnedIPv4Pool != *b.ko.Spec.CustomerOwnedIPv4Pool {
			delta.Add("Spec.CustomerOwnedIPv4Pool", a.ko.Spec.CustomerOwnedIPv4Pool, b.ko.Spec.CustomerOwnedIPv4Pool)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.EnableDNS64, b.ko.Spec.EnableDNS64) {
		delta.Add("Spec.EnableDNS64", a.ko.Spec.EnableDNS64, b.ko.Spec.EnableDNS64)
	} else if a.ko.Spec.EnableDNS64 != nil && b.ko.Spec.EnableDNS64 != nil {
		if *a.ko.Spec.EnableDNS64 != *b.ko.Spec.EnableDNS64 {
			delta.Add("Spec.EnableDNS64", a.ko.Spec.EnableDNS64, b.ko.Spec.EnableDNS64)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.EnableResourceNameDNSAAAARecord, b.ko.Spec.EnableResourceNameDNSAAAARecord) {
		delta.Add("Spec.EnableResourceNameDNSAAAARecord", a.ko.Spec.EnableResourceNameDNSAAAARecord, b.ko.Spec.EnableResourceNameDNSAAAARecord)
	} else if a.ko.Spec.EnableResourceNameDNSAAAARecord != nil && b.ko.Spec.EnableResourceNameDNSAAAARecord != nil {
		if *a.ko.Spec.EnableResourceNameDNSAAAARecord != *b.ko.Spec.EnableResourceNameDNSAAAARecord {
			delta.Add("Spec.EnableResourceNameDNSAAAARecord", a.ko.Spec.EnableResourceNameDNSAAAARecord, b.ko.Spec.EnableResourceNameDNSAAAARecord)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.EnableResourceNameDNSARecord, b.ko.Spec.EnableResourceNameDNSARecord) {
		delta.Add("Spec.EnableResourceNameDNSARecord", a.ko.Spec.EnableResourceNameDNSARecord, b.ko.Spec.EnableResourceNameDNSARecord)
	} else if a.ko.Spec.EnableResourceNameDNSARecord != nil && b.ko.Spec.EnableResourceNameDNSARecord != nil {
		if *a.ko.Spec.EnableResourceNameDNSARecord != *b.ko.Spec.EnableResourceNameDNSARecord {
			delta.Add("Spec.EnableResourceNameDNSARecord", a.ko.Spec.EnableResourceNameDNSARecord, b.ko.Spec.EnableResourceNameDNSARecord)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.HostnameType, b.ko.Spec.HostnameType) {
		delta.Add("Spec.HostnameType", a.ko.Spec.HostnameType, b.ko.Spec.HostnameType)
	} else if a.ko.Spec.HostnameType != nil && b.ko.Spec.HostnameType != nil {
		if *a.ko.Spec.HostnameType != *b.ko.Spec.HostnameType {
			delta.Add("Spec.HostnameType", a.ko.Spec.HostnameType, b.ko.Spec.HostnameType)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.IPv6CIDRBlock, b.ko.Spec.IPv6CIDRBlock) {
		delta.Add("Spec.IPv6CIDRBlock", a.ko.Spec.IPv6CIDRBlock, b.ko.Spec.IPv6CIDRBlock)
	} else if a.ko.Spec.IPv6CIDRBlock != nil && b.ko.Spec.IPv6CIDRBlock != nil {
		if *a.ko.Spec.IPv6CIDRBlock != *b.ko.Spec.IPv6CIDRBlock {
			delta.Add("Spec.IPv6CIDRBlock", a.ko.Spec.IPv6CIDRBlock, b.ko.Spec.IPv6CIDRBlock)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.IPv6Native, b.ko.Spec.IPv6Native) {
		delta.Add("Spec.IPv6Native", a.ko.Spec.IPv6Native, b.ko.Spec.IPv6Native)
	} else if a.ko.Spec.IPv6Native != nil && b.ko.Spec.IPv6Native != nil {
		if *a.ko.Spec.IPv6Native != *b.ko.Spec.IPv6Native {
			delta.Add("Spec.IPv6Native", a.ko.Spec.IPv6Native, b.ko.Spec.IPv6Native)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.MapPublicIPOnLaunch, b.ko.Spec.MapPublicIPOnLaunch) {
		delta.Add("Spec.MapPublicIPOnLaunch", a.ko.Spec.MapPublicIPOnLaunch, b.ko.Spec.MapPublicIPOnLaunch)
	} else if a.ko.Spec.MapPublicIPOnLaunch != nil && b.ko.Spec.MapPublicIPOnLaunch != nil {
		if *a.ko.Spec.MapPublicIPOnLaunch != *b.ko.Spec.MapPublicIPOnLaunch {
			delta.Add("Spec.MapPublicIPOnLaunch", a.ko.Spec.MapPublicIPOnLaunch, b.ko.Spec.MapPublicIPOnLaunch)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.OutpostARN, b.ko.Spec.OutpostARN) {
		delta.Add("Spec.OutpostARN", a.ko.Spec.OutpostARN, b.ko.Spec.OutpostARN)
	} else if a.ko.Spec.OutpostARN != nil && b.ko.Spec.OutpostARN != nil {
		if *a.ko.Spec.OutpostARN != *b.ko.Spec.OutpostARN {
			delta.Add("Spec.OutpostARN", a.ko.Spec.OutpostARN, b.ko.Spec.OutpostARN)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.RouteTableRefs, b.ko.Spec.RouteTableRefs) {
		delta.Add("Spec.RouteTableRefs", a.ko.Spec.RouteTableRefs, b.ko.Spec.RouteTableRefs)
	}
	if len(a.ko.Spec.RouteTables) != len(b.ko.Spec.RouteTables) {
		delta.Add("Spec.RouteTables", a.ko.Spec.RouteTables, b.ko.Spec.RouteTables)
	} else if len(a.ko.Spec.RouteTables) > 0 {
		if !ackcompare.SliceStringPEqual(a.ko.Spec.RouteTables, b.ko.Spec.RouteTables) {
			delta.Add("Spec.RouteTables", a.ko.Spec.RouteTables, b.ko.Spec.RouteTables)
		}
	}
	if !ackcompare.MapStringStringEqual(ToACKTags(a.ko.Spec.Tags), ToACKTags(b.ko.Spec.Tags)) {
		delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
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
