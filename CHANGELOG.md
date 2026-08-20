# Changelog

## Unreleased

## 0.6.0 - 2026-08-20

### IncidentRelay 2.0 compatibility

- Add `incidentrelay_event_orchestration`, which manages metadata, a JSON rule
  tree, immutable publication, and `disabled`/`shadow`/`active` runtime modes.
- Add `incidentrelay_orchestration_webhook_action` with encrypted write-only
  header preservation and redacted-URL drift protection.
- Add `apply_to_existing` and `reactivate_on_end` to silences and maintenance
  windows.
- Add `matcher_preset_id` to silences and manage their enabled state through
  the IncidentRelay 2.0 lifecycle endpoints instead of sending an unsupported
  request field.
- Document the new `uptime_kuma` route source.
- Document the granular IncidentRelay 2.0 API-token scopes required by each
  Terraform configuration domain.

## 0.5.0 - 2026-08-16

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
