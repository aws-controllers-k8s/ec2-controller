{{/* The name of the application this chart installs */}}
{{- define "ack-ec2-controller.app.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "ack-ec2-controller.app.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/* The name and version as used by the chart label */}}
{{- define "ack-ec2-controller.chart.name-version" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* The name of the service account to use */}}
{{- define "ack-ec2-controller.service-account.name" -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}

{{- define "ack-ec2-controller.watch-namespace" -}}
{{- if eq .Values.installScope "namespace" -}}
{{ .Values.watchNamespace | default .Release.Namespace }}
{{- end -}}
{{- end -}}

{{/* The mount path for the shared credentials file */}}
{{- define "ack-ec2-controller.aws.credentials.secret_mount_path" -}}
{{- "/var/run/secrets/aws" -}}
{{- end -}}

{{/* The path the shared credentials file is mounted */}}
{{- define "ack-ec2-controller.aws.credentials.path" -}}
{{ $secret_mount_path := include "ack-ec2-controller.aws.credentials.secret_mount_path" . }}
{{- printf "%s/%s" $secret_mount_path .Values.aws.credentials.secretKey -}}
{{- end -}}

{{/* The rules a of ClusterRole or Role */}}
{{- define "ack-ec2-controller.rbac-rules" -}}
rules:
- apiGroups:
  - ""
  resources:
  - configmaps
  - secrets
  verbs:
  - get
  - list
  - patch
  - watch
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - capacityreservations
  - dhcpoptions
  - elasticipaddresses
  - flowlogs
  - instances
  - internetgateways
  - launchtemplates
  - managedprefixlists
  - natgateways
  - networkacls
  - routetables
  - securitygroups
  - subnets
  - transitgateways
  - transitgatewayvpcattachments
  - vpcendpoints
  - vpcendpointserviceconfigurations
  - vpcpeeringconnections
  - vpcs
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - capacityreservations/status
  - dhcpoptions/status
  - elasticipaddresses/status
  - flowlogs/status
  - instances/status
  - internetgateways/status
  - launchtemplates/status
  - managedprefixlists/status
  - natgateways/status
  - networkacls/status
  - routetables/status
  - securitygroups/status
  - subnets/status
  - transitgateways/status
  - transitgatewayvpcattachments/status
  - vpcendpoints/status
  - vpcendpointserviceconfigurations/status
  - vpcpeeringconnections/status
  - vpcs/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - services.k8s.aws
  resources:
  - fieldexports
  - iamroleselectors
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - services.k8s.aws
  resources:
  - fieldexports/status
  - iamroleselectors/status
  verbs:
  - get
  - patch
  - update
{{- end }}

{{/* Convert k/v map to string like: "key1=value1,key2=value2,..." */}}
{{- define "ack-ec2-controller.feature-gates" -}}
{{- $list := list -}}
{{- range $k, $v := .Values.featureGates -}}
{{- $list = append $list (printf "%s=%s" $k ( $v | toString)) -}}
{{- end -}}
{{ join "," $list }}
{{- end -}}
