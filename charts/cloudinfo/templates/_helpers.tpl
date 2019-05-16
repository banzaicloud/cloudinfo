{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "cloudinfo.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cloudinfo.fullname" -}}
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
{{- define "cloudinfo.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Call nested templates.
Source: https://stackoverflow.com/a/52024583/3027614
*/}}
{{- define "call-nested" }}
{{- $dot := index . 0 }}
{{- $subchart := index . 1 }}
{{- $template := index . 2 }}
{{- $subchartValues := index $dot.Values $subchart -}}
{{- $globalValues := dict "global" (index $dot.Values "global") -}}
{{- $values := merge $globalValues $subchartValues -}}
{{- include $template (dict "Chart" (dict "Name" $subchart) "Values" $values "Release" $dot.Release "Capabilities" $dot.Capabilities) }}
{{- end -}}

{{/*
Create a default fully qualified app name for the frontend component.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cloudinfo.frontend.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- printf "%s-frontend" .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- printf "%s-frontend" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s-frontend" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create a default fully qualified app name for the scraper component.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cloudinfo.scraper.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- printf "%s-scraper" .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- printf "%s-scraper" .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s-scraper" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "cloudinfo.redis.host" -}}
{{- if .Values.store.redis.host -}}
{{- .Values.store.redis.host -}}
{{- else if .Values.redis.enabled -}}
{{- printf "%s-headless" (include "call-nested" (list . "redis" "redis.fullname")) -}}
{{- else -}}
{{- required "Please specify redis host" .Values.store.redis.host -}}
{{- end -}}
{{- end -}}

{{- define "cloudinfo.redis.port" -}}
{{- if .Values.store.redis.port -}}
{{- .Values.store.redis.port -}}
{{- else if .Values.redis.enabled -}}
{{- .Values.redis.redisPort -}}
{{- else -}}
{{- required "Please specify redis port" .Values.store.redis.port -}}
{{- end -}}
{{- end -}}
