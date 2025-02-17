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

package vpc_endpoint

import (
	"context"
	"errors"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"

	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/ec2/client/describe_vpc_endpoints.html
const (
	StatusPendingAcceptance = "pendingAcceptance"
	StatusPending           = "pending"
	StatusAvailable         = "available"
	StatusDeleting          = "deleting"
	StatusDeleted           = "deleted"
	StatusRejected          = "rejected"
	StatusFailed            = "failed"
)

func vpcEndpointPending(r *resource) bool {
	if r.ko.Status.State == nil {
		return false
	}
	cs := *r.ko.Status.State
	return cs == StatusPending
}

// addIDToDeleteRequest adds resource's Vpc Endpoint ID to DeleteRequest.
// Return error to indicate to callers that the resource is not yet created.
func addIDToDeleteRequest(r *resource,
	input *svcsdk.DeleteVpcEndpointsInput) error {
	if r.ko.Status.VPCEndpointID == nil {
		return errors.New("unable to extract VPCEndpointID from resource")
	}
	input.VpcEndpointIds = []string{*r.ko.Status.VPCEndpointID}
	return nil
}

func (rm *resourceManager) customUpdateVPCEndpoint(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateVPCEndpoint")
	defer exit(err)

	// Default `updated` to `desired` because it is likely
	// EC2 `modify` APIs do NOT return output, only errors.
	// If the `modify` calls (i.e. `sync`) do NOT return
	// an error, then the update was successful and desired.Spec
	// (now updated.Spec) reflects the latest resource state.
	updated = rm.concreteResource(desired.DeepCopy())

	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.VPCEndpointID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	if !delta.DifferentExcept("Spec.Tags") {
        return desired, nil
    }

	// Handle modifications that require ModifyVpcEndpoint API call
	// avoid making the ModifyVpcEndpoint API call if only tags are modified
	input := &svcsdk.ModifyVpcEndpointInput{
		VpcEndpointId: latest.ko.Status.VPCEndpointID,
	}

	if delta.DifferentAt("Spec.SubnetIDs") {
		toAdd, toRemove := calculateSubnetDifferences(
			aws.ToStringSlice(desired.ko.Spec.SubnetIDs),
			aws.ToStringSlice(latest.ko.Spec.SubnetIDs))
		input.AddSubnetIds = toAdd
		input.RemoveSubnetIds = toRemove
	}

	if delta.DifferentAt("Spec.RouteTableIDs") {
		toAdd, toRemove := calculateSubnetDifferences(
			aws.ToStringSlice(desired.ko.Spec.RouteTableIDs),
			aws.ToStringSlice(latest.ko.Spec.RouteTableIDs))
		input.AddRouteTableIds = toAdd
		input.RemoveRouteTableIds = toRemove
	}

	if delta.DifferentAt("Spec.PolicyDocument") {
		input.PolicyDocument = desired.ko.Spec.PolicyDocument
	}

	if delta.DifferentAt("Spec.PrivateDNSEnabled") {
		input.PrivateDnsEnabled = desired.ko.Spec.PrivateDNSEnabled
	}

	if delta.DifferentAt("Spec.SecurityGroupIDs") {
		toAdd, toRemove := calculateSubnetDifferences(
			aws.ToStringSlice(desired.ko.Spec.SecurityGroupIDs),
			aws.ToStringSlice(latest.ko.Spec.SecurityGroupIDs))
		input.AddSecurityGroupIds = toAdd
		input.RemoveSecurityGroupIds = toRemove
	}

	if delta.DifferentAt("Spec.DNSOptions") && desired.ko.Spec.DNSOptions != nil {
		if desired.ko.Spec.DNSOptions != nil {
			input.DnsOptions = &svcsdktypes.DnsOptionsSpecification{
				DnsRecordIpType: svcsdktypes.DnsRecordIpType(*desired.ko.Spec.DNSOptions.DNSRecordIPType),
			}
		}
	}

	if delta.DifferentAt("Spec.IPAddressType") {
		if desired.ko.Spec.IPAddressType != nil {
			input.IpAddressType = svcsdktypes.IpAddressType(*desired.ko.Spec.IPAddressType)
		}
	}

	_, err = rm.sdkapi.ModifyVpcEndpoint(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "ModifyVpcEndpoint", err)
		if err != nil {
			return nil, err
	}
	return updated, nil
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateVpcEndpointInput.TagSpecification
// and ensures the ResourceType is always set to 'vpc-endpoint'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateVpcEndpointInput) {
	input.TagSpecifications = nil
	desiredTagSpecs := svcsdktypes.TagSpecification{}
	if r.ko.Spec.Tags != nil {
		requestedTags := []svcsdktypes.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {
			// Add in tags defined in the Spec
			tag := svcsdktypes.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				tag.Key = desiredTag.Key
				tag.Value = desiredTag.Value
			}
			requestedTags = append(requestedTags, tag)
		}
		desiredTagSpecs.ResourceType = "vpc-endpoint"
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}

// calculateSubnetDifferences returns two slices:
// 1. Elements in desired that are not in latest (to add)
// 2. Elements in latest that are not in desired (to remove)
func calculateSubnetDifferences(desired, latest []string) ([]string, []string) {
	if desired == nil {
		desired = []string{}
	}
	if latest == nil {
		latest = []string{}
	}

	desiredMap := make(map[string]bool)
	latestMap := make(map[string]bool)

	for _, id := range desired {
		desiredMap[id] = true
	}
	for _, id := range latest {
		latestMap[id] = true
	}

	var toAdd []string
	var toRemove []string

	// Find elements to add (in desired but not in latest)
	for id := range desiredMap {
		if !latestMap[id] {
			toAdd = append(toAdd, id)
		}
	}

	// Find elements to remove (in latest but not in desired)
	for id := range latestMap {
		if !desiredMap[id] {
			toRemove = append(toRemove, id)
		}
	}

	return toAdd, toRemove
}
