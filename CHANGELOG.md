# Changelog

## Unreleased

### Priority policies and lookups

- Add `incidentrelay_priority_policy` and
  `incidentrelay_priority_policy_rule` resources using the IncidentRelay 1.2
  API model.
- Add `incidentrelay_channel`, `incidentrelay_rotation`,
  `incidentrelay_incident_priority`, `incidentrelay_escalation_policy`, and
  `incidentrelay_notification_policy` data sources.
- Add the `incidentrelay_service_match_rule` data source and expose the
  IncidentRelay 1.2 `matcher_preset_id` field on its resource.
- Add `priority_policy_id` support to `incidentrelay_service`.

## 0.4.0 - 2026-07-29

### SSO configuration

- Add `incidentrelay_sso_provider` for OIDC and SAML configuration.
- Add `incidentrelay_sso_group_mapping` for external group synchronization.
- Preserve configured OIDC client secrets and SAML SP private keys when the API
  returns only secret-presence flags.
- Normalize the `On-call` documentation category to prevent a duplicate
  Terraform Registry navigation section.

## 0.3.0 - 2026-07-19

### IncidentRelay 1.2 compatibility

- Support `datadog` as an incoming route source.
- Preserve Slack Bot API secrets when IncidentRelay returns masked channel
  configuration, preventing perpetual Terraform drift after refresh.
- Document and acceptance-test Slack Socket Mode with `app_token`.
