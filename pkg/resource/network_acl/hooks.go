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

package network_acl

import (
	"context"
	"errors"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/samber/lo"
)

var defaultRuleNumber = 32767

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

	resourceId := []*string{latest.ko.Status.ID}

	desiredTags := ToACKTags(desired.ko.Spec.Tags)
	latestTags := ToACKTags(latest.ko.Spec.Tags)

	added, _, removed := ackcompare.GetTagsDifference(latestTags, desiredTags)

	toAdd := FromACKTags(added)
	toDelete := FromACKTags(removed)

	if len(toDelete) > 0 {
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

// // sdkTags converts *svcapitypes.Tag array to a *svcsdk.Tag array
func (rm *resourceManager) sdkTags(

	tags []*svcapitypes.Tag,

) (sdktags []*svcsdk.Tag) {

	for _, i := range tags {
		sdktag := rm.newTag(*i)
		sdktags = append(sdktags, sdktag)
	}

	return sdktags
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

func (rm *resourceManager) customUpdateNetworkAcl(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateNetworkAcl")
	defer exit(err)

	// Default `updated` to `desired` because it is likely
	// EC2 `modify` APIs do NOT return output, only errors.
	// If the `modify` calls (i.e. `sync`) do NOT return
	// an error, then the update was successful and desired.Spec
	// (now updated.Spec) reflects the latest resource state.
	updated = rm.concreteResource(desired.DeepCopy())

	if delta.DifferentAt("Spec.Entries") {
		if err := rm.syncRules(ctx, desired, latest); err != nil {
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
		updated.ko.Spec.Tags = desired.ko.Spec.Tags
	}

	if delta.DifferentAt("Spec.Associations") {
		if err := rm.syncAssociation(ctx, desired, latest); err != nil {
			return nil, err
		}
		updated.ko.Spec.Associations = desired.ko.Spec.Associations
	}

	return updated, nil
}

func (rm *resourceManager) requiredFieldsMissingForCreateNetworkAcl(
	r *resource,
) bool {
	return r.ko.Status.ID == nil
}

func (rm *resourceManager) createRules(
	ctx context.Context,
	r *resource,
) error {
	if r.ko.Spec.Entries != nil {
		if err := rm.syncRules(ctx, r, nil); err != nil {
			return err
		}
	}
	return nil
}

func (rm *resourceManager) createAssociation(
	ctx context.Context,
	r *resource,
) error {
	if r.ko.Spec.Entries != nil {
		if err := rm.syncAssociation(ctx, r, nil); err != nil {
			return err
		}
	}
	return nil
}

func (rm *resourceManager) syncAssociation(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncAssociation")
	defer exit(err)

	latest_associations := make(map[string]string)
	desired_associations := make(map[string]string)
	associationid_subnet := make(map[string]string)

	if latest != nil {
		for _, association := range latest.ko.Spec.Associations {
			if association.NetworkACLAssociationID != nil {
				latest_associations[*association.NetworkACLAssociationID] = "NetworkACLAssociationID"
			}
			if association.SubnetID != nil {
				latest_associations[*association.SubnetID] = "SubnetID"
			}
			if association.NetworkACLAssociationID != nil && association.SubnetID != nil {
				associationid_subnet[*association.SubnetID] = *association.NetworkACLAssociationID
			}
		}
	}
	if desired != nil {
		for _, association := range desired.ko.Spec.Associations {
			if association.NetworkACLAssociationID != nil {
				desired_associations[*association.NetworkACLAssociationID] = "NetworkACLAssociationID"
			}
			if association.SubnetID != nil {
				desired_associations[*association.SubnetID] = "SubnetID"
			}
		}
	}
	to_Add := lo.OmitByKeys(desired_associations, lo.Keys(latest_associations))
	included_subnets := lo.PickByKeys(associationid_subnet, lo.Keys(desired_associations))
	y := lo.OmitByKeys(latest_associations, lo.Keys(desired_associations))
	to_Delete := lo.OmitByKeys(y, lo.Values(included_subnets))

	for rid, rtype := range to_Add {
		input := &svcsdk.ReplaceNetworkAclAssociationInput{}

		if rtype == "NetworkACLAssociationID" {
			input.AssociationId = &rid
			input.NetworkAclId = latest.ko.Status.ID
			_, err = rm.sdkapi.ReplaceNetworkAclAssociationWithContext(ctx, input)
			rm.metrics.RecordAPICall("UPDATE", "ReplaceNetworkAclAssociation", err)
			if err != nil {
				return err
			}
		}
		if rtype == "SubnetID" {
			dna_input := &svcsdk.DescribeNetworkAclsInput{
				Filters: []*svcsdk.Filter{
					{
						Name:   toStrPtr("association.subnet-id"),
						Values: []*string{&rid},
					},
				},
			}
			dna_output, err := rm.sdkapi.DescribeNetworkAclsWithContext(ctx, dna_input)
			rm.metrics.RecordAPICall("DESCRIBE", "DescribeNetworkAcls", err)
			if err != nil {
				return err
			}
			input.NetworkAclId = desired.ko.Status.ID
			if len(dna_output.NetworkAcls) != 1 {
				return errors.New("unexpected output from describenetworkacls for the given subnet")
			} else {
				for _, association := range dna_output.NetworkAcls[0].Associations {
					if *association.SubnetId == rid {
						input.AssociationId = association.NetworkAclAssociationId
						break
					}
				}
			}
			_, err = rm.sdkapi.ReplaceNetworkAclAssociationWithContext(ctx, input)
			rm.metrics.RecordAPICall("UPDATE", "ReplaceNetworkAclAssociation", err)
			if err != nil {
				return err
			}
		}
	}

	x := &svcsdk.DescribeNetworkAclsInput{
		Filters: []*svcsdk.Filter{
			{
				Name:   toStrPtr("default"),
				Values: []*string{toStrPtr("true")},
			},
			{
				Name:   toStrPtr("vpc-id"),
				Values: []*string{desired.ko.Spec.VPCID},
			},
		},
	}
	default_nacl, err := rm.sdkapi.DescribeNetworkAclsWithContext(ctx, x)
	rm.metrics.RecordAPICall("DESCRIBE", "DescribeNetworkAcls", err)
	if err != nil {
		return err
	}
	if default_nacl == nil {
		return errors.New("could not determine")
	}
	for rid, rtype := range to_Delete {
		input := &svcsdk.ReplaceNetworkAclAssociationInput{}

		if rtype == "NetworkACLAssociationID" {
			input.AssociationId = &rid
			input.NetworkAclId = default_nacl.NetworkAcls[0].NetworkAclId
			_, err = rm.sdkapi.ReplaceNetworkAclAssociationWithContext(ctx, input)
			rm.metrics.RecordAPICall("UPDATE", "ReplaceNetworkAclAssociation", err)
			if err != nil {
				return err
			}
		}
		if rtype == "SubnetID" {
			dna_input := &svcsdk.DescribeNetworkAclsInput{
				Filters: []*svcsdk.Filter{
					{
						Name:   toStrPtr("network-acl-id"),
						Values: []*string{desired.ko.Status.ID},
					},
					{
						Name:   toStrPtr("association.subnet-id"),
						Values: []*string{&rid},
					},
				},
			}
			dna_output, err := rm.sdkapi.DescribeNetworkAclsWithContext(ctx, dna_input)
			rm.metrics.RecordAPICall("DESCRIBE", "DescribeNetworkAcls", err)
			if err != nil {
				return err
			}
			input.NetworkAclId = default_nacl.NetworkAcls[0].NetworkAclId
			if len(dna_output.NetworkAcls) != 1 {
				return errors.New("unexpected output from describenetworkacls for the given subnet")
			} else {
				for _, association := range dna_output.NetworkAcls[0].Associations {
					if *association.SubnetId == rid {
						input.AssociationId = association.NetworkAclAssociationId
						break
					}
				}
			}
			_, err = rm.sdkapi.ReplaceNetworkAclAssociationWithContext(ctx, input)
			rm.metrics.RecordAPICall("UPDATE", "ReplaceNetworkAclAssociation", err)
			if err != nil {
				return err
			}
		}
	}
	return nil

}

func toStrPtr(str string) *string {
	return &str
}

func (rm *resourceManager) syncRules(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncRules")
	defer exit(err)
	toAdd := []*svcapitypes.NetworkACLEntry{}
	toDelete := []*svcapitypes.NetworkACLEntry{}
	toUpdate := []*svcapitypes.NetworkACLEntry{}

	ruleNumber := make(map[int]bool)

	for _, entry := range desired.ko.Spec.Entries {
		if value, present := ruleNumber[int(*entry.RuleNumber)]; present {
			if value == (*entry.Egress) {
				return errors.New("multple rules with the same rule number and Egress in the desired spec")
			}
		} else {
			ruleNumber[int(*entry.RuleNumber)] = (*entry.Egress)
		}
	}

	for _, desiredEntry := range desired.ko.Spec.Entries {

		if *((*desiredEntry).RuleNumber) == int64(defaultRuleNumber) {
			// no-op for default route
			continue
		}

		if latest != nil && !containsEntry(latest.ko.Spec.Entries, desiredEntry) {
			// a desired rule is not in the latest resource; therefore, create
			toAdd = append(toAdd, desiredEntry)
		}
	}

	if latest != nil {
		for _, latestEntry := range latest.ko.Spec.Entries {
			if *((*latestEntry).RuleNumber) == int64(defaultRuleNumber) {
				// no-op for default route
				continue
			}
			if !containsEntry(desired.ko.Spec.Entries, latestEntry) {
				// entry is in latest resource, but not in desired resource; therefore, delete
				toDelete = append(toDelete, latestEntry)
			}
		}
	}

	// Checking latest for entries which has the same rule number as the entry in toAdd
	for index, entry := range toAdd {
		for _, latestEntry := range latest.ko.Spec.Entries {
			if *(entry.RuleNumber) == *(latestEntry.RuleNumber) && *(entry.Egress) == *(latestEntry.Egress) {
				toUpdate = append(toUpdate, entry)
				toAdd[index] = nil
			}
		}
	}

	for _, entry := range toAdd {
		if entry == nil {
			continue
		}

		if err = rm.createEntry(ctx, desired, *entry); err != nil {
			return err
		}
	}

	for _, entry := range toDelete {
		if err = rm.deleteEntry(ctx, latest, *entry); err != nil {
			return err
		}
	}

	for _, entry := range toUpdate {
		if err = rm.updateEntry(ctx, latest, *entry); err != nil {
			return err
		}
	}

	return nil

}

// containsRule returns true if entry
// is found in the entry collection (all fields must match);
// otherwise, return false.
func containsEntry(
	entryCollection []*svcapitypes.NetworkACLEntry,
	entry *svcapitypes.NetworkACLEntry,
) bool {
	if entryCollection == nil || entry == nil {
		return false
	}

	for _, e := range entryCollection {
		delta := compareNetworkACLEntry(e, entry)

		if len(delta.Differences) == 0 {
			return true
		}
	}
	return false
}
func (rm *resourceManager) createEntry(
	ctx context.Context,
	r *resource,
	c svcapitypes.NetworkACLEntry,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.createEntry")
	defer exit(err)

	if r == nil {
		return nil
	}

	res := &svcsdk.CreateNetworkAclEntryInput{}
	if c.CIDRBlock != nil {
		res.SetCidrBlock(*c.CIDRBlock)
	}
	if c.Egress != nil {
		res.SetEgress(*c.Egress)
	}
	if c.ICMPTypeCode != nil {
		resf3 := &svcsdk.IcmpTypeCode{}
		if c.ICMPTypeCode.Code != nil {
			resf3.SetCode(*c.ICMPTypeCode.Code)
		}
		if c.ICMPTypeCode.Type != nil {
			resf3.SetType(*c.ICMPTypeCode.Type)
		}
		res.SetIcmpTypeCode(resf3)
	}
	if c.IPv6CIDRBlock != nil {
		res.SetIpv6CidrBlock(*c.IPv6CIDRBlock)
	}
	if c.PortRange != nil {
		resf6 := &svcsdk.PortRange{}
		if c.PortRange.From != nil {
			resf6.SetFrom(*c.PortRange.From)
		}
		if c.PortRange.To != nil {
			resf6.SetTo(*c.PortRange.To)
		}
		res.SetPortRange(resf6)
	}
	if c.Protocol != nil {
		res.SetProtocol(*c.Protocol)
	}
	if c.RuleAction != nil {
		res.SetRuleAction(*c.RuleAction)
	}
	if c.RuleNumber != nil {
		res.SetRuleNumber(*c.RuleNumber)
	}

	res.NetworkAclId = r.ko.Status.ID
	_, err = rm.sdkapi.CreateNetworkAclEntryWithContext(ctx, res)
	rm.metrics.RecordAPICall("CREATE", "CreateNetworkAclEntry", err)
	return err
}

func (rm *resourceManager) updateEntry(
	ctx context.Context,
	r *resource,
	c svcapitypes.NetworkACLEntry,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.updateEntry")
	defer exit(err)

	if r == nil {
		return nil
	}

	res := &svcsdk.ReplaceNetworkAclEntryInput{}
	if c.CIDRBlock != nil {
		res.SetCidrBlock(*c.CIDRBlock)
	}
	if c.Egress != nil {
		res.SetEgress(*c.Egress)
	}
	if c.ICMPTypeCode != nil {
		resf3 := &svcsdk.IcmpTypeCode{}
		if c.ICMPTypeCode.Code != nil {
			resf3.SetCode(*c.ICMPTypeCode.Code)
		}
		if c.ICMPTypeCode.Type != nil {
			resf3.SetType(*c.ICMPTypeCode.Type)
		}
		res.SetIcmpTypeCode(resf3)
	}
	if c.IPv6CIDRBlock != nil {
		res.SetIpv6CidrBlock(*c.IPv6CIDRBlock)
	}
	if c.PortRange != nil {
		resf6 := &svcsdk.PortRange{}
		if c.PortRange.From != nil {
			resf6.SetFrom(*c.PortRange.From)
		}
		if c.PortRange.To != nil {
			resf6.SetTo(*c.PortRange.To)
		}
		res.SetPortRange(resf6)
	}
	if c.Protocol != nil {
		res.SetProtocol(*c.Protocol)
	}
	if c.RuleAction != nil {
		res.SetRuleAction(*c.RuleAction)
	}
	if c.RuleNumber != nil {
		res.SetRuleNumber(*c.RuleNumber)
	}

	res.NetworkAclId = r.ko.Status.ID
	_, err = rm.sdkapi.ReplaceNetworkAclEntryWithContext(ctx, res)
	rm.metrics.RecordAPICall("REPLACE", "ReplaceNetworkAclEntry", err)
	return err
}

func (rm *resourceManager) deleteEntry(
	ctx context.Context,
	r *resource,
	c svcapitypes.NetworkACLEntry,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.createEntry")
	defer exit(err)

	if r == nil {
		return nil
	}

	res := &svcsdk.DeleteNetworkAclEntryInput{}
	if c.Egress != nil {
		res.SetEgress(*c.Egress)
	}
	if c.RuleNumber != nil {
		res.SetRuleNumber(*c.RuleNumber)
	}

	res.NetworkAclId = r.ko.Status.ID
	_, err = rm.sdkapi.DeleteNetworkAclEntryWithContext(ctx, res)
	rm.metrics.RecordAPICall("DELETE", "DeleteNetworkAclEntry", err)
	return err
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
// Tags defined in the Spec to CreateNetworkAclInput.TagSpecification
// and ensures the ResourceType is always set to 'network-acl'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateNetworkAclInput) {
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
		desiredTagSpecs.SetResourceType("network-acl")
		desiredTagSpecs.SetTags(requestedTags)
		input.TagSpecifications = []*svcsdk.TagSpecification{&desiredTagSpecs}
	}
}
