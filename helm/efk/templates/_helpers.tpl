{{- define "kubernetes.conf" }}
{{- range $target := .Values.fluentd.targets }}
<source>
  @type tail
  @id in_tail_container_logs
  path /var/log/containers/*_{{ $target.namespace }}_{{ $target.name }}-*.log
  pos_file /var/log/fluentd-containers.log.pos
  tag kubernetes.*
  read_from_head true
  <parse>
    @type regexp
    expression /^(?<time>.+) (?<stream>stdout|stderr)( (?<logtag>.))? (?<log>.*)$/
  </parse>
</source>
{{- end }}
<filter kubernetes.var.log.containers.**>
  @type parser
  <parse>
    @type json
    json_parser json
  </parse>
  replace_invalid_sequence true
  emit_invalid_record_to_error false
  key_name log
  reserve_data true
</filter>
{{- end }}
