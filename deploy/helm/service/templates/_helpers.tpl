{{- define "service.fullname" -}}
{{- default .Release.Name .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "service.imageTag" -}}
{{- required "image.tag is required" .Values.image.tag -}}
{{- end -}}

{{- define "service.baseSelectorLabels" -}}
app.kubernetes.io/name: {{ include "service.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "service.selectorLabels" -}}
{{ include "service.baseSelectorLabels" . }}
app.kubernetes.io/component: app
{{- end -}}

{{- define "service.commonLabels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
app.kubernetes.io/version: {{ include "service.imageTag" . | quote }}
app.kubernetes.io/part-of: backend
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "service.labels" -}}
{{ include "service.selectorLabels" . }}
{{ include "service.commonLabels" . }}
{{- end -}}

{{- define "service.secretName" -}}
{{- if .Values.secret.create -}}
{{ include "service.fullname" . }}
{{- else -}}
{{ .Values.secret.existingSecret }}
{{- end -}}
{{- end -}}

{{- define "service.goMemLimit" -}}
{{- $memory := toString .Values.resources.limits.memory -}}
{{- $mib := 0 -}}
{{- if hasSuffix "Gi" $memory -}}
{{- $mib = mul (trimSuffix "Gi" $memory | atoi) 1024 -}}
{{- else if hasSuffix "Mi" $memory -}}
{{- $mib = trimSuffix "Mi" $memory | atoi -}}
{{- else -}}
{{- fail "resources.limits.memory must be set in Mi or Gi" -}}
{{- end -}}
{{- printf "%dMiB" (div (mul $mib .Values.goMemLimitPercent) 100) -}}
{{- end -}}

{{- define "service.podSecurityContext" -}}
runAsNonRoot: true
runAsUser: 65532
runAsGroup: 65532
seccompProfile:
  type: RuntimeDefault
{{- end -}}

{{- define "service.containerSecurityContext" -}}
allowPrivilegeEscalation: false
readOnlyRootFilesystem: true
capabilities:
  drop:
    - ALL
{{- end -}}
