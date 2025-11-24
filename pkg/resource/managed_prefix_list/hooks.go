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

package managed_prefix_list

import (
	"context"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	svcapitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/ec2-controller/pkg/tags"
)

// updateTagSpecificationsInCreateRequest adds
// Tags defined in the Spec to CreateManagedPrefixListInput.TagSpecification
// and ensures the ResourceType is always set to 'prefix-list'
func updateTagSpecificationsInCreateRequest(r *resource,
	input *svcsdk.CreateManagedPrefixListInput) {
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
		desiredTagSpecs.ResourceType = "prefix-list"
		desiredTagSpecs.Tags = requestedTags
		input.TagSpecifications = []svcsdktypes.TagSpecification{desiredTagSpecs}
	}
}

// customUpdateManagedPrefixList provides custom logic for updating ManagedPrefixList
func (rm *resourceManager) customUpdateManagedPrefixList(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (*resource, error) {
	// Handle tag updates first
	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.ID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	// Only continue if something other than Tags has changed in the Spec
	if !delta.DifferentExcept("Spec.Tags") {
		return desired, nil
	}

	// Build the modify input
	input := &svcsdk.ModifyManagedPrefixListInput{}
	input.PrefixListId = latest.ko.Status.ID

	// Always set the Name field from desired state
	input.PrefixListName = desired.ko.Spec.Name

	if delta.DifferentAt("Spec.MaxEntries") {
		if desired.ko.Spec.MaxEntries != nil {
			maxEntriesCopy := int32(*desired.ko.Spec.MaxEntries)
			input.MaxEntries = &maxEntriesCopy
		}
	}

	// Handle entries changes
	if delta.DifferentAt("Spec.Entries") {
		// Build maps of current and desired entries
		currentEntries := buildEntriesMap(latest.ko.Spec.Entries)
		desiredEntries := buildEntriesMap(desired.ko.Spec.Entries)

		// Calculate entries to add and remove
		addEntries := computeEntriesToAdd(desiredEntries, currentEntries)
		removeEntries := computeEntriesToRemove(desiredEntries, currentEntries)

		if len(addEntries) > 0 {
			input.AddEntries = addEntries
		}
		if len(removeEntries) > 0 {
			input.RemoveEntries = removeEntries
		}
	}

	// Set current version for optimistic locking (required by AWS for any modification)
	if latest.ko.Status.Version != nil {
		input.CurrentVersion = latest.ko.Status.Version
	}

	// Only call ModifyManagedPrefixList if there are actual changes
	if input.PrefixListName != nil || input.MaxEntries != nil ||
		len(input.AddEntries) > 0 || len(input.RemoveEntries) > 0 {
		resp, err := rm.sdkapi.ModifyManagedPrefixList(ctx, input)
		rm.metrics.RecordAPICall("UPDATE", "ModifyManagedPrefixList", err)
		if err != nil {
			return nil, err
		}

		// Update the status with the response
		if resp.PrefixList != nil {
			if resp.PrefixList.State != "" {
				desired.ko.Status.State = aws.String(string(resp.PrefixList.State))
			}
			if resp.PrefixList.Version != nil {
				desired.ko.Status.Version = resp.PrefixList.Version
			}
		}
	}

	return desired, nil
}

// buildEntriesMap converts a list of prefix list entries into a map
// where the key is the CIDR and the value is the description.
// Returns an empty map if the input is nil.
func buildEntriesMap(entries []*svcapitypes.AddPrefixListEntry) map[string]string {
	entriesMap := make(map[string]string)
	if entries == nil {
		return entriesMap
	}

	for _, entry := range entries {
		if entry.CIDR != nil {
			desc := ""
			if entry.Description != nil {
				desc = *entry.Description
			}
			entriesMap[*entry.CIDR] = desc
		}
	}
	return entriesMap
}

// computeEntriesToAdd returns a list of entries that need to be added.
// An entry needs to be added if it exists in desired but not in current,
// or if it exists in both but the description has changed.
func computeEntriesToAdd(desired, current map[string]string) []svcsdktypes.AddPrefixListEntry {
	var addEntries []svcsdktypes.AddPrefixListEntry

	for cidr, desc := range desired {
		currentDesc, exists := current[cidr]
		if !exists || currentDesc != desc {
			entry := svcsdktypes.AddPrefixListEntry{
				Cidr: aws.String(cidr),
			}
			if desc != "" {
				entry.Description = aws.String(desc)
			}
			addEntries = append(addEntries, entry)
		}
	}

	return addEntries
}

// computeEntriesToRemove returns a list of entries that need to be removed.
// An entry needs to be removed if it exists in current but not in desired.
func computeEntriesToRemove(desired, current map[string]string) []svcsdktypes.RemovePrefixListEntry {
	var removeEntries []svcsdktypes.RemovePrefixListEntry

	for cidr := range current {
		if _, exists := desired[cidr]; !exists {
			removeEntries = append(removeEntries, svcsdktypes.RemovePrefixListEntry{
				Cidr: aws.String(cidr),
			})
		}
	}

	return removeEntries
}

// checkForMissingRequiredFields validates that ID is present in Status
// before attempting a read. Prefix list names are not unique in AWS, so we require
// the ID for safe lookups.
func (rm *resourceManager) checkForMissingRequiredFields(r *resource) bool {
	return r.ko.Status.ID == nil
}
