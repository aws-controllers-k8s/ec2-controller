{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for NAT Gateway */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.NatGateway.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $natGatewayRefName, $natGatewayMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $natGatewayRefName "Tags" }}
{{- $natGatewayRef := $natGatewayMemberRefs.Shape.MemberRef }}
{{- $natGatewayRefName = "Tag" }}
func (rm *resourceManager) new{{ $natGatewayRefName }}(
	    c svcapitypes.{{ $natGatewayRefName }},
) *svcsdktypes.{{ $natGatewayRefName }} {
	res := &svcsdktypes.{{ $natGatewayRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $natGatewayRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}