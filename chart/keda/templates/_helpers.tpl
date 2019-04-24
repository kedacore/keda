{{/* https://github.com/helm/charts/blob/master/stable/kubernetes-dashboard/templates/_helpers.tpl */}}

{{/*
Expand the name of the chart.
*/}}
{{- define "keda.name" -}}
    {{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by DNS naming spec).
If release name contains chart name it will be used as a full name
*/}}
{{- define "keda.fullname" -}}
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

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "keda.chart" -}}
    {{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "keda.serviceAccountName" -}}
	{{- default "keda-serviceaccount" .Values.serviceAccount.name -}}
{{- end -}}

{{/*
Shared labels
*/}}
{{- define "keda.labels" -}}
app: {{ template "keda.name" . }}
chart: {{ template "keda.chart" . }}
release: {{ .Release.Name }}
date: {{ now | htmlDate }}
{{- end -}}

{{/*
Create the image registry credentials.
https://github.com/helm/helm/blob/master/docs/charts_tips_and_tricks.md#creating-image-pull-secrets
*/}}
{{- define "image-pull-secret" }}
{{- printf "{\"auths\": {\"%s\": {\"auth\": \"%s\"}}}" .Values.imageCredentials.registry (printf "%s:%s" .Values.imageCredentials.username .Values.imageCredentials.password | b64enc) | b64enc }}
{{- end }}