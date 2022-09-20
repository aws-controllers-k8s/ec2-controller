{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{ range $specFieldName, $specField := $CRD.Config.Resources.RouteTable.Fields -}}

{{/* Route is a CustomField */}}
{{- if $specField.CustomField }}
{{- $memberRefName := $specField.CustomField.ListOf }}
{{- range $index, $customShape := $SDKAPI.CustomShapes }}
{{- if (eq (Dereference $customShape.MemberShapeName) $memberRefName) }}

{{- $memberRef := $customShape.Shape.MemberRef }}

func compare{{$memberRefName}} (
	a *svcapitypes.{{ $memberRefName }},
	b *svcapitypes.{{ $memberRefName }},
) *ackcompare.Delta {
	delta := ackcompare.NewDelta()
{{ GoCodeCompareStruct $CRD $memberRef.Shape "delta" "a" "b" $memberRefName 1 }}
	return delta
}

func (rm *resourceManager) new{{ $memberRefName }}(
	c svcapitypes.{{ $memberRefName }},
) *svcsdk.{{ $memberRefName }} {
	res := &svcsdk.{{ $memberRefName }}{}

{{ GoCodeSetSDKForStruct $CRD "" "res" $memberRef "" "c" 1 }}

	return res
}

{{/* Helper method for tag support */}}
{{- range $specFieldName, $specField := $CRD.Config.Resources.RouteTable.Fields }}
{{- if $specField.From }}
{{- $operationName := $specField.From.Operation }}
{{- $operation := (index $SDKAPI.API.Operations $operationName) -}}
{{- range $rtRefName, $rtMemberRefs := $operation.InputRef.Shape.MemberRefs -}}
{{- if eq $rtRefName "Tags" }}
{{- $rtRef := $rtMemberRefs.Shape.MemberRef }}
{{- $rtRefName = "Tag" }}

func (rm *resourceManager) new{{ $rtRefName }}(
	    c svcapitypes.{{ $rtRefName }},
) *svcsdk.{{ $rtRefName }} {
	res := &svcsdk.{{ $rtRefName }}{}
{{ GoCodeSetSDKForStruct $CRD "" "res" $rtRef "" "c" 1 }}
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

{{/* Delete operation for Routes */}}

{{- $deleteInputRef := (index $SDKAPI.API.Operations "DeleteRoute").InputRef }}
{{- $deleteInputName := $deleteInputRef.ShapeName }}

func (rm *resourceManager) new{{ $deleteInputName }}(
	c svcapitypes.CreateRouteInput,
) *svcsdk.{{ $deleteInputName }} {
	res := &svcsdk.{{ $deleteInputName }}{}

{{ GoCodeSetSDKForStruct $CRD "" "res" $deleteInputRef "" "c" 1 }}

	return res
}

{{/* Setter for Route */}}

{{- $routeRef := (index (index $SDKAPI.API.Shapes "RouteTable").MemberRefs "Routes").Shape.MemberRef }}
{{- $routeRefName := $routeRef.ShapeName }}

// set{{ $routeRefName }} sets a resource {{ $routeRefName }} type
// given the SDK type.
func (rm *resourceManager) setResource{{ $routeRefName }}(
    resp *svcsdk.{{ $routeRefName }},
) *svcapitypes.{{ $routeRefName }} {
    res := &svcapitypes.{{ $routeRefName }}{}

{{ GoCodeSetResourceForStruct $CRD "RouteStatuses" "res" $routeRef "resp" $routeRef 1 }}
    return res
}