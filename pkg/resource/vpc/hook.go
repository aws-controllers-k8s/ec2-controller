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

package vpc

import (
	"context"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

type DNSAttrs struct {
	EnableSupport   *bool
	EnableHostnames *bool
}

func newDescribeVpcAttributePayload(
	vpcID string,
	attribute string,
) *svcsdk.DescribeVpcAttributeInput {
	res := &svcsdk.DescribeVpcAttributeInput{}
	res.SetVpcId(vpcID)
	res.SetAttribute(attribute)
	return res
}

func (rm *resourceManager) getDNSAttributes(
	ctx context.Context,
	vpcID string,
) (res *DNSAttrs, err error) {
	res = &DNSAttrs{}
	dnsSupport, err := rm.sdkapi.DescribeVpcAttributeWithContext(
		ctx,
		newDescribeVpcAttributePayload(vpcID, svcsdk.VpcAttributeNameEnableDnsSupport),
	)
	if err != nil {
		return nil, err
	}
	res.EnableSupport = dnsSupport.EnableDnsSupport.Value

	dnsHostnames, err := rm.sdkapi.DescribeVpcAttributeWithContext(
		ctx,
		newDescribeVpcAttributePayload(vpcID, svcsdk.VpcAttributeNameEnableDnsHostnames),
	)
	if err != nil {
		return nil, err
	}
	res.EnableHostnames = dnsHostnames.EnableDnsHostnames.Value

	return res, nil
}

func newModifyDNSSupportAttributeInputPayload(
	r *resource,
) *svcsdk.ModifyVpcAttributeInput {
	res := &svcsdk.ModifyVpcAttributeInput{}
	res.SetVpcId(*r.ko.Status.VPCID)

	if r.ko.Spec.EnableDNSSupport != nil {
		res.SetEnableDnsSupport(&svcsdk.AttributeBooleanValue{
			Value: r.ko.Spec.EnableDNSSupport,
		})
	}

	return res
}

func newModifyDNSHostnamesAttributeInputPayload(
	r *resource,
) *svcsdk.ModifyVpcAttributeInput {
	res := &svcsdk.ModifyVpcAttributeInput{}
	res.SetVpcId(*r.ko.Status.VPCID)

	if r.ko.Spec.EnableDNSHostnames != nil {
		res.SetEnableDnsHostnames(&svcsdk.AttributeBooleanValue{
			Value: r.ko.Spec.EnableDNSHostnames,
		})
	}

	return res
}

func (rm *resourceManager) syncDNSSupportAttribute(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncDNSSupportAttribute")
	defer exit(err)
	input := newModifyDNSSupportAttributeInputPayload(r)

	_, err = rm.sdkapi.ModifyVpcAttributeWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "ModifyVpcAttribute", err)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) syncDNSHostnamesAttribute(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncDNSHostnamesAttribute")
	defer exit(err)
	input := newModifyDNSHostnamesAttributeInputPayload(r)

	_, err = rm.sdkapi.ModifyVpcAttributeWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "ModifyVpcAttribute", err)
	if err != nil {
		return err
	}

	return nil
}

// computeStringPDifference uses the underlying string value
// to discern which elements are in slice `a` that aren't in slice `b`
// and vice-versa
func computeStringPDifference(a, b []*string) (aNotB, bNotA []*string) {
	mapOfB := map[string]struct{}{}
	for _, elemB := range b {
		mapOfB[*elemB] = struct{}{}
	}
	mapOfA := map[string]struct{}{}
	for _, elemA := range a {
		mapOfA[*elemA] = struct{}{}
	}

	for _, elemA := range a {
		if _, found := mapOfB[*elemA]; !found {
			aNotB = append(aNotB, elemA)
		}
	}
	for _, elemB := range b {
		if _, found := mapOfB[*elemB]; !found {
			bNotA = append(bNotA, elemB)
		}
	}

	return aNotB, bNotA
}

// syncCIDRBlocks analyzes desired and latest
// IPv4 CIDRBlocks and executes API calls to
// Associate/Disassociate CIDRs as needed
func (rm *resourceManager) syncCIDRBlocks(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncCIDRBlocks")
	defer exit(err)

	desiredCIDRs := desired.ko.Spec.CIDRBlocks
	latestCIDRs := latest.ko.Spec.CIDRBlocks
	latestCIDRStates := latest.ko.Status.CIDRBlockAssociationSet
	toAddCIDRs, toDeleteCIDRs := computeStringPDifference(desiredCIDRs, latestCIDRs)

	// extract associationID for the DisassociateVpcCidr request
	for _, cidr := range toDeleteCIDRs {
		input := &svcsdk.DisassociateVpcCidrBlockInput{}
		for _, cidrAssociation := range latestCIDRStates {
			if *cidr == *cidrAssociation.CIDRBlock {
				input.AssociationId = cidrAssociation.AssociationID
				_, err = rm.sdkapi.DisassociateVpcCidrBlockWithContext(ctx, input)
				rm.metrics.RecordAPICall("UPDATE", "DisassociateVpcCidrBlock", err)
				if err != nil {
					return err
				}
			}
		}
	}

	for _, cidr := range toAddCIDRs {
		input := &svcsdk.AssociateVpcCidrBlockInput{
			VpcId:     latest.ko.Status.VPCID,
			CidrBlock: cidr,
		}
		_, err = rm.sdkapi.AssociateVpcCidrBlockWithContext(ctx, input)
		rm.metrics.RecordAPICall("UPDATE", "AssociateVpcCidrBlock", err)
		if err != nil {
			return err
		}
	}

	return nil
}

// setSpecCIDRs sets Spec.CIDRBlocks using the CIDRs in
// Status.CIDRBlockAssociationSet, which is set via sdkCreate/sdkFind
func (rm *resourceManager) setSpecCIDRs(
	ko *svcapitypes.VPC,
) {
	ko.Spec.CIDRBlocks = nil
	if ko.Status.CIDRBlockAssociationSet != nil {
		for _, cidrAssoc := range ko.Status.CIDRBlockAssociationSet {
			ko.Spec.CIDRBlocks = append(ko.Spec.CIDRBlocks, cidrAssoc.CIDRBlock)
		}
	}
}

func (rm *resourceManager) createAttributes(
	ctx context.Context,
	r *resource,
) (err error) {
	if r.ko.Spec.EnableDNSHostnames != nil {
		if err = rm.syncDNSHostnamesAttribute(ctx, r); err != nil {
			return err
		}
	}

	if r.ko.Spec.EnableDNSSupport != nil {
		if err = rm.syncDNSSupportAttribute(ctx, r); err != nil {
			return err
		}
	}

	return nil
}

func (rm *resourceManager) customUpdate(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateVPC")
	defer exit(err)

	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if delta.DifferentAt("Spec.CIDRBlocks") {
		if err := rm.syncCIDRBlocks(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.EnableDNSSupport") {
		if err := rm.syncDNSSupportAttribute(ctx, desired); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.EnableDNSHostnames") {
		if err := rm.syncDNSHostnamesAttribute(ctx, desired); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// applyPrimaryCIDRBlockInCreateRequest populates
// CreateVpcInput.CidrBlock field with the FIRST
// CIDR block defined in the resource's Spec
func applyPrimaryCIDRBlockInCreateRequest(r *resource,
	input *svcsdk.CreateVpcInput) {
	if len(r.ko.Spec.CIDRBlocks) > 0 {
		input.CidrBlock = r.ko.Spec.CIDRBlocks[0]
	}
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

	resourceId := []*string{latest.ko.Status.VPCID}

	toAdd, toDelete := computeTagsDelta(
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)

	if len(toDelete) > 0 {
		rlog.Debug("removing tags from parameter group", "tags", toDelete)
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
		rlog.Debug("adding tags to parameter group", "tags", toAdd)
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

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateVpcInput.TagSpecification
// and ensures the ResourceType is always set to 'vpc'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateVpcInput) {
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
		desiredTagSpecs.SetResourceType("vpc")
		desiredTagSpecs.SetTags(requestedTags)
	}
	input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
}
