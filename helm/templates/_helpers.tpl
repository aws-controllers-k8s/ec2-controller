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
  - ""
  resources:
  - secrets
  verbs:
  - get
  - list
  - patch
  - watch
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - dhcpoptions
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
  - dhcpoptions/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - elasticipaddresses
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
  - elasticipaddresses/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - flowlogs
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
  - flowlogs/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - instances
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
  - instances/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - internetgateways
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
  - internetgateways/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - natgateways
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
  - natgateways/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - networkacls
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
  - networkacls/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - routetables
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
  - routetables/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - securitygroups
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
  - securitygroups/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - subnets
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
  - subnets/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - transitgateways
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
  - transitgateways/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - vpcendpoints
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
  - vpcendpoints/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - vpcendpointserviceconfigurations
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
  - vpcendpointserviceconfigurations/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
  - vpcpeeringconnections
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
  - vpcpeeringconnections/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - ec2.services.k8s.aws
  resources:
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
  - vpcs/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - services.k8s.aws
  resources:
  - adoptedresources
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
  - adoptedresources/status
  verbs:
  - get
  - patch
  - update
- apiGroups:
  - services.k8s.aws
  resources:
  - fieldexports
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
