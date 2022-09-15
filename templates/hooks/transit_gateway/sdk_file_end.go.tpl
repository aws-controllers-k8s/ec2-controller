{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for Transit Gateway */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.TransitGateway.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $transitGatewayRefName, $transitGatewayMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $transitGatewayRefName "Tags" }}
{{- $transitGatewayRef := $transitGatewayMemberRefs.Shape.MemberRef }}
{{- $transitGatewayRefName = "Tag" }}
func (rm *resourceManager) new{{ $transitGatewayRefName }}(
	    c svcapitypes.{{ $transitGatewayRefName }},
) *svcsdk.{{ $transitGatewayRefName }} {
	res := &svcsdk.{{ $transitGatewayRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $transitGatewayRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}