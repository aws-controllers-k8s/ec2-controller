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
	// If there are no changes, return the latest
	if delta == nil || len(delta.Differences) == 0 {
		return desired, nil
	}

	// Build the modify input
	input := &svcsdk.ModifyManagedPrefixListInput{}
	input.PrefixListId = latest.ko.Status.PrefixListID

	// Check if we need to update the prefix list name
	if delta.DifferentAt("Spec.PrefixListName") {
		input.PrefixListName = desired.ko.Spec.PrefixListName
	}

	// Check if we need to update max entries
	if delta.DifferentAt("Spec.MaxEntries") {
		if desired.ko.Spec.MaxEntries != nil {
			maxEntriesCopy := int32(*desired.ko.Spec.MaxEntries)
			input.MaxEntries = &maxEntriesCopy
		}
	}

	// Handle entries changes
	if delta.DifferentAt("Spec.Entries") {
		// Calculate entries to add and remove
		currentEntries := make(map[string]string)
		if latest.ko.Spec.Entries != nil {
			for _, entry := range latest.ko.Spec.Entries {
				if entry.CIDR != nil {
					desc := ""
					if entry.Description != nil {
						desc = *entry.Description
					}
					currentEntries[*entry.CIDR] = desc
				}
			}
		}

		desiredEntries := make(map[string]string)
		if desired.ko.Spec.Entries != nil {
			for _, entry := range desired.ko.Spec.Entries {
				if entry.CIDR != nil {
					desc := ""
					if entry.Description != nil {
						desc = *entry.Description
					}
					desiredEntries[*entry.CIDR] = desc
				}
			}
		}

		// Entries to add (in desired but not in current, or descriptions changed)
		var addEntries []svcsdktypes.AddPrefixListEntry
		for cidr, desc := range desiredEntries {
			currentDesc, exists := currentEntries[cidr]
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

		// Entries to remove (in current but not in desired)
		var removeEntries []svcsdktypes.RemovePrefixListEntry
		for cidr := range currentEntries {
			if _, exists := desiredEntries[cidr]; !exists {
				removeEntries = append(removeEntries, svcsdktypes.RemovePrefixListEntry{
					Cidr: aws.String(cidr),
				})
			}
		}

		if len(addEntries) > 0 {
			input.AddEntries = addEntries
		}
		if len(removeEntries) > 0 {
			input.RemoveEntries = removeEntries
		}

		// Set current version for optimistic locking
		if latest.ko.Status.Version != nil {
			input.CurrentVersion = latest.ko.Status.Version
		}
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

	// Handle tag updates separately
	if delta.DifferentAt("Spec.Tags") {
		if err := tags.Sync(
			ctx, rm.sdkapi, rm.metrics, *latest.ko.Status.PrefixListID,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
		); err != nil {
			return nil, err
		}
	}

	return desired, nil
}
