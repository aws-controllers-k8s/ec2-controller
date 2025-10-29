{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for ManagedPrefixList */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.ManagedPrefixList.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $prefixListRefName, $prefixListMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $prefixListRefName "Tags" }}
{{- $prefixListRef := $prefixListMemberRefs.Shape.MemberRef }}
{{- $prefixListRefName = "Tag" }}
func (rm *resourceManager) new{{ $prefixListRefName }}(
	    c svcapitypes.{{ $prefixListRefName }},
) *svcsdktypes.{{ $prefixListRefName }} {
	res := &svcsdktypes.{{ $prefixListRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $prefixListRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{/* Generate helper methods for AddPrefixListEntry */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.ManagedPrefixList.Fields }}
{{- if $specField.CustomField }}
{{- $entryRefName := $specField.CustomField.ListOf }}
{{- if eq $entryRefName "AddPrefixListEntry" }}
{{- range $index, $customShape := $SDKAPI.CustomShapes }}
{{- if (eq (Dereference $customShape.MemberShapeName) $entryRefName) }}
{{- $entryRef := $customShape.Shape.MemberRef }}

func (rm *resourceManager) new{{ $entryRefName }}(
	c svcapitypes.{{ $entryRefName }},
) (*svcsdktypes.{{ $entryRef.ShapeName }}, error) {
	res := &svcsdktypes.{{ $entryRef.ShapeName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $entryRef "" "c" 1 }}
	return res, nil
}

func (rm *resourceManager) setResource{{ $entryRefName }}(
	resp *svcsdktypes.PrefixListEntry,
) *svcapitypes.{{ $entryRefName }} {
	res := &svcapitypes.{{ $entryRefName }}{}
	if resp.Cidr != nil {
		res.CIDR = resp.Cidr
	}
	if resp.Description != nil {
		res.Description = resp.Description
	}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

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
		input.MaxEntries = desired.ko.Spec.MaxEntries
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
			if resp.PrefixList.State != nil {
				desired.ko.Status.State = resp.PrefixList.State
			}
			if resp.PrefixList.Version != nil {
				desired.ko.Status.Version = resp.PrefixList.Version
			}
		}
	}

	// Handle tag updates separately
	if delta.DifferentAt("Spec.Tags") {
		if err := rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	return desired, nil
}


