{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for Instance */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.Instance.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $instanceRefName, $instanceMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $instanceRefName "Tags" }}
{{- $instanceRef := $instanceMemberRefs.Shape.MemberRef }}
{{- $instanceRefName = "Tag" }}
func (rm *resourceManager) new{{ $instanceRefName }}(
	    c svcapitypes.{{ $instanceRefName }},
) *svcsdk.{{ $instanceRefName }} {
	res := &svcsdk.{{ $instanceRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $instanceRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}