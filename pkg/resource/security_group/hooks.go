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

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	awserr "github.com/aws/aws-sdk-go/aws/awserr"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

// addRulesToSpec updates a resource's Spec EgressRules and IngressRules
// using data from a DescribeSecurityGroups response
func (rm *resourceManager) addRulesToSpec(
	ko *svcapitypes.SecurityGroup,
	resp *svcsdk.SecurityGroup,
) {
	// if there are no rules to add to Spec, then
	// set Spec rules to nil to align with latest state;
	// otherwise, data from an older version of the
	// resource will persist.
	ko.Spec.IngressRules = nil
	ko.Spec.EgressRules = nil

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

// getRules calls DescribeSecurityGroupRules
// and returns the  response data to populate a Security Group's
// Status.Rules
func (rm *resourceManager) getRules(
	ctx context.Context,
	res *resource,
) (rules []*svcapitypes.SecurityGroupRule, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getRules")
	defer exit(err)

	groupIDFilter := "group-id"
	input := &svcsdk.DescribeSecurityGroupRulesInput{
		Filters: []*svcsdk.Filter{
			{
				Name:   &groupIDFilter,
				Values: []*string{res.ko.Status.ID},
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
			rules = append(rules, rm.setResourceSecurityGroupRule(sgRule))
		}
		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		input.SetNextToken(*resp.NextToken)
	}
	if err != nil {
		return nil, err
	}

	return rules, nil
}

func (rm *resourceManager) requiredFieldsMissingForSGRule(
	r *resource,
) bool {
	return r.ko.Status.ID == nil
}

// syncSGRules analyzes desired and latest (if any)
// resources and executes API calls to Create/Delete
// rules in order to achieve desired state.
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
		if latest == nil || !containsRule(latest.ko.Spec.IngressRules, desiredIngress) {
			// a desired rule is not in the latest resource; therefore, create
			toAddIngress = append(toAddIngress, desiredIngress)
		}
	}
	for _, desiredEgress := range desired.ko.Spec.EgressRules {
		if latest == nil || !containsRule(latest.ko.Spec.EgressRules, desiredEgress) {
			toAddEgress = append(toAddEgress, desiredEgress)
		}
	}
	if latest != nil {
		for _, latestIngress := range latest.ko.Spec.IngressRules {
			if !containsRule(desired.ko.Spec.IngressRules, latestIngress) {
				// a rule is in latest resource, but not in desired resource; therefore, delete
				toDeleteIngress = append(toDeleteIngress, latestIngress)
			}
		}
		for _, latestEgress := range latest.ko.Spec.EgressRules {
			if !containsRule(desired.ko.Spec.EgressRules, latestEgress) {
				toDeleteEgress = append(toDeleteEgress, latestEgress)
			}
		}
	}

	// Delete before create for the following reasons:
	// - Updating a rule requires that it be removed before the updated version be added.
	// - If there is an error with adding new rules, it occurs after deletion of old ones;
	//   This is safer and closer to achieving desired resource state.
	if err = rm.deleteSecurityGroupRules(ctx, latest, toDeleteIngress, toDeleteEgress); err != nil {
		return err
	}
	if err = rm.createSecurityGroupRules(ctx, desired, toAddIngress, toAddEgress); err != nil {
		return err
	}

	return nil
}

// syncTags used to keep tags in sync by calling Create and Delete Tags API's
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

	resourceId := []*string{latest.ko.Status.ID}

	toAdd, toDelete := computeTagsDelta(
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)

	if len(toDelete) > 0 {
		rlog.Debug("removing tags from security group resource", "tags", toDelete)
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
		rlog.Debug("adding tags to security group resource", "tags", toAdd)
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
// Tags defined in the Spec to CreateSecurityGroupInput.TagSpecification
// and ensures the ResourceType is always set to 'security-group'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateSecurityGroupInput) {
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
		desiredTagSpecs.SetResourceType("security-group")
		desiredTagSpecs.SetTags(requestedTags)
		input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
	}
}

// createSecurityGroupRules takes a list of ingress and egress
// rules and attaches them to a SecurityGroup resource via
// AuthorizeSecurityGroup API calls
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

// deleteDefaultSecurityGroupRule deletes the default
// egress rule that is attached to a SecurityGroup upon creation.
// The rule is set explicitly (without helpers); otherwise, the quotes
// around IpProtocol value will not be set properly
func (rm *resourceManager) deleteDefaultSecurityGroupRule(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteDefaultSecurityGroupRule")
	defer exit(err)

	ipRange := &svcsdk.IpRange{
		CidrIp: toStrPtr("0.0.0.0/0"),
	}
	input := &svcsdk.IpPermission{
		FromPort:   toInt64Ptr(-1),
		ToPort:     toInt64Ptr(-1),
		IpProtocol: toStrPtr("-1"),
		IpRanges:   []*svcsdk.IpRange{ipRange},
	}
	req := &svcsdk.RevokeSecurityGroupEgressInput{
		GroupId:       r.ko.Status.ID,
		IpPermissions: []*svcsdk.IpPermission{input},
	}
	_, err = rm.sdkapi.RevokeSecurityGroupEgressWithContext(ctx, req)
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

	return err
}

// deleteSecurityGroupRules takes a list of ingress and egress
// rules and removes them from a SecurityGroup resource via
// RevokeSecurityGroup API calls
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

// customUpdateSecurityGroup updates IngressRules and/or
// EgressRules, if a delta be detected between resources.
func (rm *resourceManager) customUpdateSecurityGroup(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateSecurityGroup")
	defer exit(err)

	// Default `updated` to `desired` because it is likely
	// EC2 `modify` APIs do NOT return output, only errors.
	// If the `modify` calls (i.e. `sync`) do NOT return
	// an error, then the update was successful and desired.Spec
	// (now updated.Spec) reflects the latest resource state.
	updated = rm.concreteResource(desired.DeepCopy())

	if delta.DifferentAt("Spec.IngressRules") || delta.DifferentAt("Spec.EgressRules") {
		if err := rm.syncSGRules(ctx, desired, latest); err != nil {
			return nil, err
		}
		// A ReadOne call for SecurityGroup Rules (NOT SecurityGroups)
		// is made to refresh Status.Rules with the recently-updated
		// data from the above `sync` call
		if rules, err := rm.getRules(ctx, desired); err != nil {
			return nil, err
		} else {
			updated.ko.Status.Rules = rules
		}
	}

	if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	return updated, nil
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

// containsRule returns true if security group rule
// is found in the rule collection (all fields must match);
// otherwise, return false.
func containsRule(
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

func toStrPtr(str string) *string {
	return &str
}

func toInt64Ptr(integer int64) *int64 {
	return &integer
}
