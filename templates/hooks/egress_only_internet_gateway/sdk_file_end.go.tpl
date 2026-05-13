{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for EgressOnlyInternetGateway */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.EgressOnlyInternetGateway.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $eoigwRefName, $eoigwMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $eoigwRefName "Tags" }}
{{- $eoigwRef := $eoigwMemberRefs.Shape.MemberRef }}
{{- $eoigwRefName = "Tag" }}
func (rm *resourceManager) new{{ $eoigwRefName }}(
	    c svcapitypes.{{ $eoigwRefName }},
) svcsdktypes.{{ $eoigwRefName }} {
	res := svcsdktypes.{{ $eoigwRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $eoigwRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
