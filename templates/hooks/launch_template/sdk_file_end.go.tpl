{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{- $launchtTemplateDataResponseRef := (index (index $SDKAPI.API.Shapes "LaunchTemplateVersion").MemberRefs "LaunchTemplateData") }}
{{- $launchtTemplateDataResponseRefName := $launchtTemplateDataResponseRef.ShapeName }}

{{- $launchtTemplateDataRequestRef := (index (index $SDKAPI.API.Shapes "CreateLaunchTemplateInput").MemberRefs "LaunchTemplateData") }}
{{- $launchtTemplateDataRequestRefName := $launchtTemplateDataRequestRef.ShapeName }}

// set{{ $launchtTemplateDataRequestRefName }} sets a resource {{ $launchtTemplateDataRequestRefName }} type
// given the SDK type.
func (rm *resourceManager) set{{ $launchtTemplateDataRequestRefName }}(
    resp *svcsdktypes.{{ $launchtTemplateDataResponseRefName }},
) *svcapitypes.{{ $launchtTemplateDataRequestRefName }} {
    res := &svcapitypes.{{ $launchtTemplateDataRequestRefName }}{}

{{ GoCodeSetResourceForStruct $CRD "Data" "res" $launchtTemplateDataRequestRef "resp" $launchtTemplateDataResponseRef 1 }}
    return res
}
