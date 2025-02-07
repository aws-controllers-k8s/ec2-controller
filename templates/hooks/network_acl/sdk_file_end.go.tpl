{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{ range $specFieldName, $specField := $CRD.Config.Resources.NetworkAcl.Fields -}}

{{/* Entry is a CustomField */}}
{{- if $specField.CustomField }}


{{- $memberRefName := $specField.CustomField.ListOf }}


{{- range $index, $customShape := $SDKAPI.CustomShapes }}

{{- if (eq (Dereference $customShape.MemberShapeName) $memberRefName) }}


{{- if eq $specFieldName "Entries" }}
{{- $memberRef := $customShape.Shape.MemberRef }}
{{ $memberRefName = "NetworkACLEntry" }}




func compare{{$memberRefName}} (
	a *svcapitypes.{{ $memberRefName }},
	b *svcapitypes.{{ $memberRefName }},
) *ackcompare.Delta {
	delta := ackcompare.NewDelta()
{{ GoCodeCompareStruct $CRD $memberRef.Shape "delta" "a" "b" $memberRefName 1 }}
	return delta
}

{{/* Helper method for tag support */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.RouteTable.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $rtRefName, $rtMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $rtRefName "Tags" }}
{{- $rtRef := $rtMemberRefs.Shape.MemberRef }}
{{- $rtRefName = "Tag" }}

func (rm *resourceManager) new{{ $rtRefName }}(
	    c svcapitypes.{{ $rtRefName }},
) *svcsdktypes.{{ $rtRefName }} {
	res := &svcsdktypes.{{ $rtRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $rtRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}


{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

