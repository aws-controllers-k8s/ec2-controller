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
	//svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	//ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	//ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"context"
	"errors"
	"fmt"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
)

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

	resourceId := []*string{latest.ko.Status.NetworkACLID}

	toAdd, toDelete := computeTagsDelta(
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)

	fmt.Println("toaddtags", toAdd)
	fmt.Println("toremovetags", toDelete)

	if len(toDelete) > 0 {
		rlog.Debug("removing tags from NetworkACL resource", "tags", toDelete)
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
		rlog.Debug("adding tags to NetworkACL resource", "tags", toAdd)
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
		// A ReadOne call is made to refresh
		// with the recently-updated data from the above `sync` call
		updated, err = rm.sdkFind(ctx, desired)
		if err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.Tags") {
		fmt.Println("DifferentAtTag")
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

func (rm *resourceManager) requiredFieldsMissingForCreateNetworkAcl(
	r *resource,
) bool {
	return r.ko.Status.NetworkACLID == nil
}

func (rm *resourceManager) createRules(
	ctx context.Context,
	r *resource,
) error {
	fmt.Println("Point3")
	if err := rm.syncRules(ctx, r, nil); err != nil {
		return err
	}
	return nil
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
	fmt.Println("Point4")

	ruleNumber := make(map[int]bool)

	for _, entry := range desired.ko.Spec.Entries {
		if value, present := ruleNumber[int(*entry.RuleNumber)]; present {
			if value == (*entry.Egress) {
				fmt.Println("myerr:", "multiple rules with same rule no")
				return errors.New("multple rules with the same rule number and Egress in the desired spec")
			}
		} else {
			ruleNumber[int(*entry.RuleNumber)] = (*entry.Egress)
		}
	}

	for _, desiredEntry := range desired.ko.Spec.Entries {

		if *((*desiredEntry).RuleNumber) == int64(32767) {
			// no-op for default route
			fmt.Println("Inside first noop")
			continue
		}
		fmt.Println("RuleNumber", (*desiredEntry).RuleNumber)

		// if latestEntry := getMatchingEntry(desiredEntry, latest); latestEntry != nil {
		// 	delta := compareNetworkACLEntry(desiredEntry, latestEntry)
		// 	if len(delta.Differences) > 0 {
		// 		toDelete = append(toDelete, latestEntry)
		// 		toAdd = append(toAdd, desiredEntry)
		// 	}
		// } else {
		// 	toAdd = append(toAdd, desiredEntry)
		// }
		// if latest != nil && containsEntryRuleNumber(latest.ko.Spec.Entries, desiredEntry) {
		// 	toUpdate = append(toUpdate, desiredEntry)
		// 	continue
		// }

		if latest != nil && !containsEntry(latest.ko.Spec.Entries, desiredEntry) {
			// a desired rule is not in the latest resource; therefore, create
			toAdd = append(toAdd, desiredEntry)
		}
	}

	if latest != nil {
		for _, latestEntry := range latest.ko.Spec.Entries {
			if *((*latestEntry).RuleNumber) == int64(32767) {
				// no-op for default route
				fmt.Println("Inside second noop")
				continue
			}
			if !containsEntry(desired.ko.Spec.Entries, latestEntry) {
				// entry is in latest resource, but not in desired resource; therefore, delete
				toDelete = append(toDelete, latestEntry)
			}
		}
	}

	// // Checking latest for entries which has the same rule number as the entry in toAdd
	for index, entry := range toAdd {
		for _, latestEntry := range latest.ko.Spec.Entries {
			if *(entry.RuleNumber) == *(latestEntry.RuleNumber) && *(entry.Egress) == *(latestEntry.Egress) {
				toUpdate = append(toUpdate, entry)
				toAdd[index] = nil
			}
		}
	}

	// // fmt.Println("AllEntriesLatest", latest.ko.Spec.Entries)
	// // fmt.Println("AllEntriesDesired", desired.ko.Spec.Entries)
	fmt.Println("toAdd", toAdd)
	fmt.Println("toDelete", toDelete)
	fmt.Println("toUpdate", toUpdate)

	for _, entry := range toAdd {
		rlog.Debug("adding entries to nacl")

		if entry == nil {
			continue
		}

		if err = rm.createEntry(ctx, desired, *entry); err != nil {
			return err
		}
	}

	for _, entry := range toDelete {
		rlog.Debug("deleting entries from nacl")
		if err = rm.deleteEntry(ctx, latest, *entry); err != nil {
			return err
		}
		fmt.Println("DeleteEntry", *entry.RuleNumber)
	}

	for _, entry := range toUpdate {
		rlog.Debug("Updating entries in nacl")
		if err = rm.updateEntry(ctx, latest, *entry); err != nil {
			return err
		}
		fmt.Println("UpdateEntry", *entry.RuleNumber)
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
		fmt.Println()
		if len(delta.Differences) == 0 {
			return true
		}
		fmt.Println("Delta:", delta.Differences)
		for _, d := range delta.Differences {
			fmt.Println(d)

		}
	}
	return false
}

// func containsEntryRuleNumber(
// 	entryCollection []*svcapitypes.NetworkACLEntry,
// 	entry *svcapitypes.NetworkACLEntry,
// ) bool {
// 	if entryCollection == nil || entry == nil {
// 		return false
// 	}
// 	for _, e := range entryCollection {
// 		if e.RuleNumber == entry.RuleNumber && e.Egress == entry.Egress {
// 			return true
// 		}
// 	}
// 	return false
// }

// func getMatchingEntry(
// 	entryToMatch *svcapitypes.NetworkACLEntry,
// 	resource *resource,
// ) *svcapitypes.NetworkACLEntry {

// 	if resource == nil {
// 		return nil
// 	}
// 	for _, entry := range resource.ko.Spec.Entries {
// 		delta := compareNetworkACLEntry(entryToMatch, entry)
// 		if len(delta.Differences) == 0 {
// 			return entry
// 		} else {

// 			if entryToMatch.RuleNumber != nil {
// 				if !delta.DifferentAt("NetworkACLEntry.RuleNumber") {
// 					return entry
// 				}
// 			}

// 		}
// 	}
// 	return nil

// }

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

	res.NetworkAclId = r.ko.Status.NetworkACLID
	_, err = rm.sdkapi.CreateNetworkAclEntryWithContext(ctx, res)
	rm.metrics.RecordAPICall("CREATE", "CreateNetworkAclEntry", err)
	fmt.Println("createentry", res)
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

	res.NetworkAclId = r.ko.Status.NetworkACLID
	_, err = rm.sdkapi.ReplaceNetworkAclEntryWithContext(ctx, res)
	rm.metrics.RecordAPICall("CREATE", "ReplaceNetworkAclEntry", err)
	fmt.Println("updateentry", res)
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

	res.NetworkAclId = r.ko.Status.NetworkACLID
	_, err = rm.sdkapi.DeleteNetworkAclEntryWithContext(ctx, res)
	rm.metrics.RecordAPICall("CREATE", "CreateNetworkAclEntry", err)
	fmt.Println("deleteentry", res)
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
