# Changelog

## 0.3.0 - 2026-07-19

### IncidentRelay 1.2 compatibility

- Support `datadog` as an incoming route source.
- Preserve Slack Bot API secrets when IncidentRelay returns masked channel
  configuration, preventing perpetual Terraform drift after refresh.
- Document and acceptance-test Slack Socket Mode with `app_token`.
