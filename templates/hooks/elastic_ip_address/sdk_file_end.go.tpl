{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for ElasticIP */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.ElasticIPAddress.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $eipRefName, $eipMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $eipRefName "Tags" }}
{{- $eipRef := $eipMemberRefs.Shape.MemberRef }}
{{- $eipRefName = "Tag" }}
func (rm *resourceManager) new{{ $eipRefName }}(
	    c svcapitypes.{{ $eipRefName }},
) *svcsdktypes.{{ $eipRefName }} {
	res := &svcsdktypes.{{ $eipRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $eipRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}