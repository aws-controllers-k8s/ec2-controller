{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for KeyPair */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.KeyPair.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $kpRefName, $eipMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $kpRefName "Tags" }}
{{- $kpRef := $eipMemberRefs.Shape.MemberRef }}
{{- $kpRefName = "Tag" }}
func (rm *resourceManager) new{{ $kpRefName }}(
	    c svcapitypes.{{ $kpRefName }},
) *svcsdktypes.{{ $kpRefName }} {
	res := &svcsdktypes.{{ $kpRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $kpRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}