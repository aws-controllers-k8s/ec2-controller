{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for Subnet */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.Subnet.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $subnetRefName, $subnetMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $subnetRefName "Tags" }}
{{- $subnetRef := $subnetMemberRefs.Shape.MemberRef }}
{{- $subnetRefName = "Tag" }}
func (rm *resourceManager) new{{ $subnetRefName }}(
	    c svcapitypes.{{ $subnetRefName }},
) *svcsdktypes.{{ $subnetRefName }} {
	res := &svcsdktypes.{{ $subnetRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $subnetRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}