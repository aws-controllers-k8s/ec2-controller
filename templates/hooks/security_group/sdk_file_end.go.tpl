{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{/* Generate helper methods for SecurityGroupRules */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.SecurityGroup.Fields }}
{{- if $specField.CustomField }}
{{- $sgRuleRefName := $specField.CustomField.ListOf }}
{{- range $index, $customShape := $SDKAPI.CustomShapes }}
{{- if (eq (Dereference $customShape.MemberShapeName) $sgRuleRefName) }}
{{- $sgRuleRef := $customShape.Shape.MemberRef }}
{{- if eq $specFieldName "IngressRules" }}
{{ $sgRuleRefName = "IPPermission" }}
func compare{{$sgRuleRefName}} (
	a *svcapitypes.{{ $sgRuleRefName }},
	b *svcapitypes.{{ $sgRuleRefName }},
) *ackcompare.Delta {
	delta := ackcompare.NewDelta()
{{ GoCodeCompareStruct $CRD $sgRuleRef.Shape "delta" "a" "b" $sgRuleRefName 1 }}
	return delta
}

func (rm *resourceManager) new{{ $sgRuleRefName }}(
	c svcapitypes.{{ $sgRuleRefName }},
) *svcsdk.{{ $sgRuleRef.ShapeName }} {
	res := &svcsdk.{{ $sgRuleRef.ShapeName }}{}

{{ GoCodeSetSDKForStruct $CRD "" "res" $sgRuleRef "" "c" 1 }}
	return res
}

{{/* Helper method for tag support */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.SecurityGroup.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $securityGroupRefName, $securityGroupMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $securityGroupRefName "Tags" }}
{{- $securityGroupRef := $securityGroupMemberRefs.Shape.MemberRef }}
{{- $securityGroupRefName = "Tag" }}

func (rm *resourceManager) new{{ $securityGroupRefName }}(
	    c svcapitypes.{{ $securityGroupRefName }},
) *svcsdk.{{ $securityGroupRefName }} {
	res := &svcsdk.{{ $securityGroupRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $securityGroupRef "" "c" 1 }}
	return res
}

{{- end }}

{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{/* Setters for SecurityGroupRules */}}

{{- $sgRuleRef := (index $SDKAPI.API.Shapes "SecurityGroupRuleList").MemberRef }}
{{- $sgRuleName := "SecurityGroupRule" }}

// set{{ $sgRuleName }} sets a resource {{ $sgRuleName }} type
// given the SDK type.
func (rm *resourceManager) setResource{{ $sgRuleName }}(
    resp *svcsdk.{{ $sgRuleName }},
) *svcapitypes.{{ $sgRuleName }} {
    res := &svcapitypes.{{ $sgRuleName }}{}

{{ GoCodeSetResourceForStruct $CRD "IngressRules" "res" $sgRuleRef "resp" $sgRuleRef 1 }}
    return res
}

{{- $ipPermRef := (index $SDKAPI.API.Shapes "IpPermissionList").MemberRef }}
{{- $ipPermName := "IPPermission" }}

// set{{ $ipPermName }} sets a resource {{ $ipPermName }} type
// given the SDK type.
func (rm *resourceManager) setResource{{ $ipPermName }}(
    resp *svcsdk.IpPermission,
) *svcapitypes.{{ $ipPermName }} {
    res := &svcapitypes.{{ $ipPermName }}{}

{{ GoCodeSetResourceForStruct $CRD "IngressRules" "res" $ipPermRef "resp" $ipPermRef 1 }}
    return res
}