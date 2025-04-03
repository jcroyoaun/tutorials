{{- define "service.fullname" -}}
{{- default .Values.service.name .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}