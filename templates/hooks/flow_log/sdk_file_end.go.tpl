{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for Flow Log */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.FlowLog.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $flowLogRefName, $flowLogMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $flowLogRefName "Tags" }}
{{- $flowLogRef := $flowLogMemberRefs.Shape.MemberRef }}
{{- $flowLogRefName = "Tag" }}
func (rm *resourceManager) new{{ $flowLogRefName }}(
	    c svcapitypes.{{ $flowLogRefName }},
) *svcsdk.{{ $flowLogRefName }} {
	res := &svcsdk.{{ $flowLogRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $flowLogRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}