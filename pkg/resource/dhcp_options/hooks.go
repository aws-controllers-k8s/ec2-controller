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
	"fmt"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/samber/lo"
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
		fmt.Println("Differ at VPC")
		if err = rm.syncVPCs(ctx, desired, latest); err != nil {
			return nil, err
		}
		updated, err = rm.sdkFind(ctx, desired)
		if err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
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

	fmt.Println("syncVPCstoAdd:", toAdd)
	fmt.Println("syncVPCstoDelete:", toDelete)

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
		Filters: []*svcsdk.Filter{
			{
				Name:   lo.ToPtr("dhcp-options-id"),
				Values: []*string{latest.ko.Status.DHCPOptionsID},
			},
		},
	}

	var resp *svcsdk.DescribeVpcsOutput
	resp, err = rm.sdkapi.DescribeVpcsWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeVpcs", err)
	if err != nil {
		return nil, err
	}
	for _, vpc := range resp.Vpcs {
		vpcID = append(vpcID, vpc.VpcId)
	}

	return vpcID, nil
}

// syncTags used to keep tags in sync by calling Create and Delete API's
func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncTags")
	defer func(err error) {
		exit(err)
	}(err)

	resourceId := []*string{latest.ko.Status.DHCPOptionsID}

	toAdd, toDelete := computeTagsDelta(
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)

	if len(toDelete) > 0 {
		rlog.Debug("removing tags from DHCPOptions resource", "tags", toDelete)
		_, err = rm.sdkapi.DeleteTagsWithContext(
			ctx,
			&svcsdk.DeleteTagsInput{
				Resources: resourceId,
				Tags:      rm.sdkTags(toDelete),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "DeleteTags", err)
		if err != nil {
			return err
		}

	}

	if len(toAdd) > 0 {
		rlog.Debug("adding tags to DHCPOptions resource", "tags", toAdd)
		_, err = rm.sdkapi.CreateTagsWithContext(
			ctx,
			&svcsdk.CreateTagsInput{
				Resources: resourceId,
				Tags:      rm.sdkTags(toAdd),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "CreateTags", err)
		if err != nil {
			return err
		}
	}

	return nil
}

// sdkTags converts *svcapitypes.Tag array to a *svcsdk.Tag array
func (rm *resourceManager) sdkTags(
	tags []*svcapitypes.Tag,
) (sdktags []*svcsdk.Tag) {

	for _, i := range tags {
		sdktag := rm.newTag(*i)
		sdktags = append(sdktags, sdktag)
	}

	return sdktags
}

// computeTagsDelta returns tags to be added and removed from the resource
func computeTagsDelta(
	desired []*svcapitypes.Tag,
	latest []*svcapitypes.Tag,
) (toAdd []*svcapitypes.Tag, toDelete []*svcapitypes.Tag) {

	desiredTags := map[string]string{}
	for _, tag := range desired {
		desiredTags[*tag.Key] = *tag.Value
	}

	latestTags := map[string]string{}
	for _, tag := range latest {
		latestTags[*tag.Key] = *tag.Value
	}

	for _, tag := range desired {
		val, ok := latestTags[*tag.Key]
		if !ok || val != *tag.Value {
			toAdd = append(toAdd, tag)
		}
	}

	for _, tag := range latest {
		_, ok := desiredTags[*tag.Key]
		if !ok {
			toDelete = append(toDelete, tag)
		}
	}

	return toAdd, toDelete

}

// compareTags is a custom comparison function for comparing lists of Tag
// structs where the order of the structs in the list is not important.
func compareTags(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	if len(a.ko.Spec.Tags) != len(b.ko.Spec.Tags) {
		delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
	} else if len(a.ko.Spec.Tags) > 0 {
		addedOrUpdated, removed := computeTagsDelta(a.ko.Spec.Tags, b.ko.Spec.Tags)
		if len(addedOrUpdated) != 0 || len(removed) != 0 {
			delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
		}
	}
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateDhcpOptionsInput.TagSpecification
// and ensures the ResourceType is always set to 'dhcp-options'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateDhcpOptionsInput) {
	input.TagSpecifications = nil
	desiredTagSpecs := svcsdk.TagSpecification{}
	if r.ko.Spec.Tags != nil {
		requestedTags := []*svcsdk.Tag{}
		for _, desiredTag := range r.ko.Spec.Tags {
			// Add in tags defined in the Spec
			tag := &svcsdk.Tag{}
			if desiredTag.Key != nil && desiredTag.Value != nil {
				tag.SetKey(*desiredTag.Key)
				tag.SetValue(*desiredTag.Value)
			}
			requestedTags = append(requestedTags, tag)
		}
		desiredTagSpecs.SetResourceType("dhcp-options")
		desiredTagSpecs.SetTags(requestedTags)
		input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
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
	_, err = rm.sdkapi.AssociateDhcpOptionsWithContext(ctx, input)
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
	_, err = rm.sdkapi.AssociateDhcpOptionsWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "AssociateDhcpOptions", err)
	if err != nil {
		return err
	}
	return nil
}
