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

package dhcp_options

import (
	"context"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/samber/lo"

	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
)

func (rm *resourceManager) customUpdateDHCPOptions(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateDHCPOptions")
	defer exit(err)

	// Default `updated` to `desired` because it is likely
	// EC2 `modify` APIs do NOT return output, only errors.
	// If the `modify` calls (i.e. `sync`) do NOT return
	// an error, then the update was successful and desired.Spec
	// (now updated.Spec) reflects the latest resource state.
	updated = rm.concreteResource(desired.DeepCopy())

	if delta.DifferentAt("Spec.VPC") {
		if err = rm.syncVPCs(ctx, desired, latest); err != nil {
			return nil, err
		}
		updated, err = rm.sdkFind(ctx, desired)
		if err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.DHCPOptionsID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

func (rm *resourceManager) syncVPCs(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncEntries")
	defer exit(err)

	latestVPC := []string{}
	desiredVPC := []string{}

	if latest != nil {
		for _, vpc := range latest.ko.Spec.VPC {
			latestVPC = append(latestVPC, *vpc)
		}
	}
	for _, vpc := range desired.ko.Spec.VPC {
		desiredVPC = append(desiredVPC, *vpc)
	}

	toAdd, toDelete := lo.Difference(desiredVPC, latestVPC)

	for _, vpc := range toAdd {
		rm.attachToVPC(ctx, desired, vpc)

	}
	for _, vpc := range toDelete {
		rm.detachFromVPC(ctx, desired, vpc)

	}
	return nil
}

func (rm *resourceManager) getAttachedVPC(
	ctx context.Context,
	latest *resource,
) (vpcID []*string, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getAttachedVPC")
	defer func(err error) {
		exit(err)
	}(err)

	input := &svcsdk.DescribeVpcsInput{
		Filters: []svcsdktypes.Filter{
			{
				Name:   lo.ToPtr("dhcp-options-id"),
				Values: []string{*latest.ko.Status.DHCPOptionsID},
			},
		},
	}

	var resp *svcsdk.DescribeVpcsOutput
	resp, err = rm.sdkapi.DescribeVpcs(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeVpcs", err)
	if err != nil {
		return nil, err
	}
	for _, vpc := range resp.Vpcs {
		vpcID = append(vpcID, vpc.VpcId)
	}

	return vpcID, nil
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateDhcpOptionsInput.TagSpecification
// and ensures the ResourceType is always set to 'dhcp-options'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateDhcpOptionsInput) {
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
		desiredTagSpecs.ResourceType = "dhcp-options"
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}

func (rm *resourceManager) attachToVPC(
	ctx context.Context,
	desired *resource,
	vpc string,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.attachToVPC")
	defer func(err error) {
		exit(err)
	}(err)

	if vpc == "" {
		return nil
	}
	input := &svcsdk.AssociateDhcpOptionsInput{
		DhcpOptionsId: desired.ko.Status.DHCPOptionsID,
		VpcId:         &vpc,
	}
	_, err = rm.sdkapi.AssociateDhcpOptions(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "AssociateDhcpOptions", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) detachFromVPC(
	ctx context.Context,
	desired *resource,
	vpc string,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.detachFromVPC")
	defer func(err error) {
		exit(err)
	}(err)

	input := &svcsdk.AssociateDhcpOptionsInput{
		DhcpOptionsId: lo.ToPtr("default"),
		VpcId:         &vpc,
	}
	_, err = rm.sdkapi.AssociateDhcpOptions(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "AssociateDhcpOptions", err)
	if err != nil {
		return err
	}
	return nil
}
