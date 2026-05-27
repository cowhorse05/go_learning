{{- define "go-app.name" -}}
{{- .Chart.Name }}
{{- end }}

{{- define "go-app.labels" -}}
app: {{ include "go-app.name" . }}
chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end }}

{{- define "go-app.selectorLabels" -}}
app: {{ include "go-app.name" . }}
{{- end }}
