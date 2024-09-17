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
	"strconv"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/samber/lo"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
)

var DefaultRuleNumber = 32767
var TypeSubnet = "SubnetID"
var TypeNaclAssocId = "NetworkACLAssociationID"

func (rm *resourceManager) customUpdateNetworkAcl(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateNetworkAcl")
	defer func(err error) { exit(err) }(err)

	updated = rm.concreteResource(desired.DeepCopy())

	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.ID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.Entries") {
		if err := rm.syncEntries(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.Associations") {
		if err := rm.syncAssociation(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	latestResource, err := rm.sdkFind(ctx, desired)
	if err != nil {
		return nil, err
	}

	// The ec2 API can sometimes sort the entries in a different order than the
	// ones we have in the desired spec. Hence, we need to conserve the order of
	// entries in the desired spec.
	updated.ko.Status = latestResource.ko.Status

	return updated, nil
}

func (rm *resourceManager) requiredFieldsMissingForCreateNetworkAcl(
	r *resource,
) bool {
	return r.ko.Status.ID == nil
}

func (rm *resourceManager) createEntries(
	ctx context.Context,
	r *resource,
) error {
	if r.ko.Spec.Entries != nil {
		if err := rm.syncEntries(ctx, r, nil); err != nil {
			return err
		}
	}
	return nil
}

func (rm *resourceManager) createAssociation(
	ctx context.Context,
	r *resource,
) error {
	if r.ko.Spec.Associations != nil {
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

	latestAssociations := make(map[string]string)
	desiredAssociations := make(map[string]string)
	associationidSubnet := make(map[string]string)

	if latest != nil {
		for _, association := range latest.ko.Spec.Associations {
			if association.NetworkACLAssociationID != nil {
				latestAssociations[*association.NetworkACLAssociationID] = TypeNaclAssocId
			}
			if association.SubnetID != nil {
				latestAssociations[*association.SubnetID] = TypeSubnet
			}
			if association.NetworkACLAssociationID != nil && association.SubnetID != nil {
				associationidSubnet[*association.SubnetID] = *association.NetworkACLAssociationID
			}
		}
	}
	if desired != nil {
		for _, association := range desired.ko.Spec.Associations {
			if association.NetworkACLAssociationID != nil {
				desiredAssociations[*association.NetworkACLAssociationID] = TypeNaclAssocId
			}
			if association.SubnetID != nil {
				desiredAssociations[*association.SubnetID] = TypeSubnet
			}
		}
	}
	// Determining the associations to be added and deleted by comparing associations of latest and desired.
	toAdd := lo.OmitByKeys(desiredAssociations, lo.Keys(latestAssociations))
	includedSubnets := lo.PickByKeys(associationidSubnet, lo.Keys(desiredAssociations))
	associations_diff := lo.OmitByKeys(latestAssociations, lo.Keys(desiredAssociations))
	toDelete := lo.OmitByKeys(associations_diff, lo.Values(includedSubnets))

	upsertErr := rm.upsertNewAssociations(ctx, desired, latest, toAdd)
	if upsertErr != nil {
		return upsertErr
	}
	deletErr := rm.deleteOldAssociations(ctx, desired, latest, toDelete)
	if deletErr != nil {
		return deletErr
	}
	return nil

}

func (rm *resourceManager) deleteOldAssociations(
	ctx context.Context,
	desired *resource,
	latest *resource,
	toDelete map[string]string,
) (err error) {
	var vpcID *string
	var aclID *string
	if desired != nil {
		vpcID = desired.ko.Spec.VPCID
		aclID = desired.ko.Status.ID
	} else {
		vpcID = latest.ko.Spec.VPCID
		aclID = latest.ko.Status.ID
	}
	naclList := &svcsdk.DescribeNetworkAclsInput{
		Filters: []*svcsdk.Filter{
			{
				Name:   lo.ToPtr("default"),
				Values: []*string{lo.ToPtr("true")},
			},
			{
				Name:   lo.ToPtr("vpc-id"),
				Values: []*string{vpcID},
			},
		},
	}
	defaultNacl, err := rm.sdkapi.DescribeNetworkAclsWithContext(ctx, naclList)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeNetworkAcls", err)
	if err != nil {
		return err
	}
	if defaultNacl == nil {
		return errors.New("could not determine default Nacl for the given VPC")
	}
	for rid, rtype := range toDelete {
		input := &svcsdk.ReplaceNetworkAclAssociationInput{}

		if rtype == TypeNaclAssocId {
			input.AssociationId = &rid
			input.NetworkAclId = defaultNacl.NetworkAcls[0].NetworkAclId
			_, err = rm.sdkapi.ReplaceNetworkAclAssociationWithContext(ctx, input)
			rm.metrics.RecordAPICall("UPDATE", "ReplaceNetworkAclAssociation", err)
			if err != nil {
				return err
			}
		}
		if rtype == TypeSubnet {
			dnaInput := &svcsdk.DescribeNetworkAclsInput{
				Filters: []*svcsdk.Filter{
					{
						Name:   lo.ToPtr("network-acl-id"),
						Values: []*string{aclID},
					},
					{
						Name:   lo.ToPtr("association.subnet-id"),
						Values: []*string{&rid},
					},
				},
			}
			dnaOutput, err := rm.sdkapi.DescribeNetworkAclsWithContext(ctx, dnaInput)
			rm.metrics.RecordAPICall("READ_MANY", "DescribeNetworkAcls", err)
			if err != nil {
				return err
			}
			if len(dnaOutput.NetworkAcls) != 1 {
				return errors.New("unexpected output from DescribeNetworkAcls for the given subnet")
			}
			input.NetworkAclId = defaultNacl.NetworkAcls[0].NetworkAclId
			for _, association := range dnaOutput.NetworkAcls[0].Associations {
				if *association.SubnetId == rid {
					input.AssociationId = association.NetworkAclAssociationId
					break
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

func (rm *resourceManager) upsertNewAssociations(
	ctx context.Context,
	desired *resource,
	latest *resource,
	toAdd map[string]string,
) (err error) {
	for rid, rtype := range toAdd {
		input := &svcsdk.ReplaceNetworkAclAssociationInput{}

		if rtype == TypeNaclAssocId {
			input.AssociationId = &rid
			input.NetworkAclId = latest.ko.Status.ID
			_, err = rm.sdkapi.ReplaceNetworkAclAssociationWithContext(ctx, input)
			rm.metrics.RecordAPICall("UPDATE", "ReplaceNetworkAclAssociation", err)
			if err != nil {
				return err
			}
		}
		if rtype == TypeSubnet {
			dnaInput := &svcsdk.DescribeNetworkAclsInput{
				Filters: []*svcsdk.Filter{
					{
						Name:   lo.ToPtr("association.subnet-id"),
						Values: []*string{&rid},
					},
				},
			}
			dnaOutput, err := rm.sdkapi.DescribeNetworkAclsWithContext(ctx, dnaInput)
			rm.metrics.RecordAPICall("READ_MANY", "DescribeNetworkAcls", err)
			if err != nil {
				return err
			}
			input.NetworkAclId = desired.ko.Status.ID
			if len(dnaOutput.NetworkAcls) != 1 {
				return errors.New("unexpected output from DescribeNetworkAcls for the given subnet")
			} else {
				for _, association := range dnaOutput.NetworkAcls[0].Associations {
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

func (rm *resourceManager) syncEntries(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncEntries")
	defer func(err error) { exit(err) }(err)

	toAdd := []*svcapitypes.NetworkACLEntry{}
	toDelete := []*svcapitypes.NetworkACLEntry{}
	toUpdate := []*svcapitypes.NetworkACLEntry{}

	uniqEntries := lo.UniqBy(desired.ko.Spec.Entries, func(entry *svcapitypes.NetworkACLEntry) string {
		return strconv.FormatBool(*entry.Egress) + strconv.Itoa(int(*entry.RuleNumber))
	})

	if len(desired.ko.Spec.Entries) != len(uniqEntries) {
		return errors.New("multple rules with the same rule number and Egress in the desired spec")
	}

	for _, desiredEntry := range desired.ko.Spec.Entries {

		if *((*desiredEntry).RuleNumber) == int64(DefaultRuleNumber) {
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
			if *((*latestEntry).RuleNumber) == int64(DefaultRuleNumber) {
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
	rm.metrics.RecordAPICall("UPDATE", "CreateNetworkAclEntry", err)
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
	rm.metrics.RecordAPICall("UPDATE", "ReplaceNetworkAclEntry", err)
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
	rm.metrics.RecordAPICall("UPDATE", "DeleteNetworkAclEntry", err)
	return err
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
