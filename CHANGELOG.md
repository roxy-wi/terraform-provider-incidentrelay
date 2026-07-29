# Changelog

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
