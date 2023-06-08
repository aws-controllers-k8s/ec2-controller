{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for NetworkAcl */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.NetworkAcl.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $networkAclRefName, $networkAclMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $networkAclRefName "Tags" }}
{{- $networkAclRef := $networkAclMemberRefs.Shape.MemberRef }}
{{- $networkAclRefName = "Tag" }}
func (rm *resourceManager) new{{ $networkAclRefName }}(
	    c svcapitypes.{{ $networkAclRefName }},
) *svcsdk.{{ $networkAclRefName }} {
	res := &svcsdk.{{ $networkAclRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $networkAclRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
