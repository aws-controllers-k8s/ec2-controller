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


