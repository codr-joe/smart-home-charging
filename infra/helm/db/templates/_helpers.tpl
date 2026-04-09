{{- define "db.name" -}}
{{- "timescaledb" }}
{{- end }}

{{- define "db.labels" -}}
app.kubernetes.io/name: {{ include "db.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "db.selectorLabels" -}}
app.kubernetes.io/name: {{ include "db.name" . }}
{{- end }}
