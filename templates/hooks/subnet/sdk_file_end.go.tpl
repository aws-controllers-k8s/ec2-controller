{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for Subnet */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.Subnet.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $subnetRefName, $subnetMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $subnetRefName "Tags" }}
{{- $subnetRef := $subnetMemberRefs.Shape.MemberRef }}
{{- $subnetRefName = "Tag" }}
func compare{{ $subnetRefName }}(
	    a *svcapitypes.{{ $subnetRefName }},
	    b *svcapitypes.{{ $subnetRefName }},
) *ackcompare.Delta {
	delta := ackcompare.NewDelta()
{{ GoCodeCompareStruct $CRD $subnetRef.Shape "delta" "a" "b" $subnetRefName 1 }}
	return delta
}

func (rm *resourceManager) new{{ $subnetRefName }}(
	    c svcapitypes.{{ $subnetRefName }},
) *svcsdk.{{ $subnetRefName }} {
	res := &svcsdk.{{ $subnetRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $subnetRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}