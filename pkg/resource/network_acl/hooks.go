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
	"strings"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
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
	desired *resource,
) error {
	if desired.ko.Spec.Entries != nil {
		if err := rm.syncEntries(ctx, desired, nil); err != nil {
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
		Filters: []svcsdktypes.Filter{
			{
				Name:   lo.ToPtr("default"),
				Values: []string{"true"},
			},
			{
				Name:   lo.ToPtr("vpc-id"),
				Values: []string{*vpcID},
			},
		},
	}
	defaultNacl, err := rm.sdkapi.DescribeNetworkAcls(ctx, naclList)
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
			_, err = rm.sdkapi.ReplaceNetworkAclAssociation(ctx, input)
			rm.metrics.RecordAPICall("UPDATE", "ReplaceNetworkAclAssociation", err)
			if err != nil {
				return err
			}
		}
		if rtype == TypeSubnet {
			dnaInput := &svcsdk.DescribeNetworkAclsInput{
				Filters: []svcsdktypes.Filter{
					{
						Name:   lo.ToPtr("network-acl-id"),
						Values: []string{*aclID},
					},
					{
						Name:   lo.ToPtr("association.subnet-id"),
						Values: []string{rid},
					},
				},
			}
			dnaOutput, err := rm.sdkapi.DescribeNetworkAcls(ctx, dnaInput)
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

			_, err = rm.sdkapi.ReplaceNetworkAclAssociation(ctx, input)
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
			_, err = rm.sdkapi.ReplaceNetworkAclAssociation(ctx, input)
			rm.metrics.RecordAPICall("UPDATE", "ReplaceNetworkAclAssociation", err)
			if err != nil {
				return err
			}
		}
		if rtype == TypeSubnet {
			dnaInput := &svcsdk.DescribeNetworkAclsInput{
				Filters: []svcsdktypes.Filter{
					{
						Name:   lo.ToPtr("association.subnet-id"),
						Values: []string{rid},
					},
				},
			}
			dnaOutput, err := rm.sdkapi.DescribeNetworkAcls(ctx, dnaInput)
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
			_, err = rm.sdkapi.ReplaceNetworkAclAssociation(ctx, input)
			rm.metrics.RecordAPICall("UPDATE", "ReplaceNetworkAclAssociation", err)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// The function filters out AWS-managed default rules (rule #32767) from both desired
// and latest states to prevent interference with AWS's automatic management of these
// rules and to maintain GitOps compatibility.
//
// Default rules behavior:
//   - If default rules (rule #32767) are explicitly defined in the spec, they will remain in the spec
//   - If default rules are not defined in the spec, they will be ignored during sync
//     operations and not be added to the spec
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

	// Filter out AWS default rules (rule #32767) from desired entries to ensure
	// they don't interfere with GitOps workflows. These rules are automatically
	// managed by AWS and should not be included in the desired state.
	filteredDesiredEntries := []*svcapitypes.NetworkACLEntry{}
	for _, entry := range desired.ko.Spec.Entries {
		if entry.RuleNumber != nil && *entry.RuleNumber == int64(DefaultRuleNumber) {
			continue
		}
		filteredDesiredEntries = append(filteredDesiredEntries, entry)
	}
	desired.ko.Spec.Entries = filteredDesiredEntries

	// Check for duplicate rule numbers within the same direction (egress/ingress)
	uniqEntries := lo.UniqBy(desired.ko.Spec.Entries, func(entry *svcapitypes.NetworkACLEntry) string {
		return strconv.FormatBool(*entry.Egress) + strconv.Itoa(int(*entry.RuleNumber))
	})

	if len(desired.ko.Spec.Entries) != len(uniqEntries) {
		return errors.New("multple rules with the same rule number and Egress in the desired spec")
	}

	// Identify new entries that need to be created
	for _, desiredEntry := range desired.ko.Spec.Entries {
		if latest != nil && !containsEntry(latest.ko.Spec.Entries, desiredEntry) {
			toAdd = append(toAdd, desiredEntry)
		}
	}
	if latest != nil {
		// Filter out AWS default rules from latest entries before comparison
		// to ensure consistent state management between desired and actual
		filteredLatestEntries := []*svcapitypes.NetworkACLEntry{}
		for _, entry := range latest.ko.Spec.Entries {
			if entry.RuleNumber != nil && *entry.RuleNumber == int64(DefaultRuleNumber) {
				continue
			}
			filteredLatestEntries = append(filteredLatestEntries, entry)
		}

		// Identify entries that need to be deleted (exist in latest but not in desired)
		for _, latestEntry := range filteredLatestEntries {
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

	// During create latest is nil, just add the entries when sync is called via createEntries
	if latest == nil {
		toAdd = append(toAdd, desired.ko.Spec.Entries...)
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
		if !delta.DifferentExcept("NetworkACLEntry.RuleAction") {
			// Case insensitive comparison for RuleAction
			if compareNetworkACLEntryAtRuleAction(e, entry) {
				return true
			}
		}
	}
	return false
}

func compareNetworkACLEntryAtRuleAction(entry1 *svcapitypes.NetworkACLEntry, entry2 *svcapitypes.NetworkACLEntry) bool {
	if entry1.RuleAction == nil && entry2.RuleAction == nil {
		return true
	}

	if entry1.RuleAction == nil || entry2.RuleAction == nil {
		return false
	}

	return strings.EqualFold(*entry1.RuleAction, *entry2.RuleAction)
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
		res.CidrBlock = c.CIDRBlock
	}
	if c.Egress != nil {
		res.Egress = c.Egress
	}
	if c.ICMPTypeCode != nil {
		resf3 := &svcsdktypes.IcmpTypeCode{}
		if c.ICMPTypeCode.Code != nil {
			resf3.Code = aws.Int32(int32(*c.ICMPTypeCode.Code))
		}
		if c.ICMPTypeCode.Type != nil {
			resf3.Type = aws.Int32(int32(*c.ICMPTypeCode.Type))
		}
		res.IcmpTypeCode = resf3
	}
	if c.IPv6CIDRBlock != nil {
		res.Ipv6CidrBlock = c.IPv6CIDRBlock
	}
	if c.PortRange != nil {
		resf6 := &svcsdktypes.PortRange{}
		if c.PortRange.From != nil {
			resf6.From = aws.Int32(int32(*c.PortRange.From))
		}
		if c.PortRange.To != nil {
			resf6.To = aws.Int32(int32(*c.PortRange.To))
		}
		res.PortRange = resf6
	}
	if c.Protocol != nil {
		res.Protocol = c.Protocol
	}
	if c.RuleAction != nil {
		res.RuleAction = svcsdktypes.RuleAction(*c.RuleAction)
	}
	if c.RuleNumber != nil {
		res.RuleNumber = aws.Int32(int32(*c.RuleNumber))
	}

	res.NetworkAclId = r.ko.Status.ID
	_, err = rm.sdkapi.CreateNetworkAclEntry(ctx, res)
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
		res.CidrBlock = c.CIDRBlock
	}
	if c.Egress != nil {
		res.Egress = c.Egress
	}
	if c.ICMPTypeCode != nil {
		resf3 := &svcsdktypes.IcmpTypeCode{}
		if c.ICMPTypeCode.Code != nil {
			resf3.Code = aws.Int32(int32(*c.ICMPTypeCode.Code))
		}
		if c.ICMPTypeCode.Type != nil {
			resf3.Type = aws.Int32(int32(*c.ICMPTypeCode.Type))
		}
		res.IcmpTypeCode = resf3
	}
	if c.IPv6CIDRBlock != nil {
		res.Ipv6CidrBlock = c.IPv6CIDRBlock
	}
	if c.PortRange != nil {
		resf6 := &svcsdktypes.PortRange{}
		if c.PortRange.From != nil {
			resf6.From = aws.Int32(int32(*c.PortRange.From))
		}
		if c.PortRange.To != nil {
			resf6.To = aws.Int32(int32(*c.PortRange.To))
		}
		res.PortRange = resf6
	}
	if c.Protocol != nil {
		res.Protocol = c.Protocol
	}
	if c.RuleAction != nil {
		res.RuleAction = svcsdktypes.RuleAction(*c.RuleAction)
	}
	if c.RuleNumber != nil {
		res.RuleNumber = aws.Int32(int32(*c.RuleNumber))
	}

	res.NetworkAclId = r.ko.Status.ID
	_, err = rm.sdkapi.ReplaceNetworkAclEntry(ctx, res)
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
		res.Egress = c.Egress
	}
	if c.RuleNumber != nil {
		res.RuleNumber = aws.Int32(int32(*c.RuleNumber))
	}

	res.NetworkAclId = r.ko.Status.ID
	_, err = rm.sdkapi.DeleteNetworkAclEntry(ctx, res)
	rm.metrics.RecordAPICall("UPDATE", "DeleteNetworkAclEntry", err)
	return err
}

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateNetworkAclInput.TagSpecification
// and ensures the ResourceType is always set to 'network-acl'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateNetworkAclInput) {
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
		desiredTagSpecs.ResourceType = "network-acl"
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}
