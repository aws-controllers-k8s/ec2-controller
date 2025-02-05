{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for VPC Peering Connection */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.VpcPeeringConnection.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $vpcPeeringConnectionRefName, $vpcPeeringConnectionMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $vpcPeeringConnectionRefName "Tags" }}
{{- $vpcPeeringConnectionRef := $vpcPeeringConnectionMemberRefs.Shape.MemberRef }}
{{- $vpcPeeringConnectionRefName = "Tag" }}
func (rm *resourceManager) new{{ $vpcPeeringConnectionRefName }}(
	    c svcapitypes.{{ $vpcPeeringConnectionRefName }},
) *svcsdktypes.{{ $vpcPeeringConnectionRefName }} {
	res := &svcsdktypes.{{ $vpcPeeringConnectionRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $vpcPeeringConnectionRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}