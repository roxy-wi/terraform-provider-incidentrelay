---
page_title: "Resource Reference - IncidentRelay Provider"
subcategory: "Guides"
description: |-
  Compact reference for IncidentRelay Terraform resources, child resources, JSON fields, sensitive fields, and import notes.
---

# Resource Reference

This guide summarizes all provider-managed IncidentRelay objects. Resource pages
contain focused examples; this page is a compact map for day-to-day authoring.

## Common Field Limits

Most provider resources and data sources limit fields named exactly `name` or
`slug` to 40 characters and `description` to 120 characters. Event
Orchestration follows its API limits: names up to 255 characters and
descriptions up to 8192 characters.

## Identity And Access

### `incidentrelay_group`

Access boundary for teams and users.

- Required: `slug`, `name`.
- Optional: `description`, `active`.
- Import: numeric group ID.

### `incidentrelay_admin_user`

Local IncidentRelay user managed through the admin API.

- Required: `username`.
- Optional: `display_name`, `email`, `phone`, `telegram_user_id`,
  `slack_user_id`, `mattermost_user_id`, `password`, `active`, `is_admin`,
  `group_id`, `group_role`.
- Computed: `active_group_id`, `active_group_slug`, `active_group_role`.
- Sensitive: `password`.
- Import: numeric user ID.

### `incidentrelay_group_membership`

Membership of an existing user in a group.

- Required: `group_id`, `user_id`.
- Optional: `role`, `active`.
- Computed: `username`, `display_name`.
- Import: numeric membership ID. Keep `group_id` in configuration before import.

## Single Sign-On

### `incidentrelay_sso_provider`

OIDC or SAML identity provider used for IncidentRelay login.

- Required: `slug`, `label`.
- Optional: `protocol`, `enabled`, claim names, `allowed_domains`, user
  provisioning and group synchronization settings, OIDC endpoints and
  credentials, SAML IdP/SP settings, `saml_name_id_format`,
  `extra_config_json`.
- Computed: `has_client_secret`, `has_saml_sp_private_key`.
- JSON: `extra_config_json`.
- Sensitive: `client_secret`, `saml_sp_private_key`.
- Import: numeric SSO provider ID. Secrets are not returned by the API.

### `incidentrelay_sso_group_mapping`

Maps an external identity-provider group to an IncidentRelay group and optional
team.

- Required: `provider_id`, `external_group`, `group_id`.
- Optional: `group_role`, `team_id`, `team_role`, `active`, `priority`.
- Computed: `group_slug`, `group_name`, `team_slug`, `team_name`.
- Import: numeric mapping ID. Keep `provider_id` in configuration before
  import.

## Teams And Notification Intake

### `incidentrelay_team`

On-call team owned by a group.

- Required: `group_id`, `slug`, `name`.
- Optional: `description`, `escalation_enabled`,
  `escalation_after_reminders`, `active`.
- Computed: `group_slug`, `group_name`.
- Import: numeric team ID.

### `incidentrelay_team_membership`

Membership of an existing group user in a team.

- Required: `team_id`, `user_id`.
- Optional: `role`, `active`.
- Computed: `username`, `display_name`.
- Import: numeric membership ID. Keep `team_id` in configuration before import.

### `incidentrelay_channel`

Outbound notification channel.

- Required: `team_id`, `name`, `channel_type`, `config_json`.
- Optional: `enabled`.
- JSON: `config_json`.
- Sensitive: `config_json`.
- Import: numeric channel ID.

### `incidentrelay_route`

Incoming alert route.

- Required: `team_id`, `name`, `source`.
- Optional: `rotation_id`, `service_id`, `escalation_policy_id`,
  `channel_ids`, `notification_channel_mode`, `matchers_json`,
  `integration_config_json`, `group_by`, `enabled`.
- Computed: `intake_token`, `intake_token_prefix`, `has_intake_token`,
  `service_name`, `service_slug`, `escalation_mode`.
- JSON: `matchers_json`, `integration_config_json`.
- Sensitive: `intake_token`.
- Import: numeric route ID. Intake token is only returned on create or
  regeneration.
- When `escalation_policy_id` is configured, the provider sends policy
  escalation mode to the API automatically.
- IncidentRelay 2.0 adds `uptime_kuma` to the supported `source` values.

## Event Orchestration

### `incidentrelay_event_orchestration`

Versioned orchestration rule tree and runtime configuration.

- Required: `group_id`, `name`.
- Optional: `description`, `scope`, `service_id`, `compatibility_mode`,
  `rules_json`, `publish_comment`, `confirm_catch_all_drop`, `mode`.
- Computed: `enabled`, `uid`, `active_version_id`,
  `active_version_number`, `created_at`, `updated_at`.
- JSON: `rules_json` must be an array.
- Import: numeric orchestration ID.
- A change to `rules_json` creates and publishes a new immutable version before
  the provider applies the runtime mode.

### `incidentrelay_orchestration_webhook_action`

Reusable encrypted webhook action for orchestration rules.

- Required: `group_id`, `name`, `url`.
- Optional: `description`, `method`, `headers_json`, `body_template`,
  `timeout_seconds`, `retry_count`, `private_network_policy`, `enabled`.
- Computed: `uid`, `has_headers`, `created_at`, `updated_at`.
- Sensitive: `url`, `headers_json`. JSON: `headers_json`.
- Import: numeric action ID. Configure write-only headers after import.

## Rotations

### `incidentrelay_rotation`

On-call rotation.

- Required: `team_id`, `name`, `start_at`.
- Optional: `description`, `rotation_type`, `interval_value`, `interval_unit`,
  `handoff_time`, `handoff_weekday`, `timezone`, `duration_seconds`,
  `reminder_interval_seconds`, `enabled`, `add_team_members`.
- Import: numeric rotation ID.

### `incidentrelay_rotation_layer`

Layer inside a rotation schedule.

- Required: `rotation_id`, `name`.
- Optional: `description`, `priority`, `start_at`, `rotation_type`,
  `interval_value`, `interval_unit`, `handoff_time`, `handoff_weekday`,
  `timezone`, `duration_seconds`, `enabled`.
- Import: numeric layer ID. Keep `rotation_id` in configuration before import.

### `incidentrelay_rotation_layer_member`

Member period in a rotation layer.

- Required: `layer_id`, `user_id`, `position`.
- Optional: `active`, `starts_at`.
- Computed: `username`, `display_name`, `ends_at`.
- Import: numeric member ID. Keep `layer_id` in configuration before import.

### `incidentrelay_rotation_override`

Temporary user override for a rotation.

- Required: `rotation_id`, `user_id`, `starts_at`, `ends_at`.
- Optional: `reason`.
- Computed: `username`, `display_name`.
- Import: numeric override ID. Keep `rotation_id` in configuration before
  import.
- Changing any configured field recreates the override because the IncidentRelay
  API exposes create/delete for overrides.

## Policies

### `incidentrelay_escalation_policy`

Escalation policy owned by a team.

- Required: `team_id`, `name`.
- Optional: `description`, `enabled`, `repeat_count`.
- Computed: `team_name`, `team_slug`, `group_id`, `group_slug`.
- Import: numeric policy ID.

### `incidentrelay_escalation_policy_rule`

Rule inside an escalation policy.

- Required: `policy_id`, `position`, `delay_seconds`, `target_type`,
  `target_id`.
- Optional: `enabled`.
- Computed: `target_name`.
- Import: numeric rule ID. Keep `policy_id` in configuration before import.

### `incidentrelay_notification_policy`

Reusable service notification policy.

- Required: `team_id`, `name`.
- Optional: `description`, `enabled`.
- Computed: `team_name`, `team_slug`, `rules_count`, `services_count`.
- Import: numeric policy ID.

### `incidentrelay_notification_policy_rule`

Rule inside a service notification policy.

- Required: `policy_id`, `name`.
- Optional: `description`, `position`, `event_types`, `matchers_json`,
  `channel_ids`, `continue_matching`, `enabled`.
- JSON: `matchers_json`.
- Import: numeric rule ID. Keep `policy_id` in configuration before import.

### `incidentrelay_priority_policy`

Incident priority policy owned by a team.

- Required: `team_id`, `name`.
- Optional: `description`, `enabled`, `default_for_team`, `update_mode`,
  `source_priority_mode`, `fallback_mode`, `fallback_priority_id`.
- Computed: `team_name`, `team_slug`, `rules_count`, `services_count`.
- Import: numeric policy ID. Changing `team_id` recreates the policy.

### `incidentrelay_priority_policy_rule`

Rule inside an incident priority policy.

- Required: `policy_id`, `name`, `priority_id`.
- Optional: `description`, `position`, `matchers_json`,
  `matcher_preset_id`, `enabled`.
- JSON: `matchers_json`.
- Import: `policy_id/rule_id`.

## Service Catalog

### `incidentrelay_service`

Technical service in the service catalog.

- Required: `team_id`, `slug`, `name`.
- Optional: `description`, `service_type`, `environment`, `criticality`,
  `tier`, `status`, `status_source`, `status_message`,
  `default_rotation_id`, `default_escalation_policy_id`,
  `notification_policy_id`, `priority_policy_id`, `labels_json`, `tags`,
  `metadata_json`, `enabled`, `public`, `public_name`, `public_description`,
  `public_order`.
- Computed: `group_id`, `team_name`, `team_slug`, `default_rotation_name`,
  `notification_policy_name`, `priority_policy_name`,
  `default_escalation_policy_name`.
- JSON: `labels_json`, `metadata_json`.
- Import: numeric service ID.

### `incidentrelay_service_match_rule`

Rule mapping incoming alerts to a service.

- Required: `team_id`, `service_id`, `name`.
- Optional: `route_id`, `position`, `description`, `matcher_preset_id`,
  `matchers_json`, `enabled`. At least one of `matcher_preset_id` or a non-empty
  `matchers_json` object is required by the API.
- Computed: `route_name`, `service_name`.
- JSON: `matchers_json`.
- Import: numeric rule ID. Keep `service_id` in configuration before import.

### `incidentrelay_service_link`

Link attached to a service.

- Required: `service_id`, `label`, `url`.
- Optional: `link_type`, `description`, `priority`, `enabled`.
- Computed: `service_name`, `service_slug`.
- Import: numeric link ID. Keep `service_id` in configuration before import.

### `incidentrelay_service_runbook`

Runbook attached to a service.

- Required: `service_id`, `title`, `url`.
- Optional: `description`, `severity`, `matchers_json`, `priority`, `enabled`.
- Computed: `service_name`, `service_slug`.
- JSON: `matchers_json`.
- Import: numeric runbook ID. Keep `service_id` in configuration before import.

### `incidentrelay_service_dependency`

Upstream service dependency.

- Required: `service_id`, `depends_on_service_id`.
- Optional: `dependency_type`, `criticality`, `correlation_enabled`,
  `propagation_delay_seconds`, `description`, `enabled`.
- Computed: `service_name`, `service_slug`, `depends_on_service_name`,
  `depends_on_service_slug`.
- Import: numeric dependency ID. Keep `service_id` in configuration before
  import.

## Operations

### `incidentrelay_silence`

Temporary alert silence.

- Required: `team_id`, `name`, `starts_at`, `ends_at`.
- Optional: `reason`, `matcher_preset_id`, `matchers_json`,
  `apply_to_existing`, `reactivate_on_end`, `enabled`.
- JSON: `matchers_json`.
- Import: numeric silence ID.

### `incidentrelay_maintenance_window`

Maintenance window with scoped suppression behavior.

- Required: `name`, `starts_at`, `ends_at`, `scopes_json`.
- Optional: `description`, `behavior`, `timezone`, `rrule`, `enabled`,
  `apply_to_existing`, `reactivate_on_end`.
- Computed: `status`, `deleted`.
- JSON: `scopes_json`.
- Import: numeric maintenance window ID.

### `incidentrelay_heartbeat`

Dead-man-switch heartbeat.

- Required: `team_id`, `route_id`, `name`, `slug`.
- Optional: `service_id`, `description`, `mode`, `expected_interval_seconds`,
  `grace_period_seconds`, `schedule_kind`, `schedule_time`,
  `schedule_weekday`, `schedule_monthday`, `timezone`, `severity`,
  `priority_slug`, `enabled`, `auto_resolve`, `instance_tracking_enabled`,
  `instance_key`, `expected_instances_mode`, `expected_instances`,
  `auto_discovery_ttl_days`, `labels_json`, `metadata_json`.
- Computed: `uid`, `status`, `token_prefix`, `token`, `ping_url`,
  `ping_url_hint`, `last_seen_at`, `next_expected_at`, `deadline_at`,
  `current_alert_group_id`.
- JSON: `labels_json`, `metadata_json`.
- Sensitive: `token`, `ping_url`.
- Import: numeric heartbeat ID. Token and ping URL are only returned on create
  or regeneration.

## Business Services

### `incidentrelay_business_service`

Customer-facing business service.

- Required: `group_id`, `slug`, `name`.
- Optional: `owner_team_id`, `description`, `criticality`, `tier`, `public`,
  `public_name`, `public_description`, `public_order`, `labels_json`,
  `metadata_json`, `enabled`.
- Computed: `status`, `components_count`.
- JSON: `labels_json`, `metadata_json`.
- Import: numeric business service ID.

### `incidentrelay_business_service_component`

Technical service component inside a business service.

- Required: `business_service_id`, `service_id`.
- Optional: `component_type`, `criticality`, `impact_weight`, `position`,
  `status_rule`, `description`, `enabled`.
- Computed: `service_name`, `service_slug`, `service_status`.
- Import: numeric component ID. Keep `business_service_id` in configuration
  before import.
