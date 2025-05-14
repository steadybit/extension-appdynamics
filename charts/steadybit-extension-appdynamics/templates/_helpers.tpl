{{/* vim: set filetype=mustache: */}}

{{- define "appdynamics.secret.name" -}}
{{- default "steadybit-extension-appdynamics" .Values.appdynamics.existingSecret -}}
{{- end -}}
