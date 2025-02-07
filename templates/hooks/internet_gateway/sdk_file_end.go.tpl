{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for Internet Gateway */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.InternetGateway.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $igwRefName, $igwMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $igwRefName "Tags" }}
{{- $igwRef := $igwMemberRefs.Shape.MemberRef }}
{{- $igwRefName = "Tag" }}
func (rm *resourceManager) new{{ $igwRefName }}(
	    c svcapitypes.{{ $igwRefName }},
) *svcsdktypes.{{ $igwRefName }} {
	res := &svcsdktypes.{{ $igwRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $igwRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}