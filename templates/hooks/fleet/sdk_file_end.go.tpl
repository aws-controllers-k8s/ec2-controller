{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for Fleet */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.Fleet.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $fleetRefName, $fleetMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $fleetRefName "Tags" }}
{{- $fleetRef := $fleetMemberRefs.Shape.MemberRef }}
{{- $fleetRefName = "Tag" }}
func (rm *resourceManager) new{{ $fleetRefName }}(
	    c svcapitypes.{{ $fleetRefName }},
) *svcsdktypes.{{ $fleetRefName }} {
	res := &svcsdktypes.{{ $fleetRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $fleetRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}