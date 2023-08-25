{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for VpcEndpointServiceConfiguration */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.VpcEndpointServiceConfiguration.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $vpcEndpointServiceConfigurationRefName, $vpcEndpointServiceConfigurationMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $vpcEndpointServiceConfigurationRefName "Tags" }}
{{- $vpcEndpointServiceConfigurationRef := $vpcEndpointServiceConfigurationMemberRefs.Shape.MemberRef }}
{{- $vpcEndpointServiceConfigurationRefName = "Tag" }}
func (rm *resourceManager) new{{ $vpcEndpointServiceConfigurationRefName }}(
	    c svcapitypes.{{ $vpcEndpointServiceConfigurationRefName }},
) *svcsdk.{{ $vpcEndpointServiceConfigurationRefName }} {
	res := &svcsdk.{{ $vpcEndpointServiceConfigurationRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $vpcEndpointServiceConfigurationRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
