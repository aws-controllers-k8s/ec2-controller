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

package security_group

import (
	"context"
	"fmt"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

func (rm *resourceManager) addRulesToSpec(
	ko *svcapitypes.SecurityGroup,
	resp *svcsdk.SecurityGroup,
) {
	if resp.IpPermissions != nil {
		specIngress := []*svcapitypes.IPPermission{}
		for _, ip := range resp.IpPermissions {
			specIngress = append(specIngress, rm.setResourceIPPermission(ip))
		}
		ko.Spec.IngressRules = specIngress
	}
	if resp.IpPermissionsEgress != nil {
		specEgress := []*svcapitypes.IPPermission{}
		for _, ep := range resp.IpPermissionsEgress {
			specEgress = append(specEgress, rm.setResourceIPPermission(ep))
		}
		ko.Spec.EgressRules = specEgress
	}
}

func (rm *resourceManager) addRulesToStatus(
	ko *svcapitypes.SecurityGroup,
	ctx context.Context,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.addRulesToStatus")
	defer exit(err)

	groupIDFilter := "group-id"
	input := &svcsdk.DescribeSecurityGroupRulesInput{
		Filters: []*svcsdk.Filter{
			{
				Name:   &groupIDFilter,
				Values: []*string{ko.Status.ID},
			},
		},
	}
	rulesForResource := []*svcapitypes.SecurityGroupRule{}
	for {
		resp, err := rm.sdkapi.DescribeSecurityGroupRulesWithContext(ctx, input)
		rm.metrics.RecordAPICall("READ_MANY", "DescribeSecurityGroupRules", err)
		if err != nil || resp == nil {
			break
		}
		for _, sgRule := range resp.SecurityGroupRules {
			rulesForResource = append(rulesForResource, rm.setResourceSecurityGroupRule(sgRule))
		}
		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		input.SetNextToken(*resp.NextToken)
	}
	if err != nil {
		return err
	}

	ko.Status.Rules = rulesForResource
	return nil
}

func (rm *resourceManager) requiredFieldsMissingForSGRule(
	r *resource,
) bool {
	return r.ko.Status.ID == nil
}

func (rm *resourceManager) removeDefaultEgressRule(
	ctx context.Context,
	res *resource,
) (err error) {
	err = rm.deleteSecurityGroupRules(ctx, res, nil, []*svcapitypes.IPPermission{defaultEgressRule()})
	if err != nil {
		return err
	}
	return rm.addRulesToStatus(res.ko, ctx)
}

func (rm *resourceManager) syncSGRules(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncSGRules")
	defer exit(err)
	toAddIngress := []*svcapitypes.IPPermission{}
	toAddEgress := []*svcapitypes.IPPermission{}
	toDeleteIngress := []*svcapitypes.IPPermission{}
	toDeleteEgress := []*svcapitypes.IPPermission{}

	for _, desiredIngress := range desired.ko.Spec.IngressRules {
		if latest == nil || !contains(latest.ko.Spec.IngressRules, desiredIngress) {
			// a desired rule is not in the latest resource; therefore, create
			toAddIngress = append(toAddIngress, desiredIngress)
		}
	}
	for _, desiredEgress := range desired.ko.Spec.EgressRules {
		if latest == nil || !contains(latest.ko.Spec.EgressRules, desiredEgress) {
			toAddEgress = append(toAddEgress, desiredEgress)
		}
	}
	if latest != nil {
		for _, latestIngress := range latest.ko.Spec.IngressRules {
			if !contains(desired.ko.Spec.IngressRules, latestIngress) {
				// a rule is in latest resource, but not in desired resource; therefore, delete
				toDeleteIngress = append(toDeleteIngress, latestIngress)
			}
		}
		for _, latestEgress := range latest.ko.Spec.EgressRules {
			if !contains(desired.ko.Spec.EgressRules, latestEgress) {
				toDeleteEgress = append(toDeleteEgress, latestEgress)
			}
		}
	}

	// remove default egress rule iff user defines their own
	if len(toAddEgress) > 0 && !contains(toAddEgress, defaultEgressRule()) {
		fmt.Printf("desired before remove Default egress rule: %+v\n", *desired)
		rm.removeDefaultEgressRule(ctx, desired)
		fmt.Printf("desired after remove Default egress rule: %+v\n", *desired)
	}

	// Delete before create for the following reasons:
	// - Updating a rule requires that it be removed before the updated version be added.
	// - If there is an error with adding new rules, it occurs after deletion of old ones;
	//   This is safer and closer to achieving desired resource state.
	if err = rm.deleteSecurityGroupRules(ctx, latest, toDeleteIngress, toDeleteEgress); err != nil {
		return err
	}
	fmt.Printf("desired after deleteSecurityGroupRules: %+v\n", *desired)
	if err = rm.createSecurityGroupRules(ctx, desired, toAddIngress, toAddEgress); err != nil {
		return err
	}
	fmt.Printf("desired after createSecurityGroupRules: %+v\n", *desired)
	return nil
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateSecurityGroupInput.TagSpecification
// and ensures the ResourceType is always set to 'security-group'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateSecurityGroupInput) {
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
		desiredTagSpecs.SetResourceType("security-group")
		desiredTagSpecs.SetTags(requestedTags)
	}
	input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
}

func (rm *resourceManager) createSecurityGroupRules(
	ctx context.Context,
	r *resource,
	ingress []*svcapitypes.IPPermission,
	egress []*svcapitypes.IPPermission,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.createSecurityGroupRules")
	defer exit(err)

	// Authorize ingress rules
	for _, i := range ingress {
		ipInput := rm.newIPPermission(*i)
		req := &svcsdk.AuthorizeSecurityGroupIngressInput{
			GroupId:       r.ko.Status.ID,
			IpPermissions: []*svcsdk.IpPermission{ipInput},
			// TODO: TagSpecs
		}
		_, err := rm.sdkapi.AuthorizeSecurityGroupIngressWithContext(ctx, req)
		rm.metrics.RecordAPICall("CREATE", "AuthorizeSecurityGroupIngress", err)
		if err != nil {
			return err
		}
	}

	// Authorize egress rules
	for _, e := range egress {
		ipInput := rm.newIPPermission(*e)
		req := &svcsdk.AuthorizeSecurityGroupEgressInput{
			GroupId:       r.ko.Status.ID,
			IpPermissions: []*svcsdk.IpPermission{ipInput},
			// TODO: TagSpecs
		}
		_, err = rm.sdkapi.AuthorizeSecurityGroupEgressWithContext(ctx, req)
		rm.metrics.RecordAPICall("CREATE", "AuthorizeSecurityGroupEgress", err)
		if err != nil {
			return err
		}
	}

	return err
}

func (rm *resourceManager) deleteSecurityGroupRules(
	ctx context.Context,
	r *resource,
	ingress []*svcapitypes.IPPermission,
	egress []*svcapitypes.IPPermission,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteSecurityGroupRules")
	defer exit(err)

	// Revoke ingress rules
	for _, i := range ingress {
		ipInput := rm.newIPPermission(*i)
		req := &svcsdk.RevokeSecurityGroupIngressInput{
			GroupId:       r.ko.Status.ID,
			IpPermissions: []*svcsdk.IpPermission{ipInput},
		}
		_, err = rm.sdkapi.RevokeSecurityGroupIngressWithContext(ctx, req)
		rm.metrics.RecordAPICall("DELETE", "RevokeSecurityGroupIngress", err)
		if err != nil {
			return err
		}
	}

	// Revoke egress rules
	for _, e := range egress {
		ipInput := rm.newIPPermission(*e)
		req := &svcsdk.RevokeSecurityGroupEgressInput{
			GroupId:       r.ko.Status.ID,
			IpPermissions: []*svcsdk.IpPermission{ipInput},
			// TODO: TagSpecs?
		}
		_, err = rm.sdkapi.RevokeSecurityGroupEgressWithContext(ctx, req)
		rm.metrics.RecordAPICall("DELETE", "RevokeSecurityGroupEgress", err)
		if err != nil {
			return err
		}
	}

	return err
}

func (rm *resourceManager) customUpdateSecurityGroup(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateSecurityGroup")
	defer exit(err)

	ko := desired.ko.DeepCopy()
	rm.setStatusDefaults(ko)

	if delta.DifferentAt("Spec.IngressRules") || delta.DifferentAt("Spec.EgressRules") {
		if err := rm.syncSGRules(ctx, desired, latest); err != nil {
			return nil, err
		}
		latest, err = rm.sdkFind(ctx, latest)
		if err != nil {
			return nil, err
		}
	}

	return latest, nil
}

// defaultEgressRule returns the egress rule that
// is created and associated with a security group by default
func defaultEgressRule() *svcapitypes.IPPermission {
	return &svcapitypes.IPPermission{
		IPRanges:   []*svcapitypes.IPRange{{CIDRIP: toStrPtr("0.0.0.0/0")}},
		FromPort:   toInt64Ptr(-1),
		IPProtocol: toStrPtr("-1"),
		ToPort:     toInt64Ptr(-1),
	}
}

// contains returns true if security group rule
// is found in the rule collection (all fields must match);
// otherwise, return false.
func contains(
	ruleCollection []*svcapitypes.IPPermission,
	rule *svcapitypes.IPPermission,
) bool {
	if ruleCollection == nil || rule == nil {
		return false
	}

	for _, r := range ruleCollection {
		delta := compareIPPermission(r, rule)
		if len(delta.Differences) == 0 {
			return true
		}
	}
	return false
}

func toBoolPtr(boolean bool) *bool {
	return &boolean
}

func toStrPtr(str string) *string {
	return &str
}

func toInt64Ptr(integer int64) *int64 {
	return &integer
}
