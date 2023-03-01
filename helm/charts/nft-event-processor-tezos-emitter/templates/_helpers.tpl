{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "nft-event-processor-tezos-emitter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "nft-event-processor-tezos-emitter.fullname" -}}
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
{{- define "nft-event-processor-tezos-emitter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create tls secret name.
*/}}
{{- define "nft-event-processor-tezos-emitter.tlsSecretName" -}}
{{- printf "%s-tls" (include "nft-event-processor-tezos-emitter.fullname" . | trunc 59 | trimSuffix "-") }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nft-event-processor-tezos-emitter.labels" -}}
helm.sh/chart: {{ include "nft-event-processor-tezos-emitter.chart" . }}
{{ include "nft-event-processor-tezos-emitter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nft-event-processor-tezos-emitter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nft-event-processor-tezos-emitter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
