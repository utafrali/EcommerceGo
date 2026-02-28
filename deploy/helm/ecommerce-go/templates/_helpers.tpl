{{/*
_helpers.tpl — EcommerceGo Helm chart template helpers
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "ecommerce-go.name" -}}
{{- default .Chart.Name .Values.global.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
Truncates at 63 chars because some Kubernetes name fields are limited to this.
*/}}
{{- define "ecommerce-go.fullname" -}}
{{- if .Values.global.fullnameOverride }}
{{- .Values.global.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.global.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Fully qualified name for a specific service component.
Usage: include "ecommerce-go.serviceName" (dict "root" . "name" "product")
*/}}
{{- define "ecommerce-go.serviceName" -}}
{{- printf "%s-%s" (include "ecommerce-go.fullname" .root) .name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart label (chart name + version, no "+" chars allowed in label values).
*/}}
{{- define "ecommerce-go.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to every resource.
*/}}
{{- define "ecommerce-go.labels" -}}
helm.sh/chart: {{ include "ecommerce-go.chart" . }}
{{ include "ecommerce-go.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels — used in Deployment.spec.selector and Service.spec.selector.
Requires a "component" key in the calling context dict.
Usage: include "ecommerce-go.selectorLabels" (dict "Chart" .Chart "Release" .Release "component" "product")
*/}}
{{- define "ecommerce-go.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ecommerce-go.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .component }}
app.kubernetes.io/component: {{ .component }}
{{- end }}
{{- end }}

{{/*
Image reference helper.
Usage: include "ecommerce-go.image" (dict "global" .Values.global "image" $service.image)
Prepends global.imageRegistry when set.
*/}}
{{- define "ecommerce-go.image" -}}
{{- $registry := .global.imageRegistry | default "" }}
{{- $repo := .image.repository }}
{{- $tag := .image.tag | default "latest" }}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry $repo $tag }}
{{- else }}
{{- printf "%s:%s" $repo $tag }}
{{- end }}
{{- end }}

{{/*
Merge resource spec: use service-level override when non-empty, else fall back to defaultResources.
Usage: include "ecommerce-go.resources" (dict "default" .Values.defaultResources "override" $service.resources)
*/}}
{{- define "ecommerce-go.resources" -}}
{{- if .override }}
{{- toYaml .override }}
{{- else }}
{{- toYaml .default }}
{{- end }}
{{- end }}

{{/*
Merge probe config: use service-level override when non-empty, else fall back to defaultProbes.
Usage: include "ecommerce-go.probes" (dict "default" .Values.defaultProbes "override" $service.probes "port" 8001)
*/}}
{{- define "ecommerce-go.probes" -}}
{{- $liveness  := .override.liveness  | default .default.liveness }}
{{- $readiness := .override.readiness | default .default.readiness }}
livenessProbe:
  httpGet:
    path: {{ $liveness.path }}
    port: http
  initialDelaySeconds: {{ $liveness.initialDelaySeconds }}
  periodSeconds: {{ $liveness.periodSeconds }}
  timeoutSeconds: {{ $liveness.timeoutSeconds }}
  failureThreshold: {{ $liveness.failureThreshold }}
readinessProbe:
  httpGet:
    path: {{ $readiness.path }}
    port: http
  initialDelaySeconds: {{ $readiness.initialDelaySeconds }}
  periodSeconds: {{ $readiness.periodSeconds }}
  timeoutSeconds: {{ $readiness.timeoutSeconds }}
  failureThreshold: {{ $readiness.failureThreshold }}
{{- end }}

{{/*
imagePullSecrets block.
Usage: include "ecommerce-go.imagePullSecrets" .
*/}}
{{- define "ecommerce-go.imagePullSecrets" -}}
{{- with .Values.global.imagePullSecrets }}
imagePullSecrets:
  {{- range . }}
  - name: {{ . }}
  {{- end }}
{{- end }}
{{- end }}
