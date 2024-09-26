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
	"fmt"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go/aws/awserr"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
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
		if _, found := mapOfA[*elemB]; !found {
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
	cidrblockassociationset := []*svcapitypes.VPCCIDRBlockAssociation{}

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
			} else {
				cidrblockassociationset = append(cidrblockassociationset, cidrAssociation)
			}
		}
	}

	for _, cidr := range toAddCIDRs {
		input := &svcsdk.AssociateVpcCidrBlockInput{
			VpcId:     latest.ko.Status.VPCID,
			CidrBlock: cidr,
		}
		var res *svcsdk.AssociateVpcCidrBlockOutput
		cidrblockassociation := &svcapitypes.VPCCIDRBlockAssociation{}
		res, err = rm.sdkapi.AssociateVpcCidrBlockWithContext(ctx, input)
		rm.metrics.RecordAPICall("UPDATE", "AssociateVpcCidrBlock", err)
		if err != nil {
			return err
		}
		if res.CidrBlockAssociation != nil {
			if res.CidrBlockAssociation.AssociationId != nil {
				cidrblockassociation.AssociationID = res.CidrBlockAssociation.AssociationId
			}
			if res.CidrBlockAssociation.CidrBlock != nil {
				cidrblockassociation.CIDRBlock = res.CidrBlockAssociation.CidrBlock
			}
			if res.CidrBlockAssociation.CidrBlockState != nil {
				cidrblockstate := &svcapitypes.VPCCIDRBlockState{}
				if res.CidrBlockAssociation.CidrBlockState.State != nil {
					cidrblockstate.State = res.CidrBlockAssociation.CidrBlockState.State
				}
				if res.CidrBlockAssociation.CidrBlockState.StatusMessage != nil {
					cidrblockstate.StatusMessage = res.CidrBlockAssociation.CidrBlockState.StatusMessage
				}
			}
			cidrblockassociationset = append(cidrblockassociationset, cidrblockassociation)
		}
	}
	if cidrblockassociationset != nil {
		if toDeleteCIDRs != nil {
			latest.ko.Status.CIDRBlockAssociationSet = cidrblockassociationset
		} else {
			latest.ko.Status.CIDRBlockAssociationSet = append(latest.ko.Status.CIDRBlockAssociationSet, cidrblockassociationset...)
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

func (rm *resourceManager) customUpdateVPC(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateVPC")
	defer exit(err)

	// Default `updated` to `desired` because it is likely
	// EC2 `modify` APIs do NOT return output, only errors.
	// If the `modify` calls (i.e. `sync`) do NOT return
	// an error, then the update was successful and desired.Spec
	// (now updated.Spec) reflects the latest resource state.
	updated = rm.concreteResource(desired.DeepCopy())

	if delta.DifferentAt("Spec.CIDRBlocks") {
		if err := rm.syncCIDRBlocks(ctx, desired, latest); err != nil {
			return nil, err
		}
	}
	updated.ko.Status.CIDRBlockAssociationSet = latest.ko.Status.CIDRBlockAssociationSet

	if delta.DifferentAt("Spec.DisallowSecurityGroupDefaultRules") {
		if desired.ko.Spec.DisallowSecurityGroupDefaultRules != nil && *desired.ko.Spec.DisallowSecurityGroupDefaultRules {
			if err = rm.deleteSecurityGroupDefaultRules(ctx, desired); err != nil {
				// if deleteSecurityGroupDefaultRules fails, assume that the
				// rules still exist and update the status accordingly.
				exist := true
				updated.ko.Status.SecurityGroupDefaultRulesExist = &exist
				return nil, err
			}
			exist := false
			updated.ko.Status.SecurityGroupDefaultRulesExist = &exist
		}
	}

	if delta.DifferentAt("Spec.EnableDNSSupport") {
		if desired.ko.Spec.EnableDNSSupport != nil {
			if err := rm.syncDNSSupportAttribute(ctx, desired); err != nil {
				return nil, err
			}
		}
	}

	if delta.DifferentAt("Spec.EnableDNSHostnames") {
		if desired.ko.Spec.EnableDNSHostnames != nil {
			if err := rm.syncDNSHostnamesAttribute(ctx, desired); err != nil {
				return nil, err
			}
		}
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.VPCID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	return updated, nil
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

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateVpcInput.TagSpecification
// and ensures the ResourceType is always set to 'vpc'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateVpcInput) {
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
		desiredTagSpecs.SetResourceType("vpc")
		desiredTagSpecs.SetTags(requestedTags)
		input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
	}
}

// hasSecurityGroupDefaultRules returns true if the vpc's 'default' security
// group has autopopulated ingress/egress rules.
func (rm *resourceManager) hasSecurityGroupDefaultRules(
	ctx context.Context,
	r *resource,
) (defaultSGRulePresent bool, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.hasSecurityGroupDefaultRules")
	defer exit(err)

	sgID, err := rm.getDefaultSGId(ctx, r)
	if err != nil {
		return false, err
	}

	groupIDFilter := "group-id"
	input := &svcsdk.DescribeSecurityGroupRulesInput{
		Filters: []*svcsdk.Filter{
			{
				Name:   &groupIDFilter,
				Values: []*string{sgID},
			},
		},
	}

	for {
		resp, err := rm.sdkapi.DescribeSecurityGroupRulesWithContext(ctx, input)
		rm.metrics.RecordAPICall("READ_MANY", "DescribeSecurityGroupRules", err)
		if err != nil || resp == nil {
			break
		}
		for _, sgRule := range resp.SecurityGroupRules {
			if rm.isDefaultSGIngressRule(sgRule) {
				return true, nil
			}
			if rm.isDefaultSGEgressRule(sgRule) {
				return true, nil
			}
		}
		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		input.SetNextToken(*resp.NextToken)
	}
	if err != nil {
		return false, err
	}

	return false, nil
}

// isDefaultSGIngressRule returns true if the SG ingress rule passed to the
// function is the auto populated ingress rule.
func (rm *resourceManager) isDefaultSGIngressRule(
	rule *svcsdk.SecurityGroupRule,
) bool {
	if rule.FromPort == nil || rule.ToPort == nil || rule.IpProtocol == nil || rule.IsEgress == nil || rule.ReferencedGroupInfo == nil || rule.ReferencedGroupInfo.GroupId == nil || rule.GroupId == nil {
		return false
	}

	if *rule.ReferencedGroupInfo.GroupId == *rule.GroupId &&
		*rule.FromPort == -1 &&
		*rule.ToPort == -1 &&
		*rule.IpProtocol == "-1" &&
		!*rule.IsEgress {
		return true
	}
	return false
}

// isDefaultSGEgressRule returns true if the SG egress rule passed to the
// function is the auto populated egress rule.
func (rm *resourceManager) isDefaultSGEgressRule(
	rule *svcsdk.SecurityGroupRule,
) bool {
	if rule.CidrIpv4 == nil || rule.FromPort == nil || rule.ToPort == nil || rule.IpProtocol == nil || rule.IsEgress == nil {
		return false
	}

	if *rule.CidrIpv4 == "0.0.0.0/0" &&
		*rule.FromPort == -1 &&
		*rule.ToPort == -1 &&
		*rule.IpProtocol == "-1" &&
		*rule.IsEgress {
		return true
	}
	return false
}

// deleteSecurityGroupDefaultRules deletes the ingress/egress rule that is
// attached to the 'default' SecurityGroup upon creation.
func (rm *resourceManager) deleteSecurityGroupDefaultRules(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteSecurityGroupDefaultRules")
	defer exit(err)

	sgID, err := rm.getDefaultSGId(ctx, r)
	if err != nil {
		return err
	}

	ipRange := &svcsdk.IpRange{
		CidrIp: ptr("0.0.0.0/0"),
	}
	egressInput := &svcsdk.IpPermission{
		FromPort:   ptr(int64(-1)),
		ToPort:     ptr(int64(-1)),
		IpProtocol: ptr("-1"),
		IpRanges:   []*svcsdk.IpRange{ipRange},
	}
	egressReq := &svcsdk.RevokeSecurityGroupEgressInput{
		GroupId:       sgID,
		IpPermissions: []*svcsdk.IpPermission{egressInput},
	}
	_, err = rm.sdkapi.RevokeSecurityGroupEgressWithContext(ctx, egressReq)
	rm.metrics.RecordAPICall("DELETE", "RevokeSecurityGroupEgress", err)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidPermission.NotFound":
				return
			}
		}
		return err
	}

	IngressInput := &svcsdk.IpPermission{
		FromPort:   ptr(int64(-1)),
		ToPort:     ptr(int64(-1)),
		IpProtocol: ptr("-1"),
		UserIdGroupPairs: []*svcsdk.UserIdGroupPair{
			{
				GroupId: sgID,
			},
		},
	}
	ingressReq := &svcsdk.RevokeSecurityGroupIngressInput{
		GroupId:       sgID,
		IpPermissions: []*svcsdk.IpPermission{IngressInput},
	}
	_, err = rm.sdkapi.RevokeSecurityGroupIngressWithContext(ctx, ingressReq)
	rm.metrics.RecordAPICall("DELETE", "RevokeSecurityGroupIngress", err)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidPermission.NotFound":
				return
			}
		}
		return err
	}

	return err
}

// getDefaultSGId calls DescribeSecurityGroups with filters as vpc-id and security
// group name ('default') and returns the security groupd id
func (rm *resourceManager) getDefaultSGId(
	ctx context.Context,
	res *resource,
) (sgID *string, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getRules")
	defer exit(err)

	vpcIDFilter := "vpc-id"
	groupNameFilter := "group-name"
	groupNameValue := "default"
	input := &svcsdk.DescribeSecurityGroupsInput{
		Filters: []*svcsdk.Filter{
			{
				Name:   &vpcIDFilter,
				Values: []*string{res.ko.Status.VPCID},
			},
			{
				Name:   &groupNameFilter,
				Values: []*string{&groupNameValue},
			},
		},
	}

	resp, err := rm.sdkapi.DescribeSecurityGroupsWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeSecurityGroupRules", err)
	if err != nil || resp == nil {
		return nil, err
	}

	if len(resp.SecurityGroups) == 0 || resp.SecurityGroups[0] == nil {
		return nil, fmt.Errorf("default security group not found")
	}

	return resp.SecurityGroups[0].GroupId, nil
}

func ptr[T any](t T) *T {
	return &t
}
