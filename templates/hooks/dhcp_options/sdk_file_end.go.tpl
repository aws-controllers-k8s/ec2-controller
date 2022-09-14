{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for DhcpOptions */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.DhcpOptions.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $dhcpOptionsRefName, $dhcpOptionsMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $dhcpOptionsRefName "Tags" }}
{{- $dhcpOptionsRef := $dhcpOptionsMemberRefs.Shape.MemberRef }}
{{- $dhcpOptionsRefName = "Tag" }}
func (rm *resourceManager) new{{ $dhcpOptionsRefName }}(
	    c svcapitypes.{{ $dhcpOptionsRefName }},
) *svcsdk.{{ $dhcpOptionsRefName }} {
	res := &svcsdk.{{ $dhcpOptionsRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $dhcpOptionsRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}