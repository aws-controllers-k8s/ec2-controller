{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for vpcEndpoint */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.VpcEndpoint.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $vpcEndpointRefName, $vpcEndpointMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $vpcEndpointRefName "Tags" }}
{{- $vpcEndpointRef := $vpcEndpointMemberRefs.Shape.MemberRef }}
{{- $vpcEndpointRefName = "Tag" }}
func (rm *resourceManager) new{{ $vpcEndpointRefName }}(
	    c svcapitypes.{{ $vpcEndpointRefName }},
) *svcsdktypes.{{ $vpcEndpointRefName }} {
	res := &svcsdktypes.{{ $vpcEndpointRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $vpcEndpointRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}