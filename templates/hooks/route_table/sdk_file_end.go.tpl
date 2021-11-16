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

{{- end }}
{{- end }}
{{- end }}
{{- end }}