{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for VPC */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.Vpc.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $vpcRefName, $vpcMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $vpcRefName "Tags" }}
{{- $vpcRef := $vpcMemberRefs.Shape.MemberRef }}
{{- $vpcRefName = "Tag" }}
func compare{{ $vpcRefName }}(
	    a *svcapitypes.{{ $vpcRefName }},
	    b *svcapitypes.{{ $vpcRefName }},
) *ackcompare.Delta {
	delta := ackcompare.NewDelta()
{{ GoCodeCompareStruct $CRD $vpcRef.Shape "delta" "a" "b" $vpcRefName 1 }}
	return delta
}

func (rm *resourceManager) new{{ $vpcRefName }}(
	    c svcapitypes.{{ $vpcRefName }},
) *svcsdk.{{ $vpcRefName }} {
	res := &svcsdk.{{ $vpcRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $vpcRef "" "c" 1 }}
	return res
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}