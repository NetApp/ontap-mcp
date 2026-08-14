{{/*
Expand the name of the chart.
*/}}
{{- define "ontap-mcp.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "ontap-mcp.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "ontap-mcp.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "ontap-mcp.labels" -}}
helm.sh/chart: {{ include "ontap-mcp.chart" . }}
{{ include "ontap-mcp.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "ontap-mcp.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ontap-mcp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "ontap-mcp.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "ontap-mcp.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Name of the Secret that carries ontap.yaml -- the caller-supplied
existingSecret, the name ESO's ExternalSecret creates, or the one this
chart creates itself, in that order of precedence.
*/}}
{{- define "ontap-mcp.configSecretName" -}}
{{- if .Values.ontapConfig.existingSecret }}
{{- .Values.ontapConfig.existingSecret }}
{{- else if .Values.externalSecret.enabled }}
{{- default (printf "%s-config" (include "ontap-mcp.fullname" .)) .Values.externalSecret.target.name }}
{{- else }}
{{- printf "%s-config" (include "ontap-mcp.fullname" .) }}
{{- end }}
{{- end }}
