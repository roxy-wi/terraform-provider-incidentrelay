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

For all provider resources and data sources, fields named exactly `name` or
`slug` are limited to 40 characters. Fields named exactly `description` are
limited to 120 characters.

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

## Service Catalog

### `incidentrelay_service`

Technical service in the service catalog.

- Required: `team_id`, `slug`, `name`.
- Optional: `description`, `service_type`, `environment`, `criticality`,
  `tier`, `status`, `status_source`, `status_message`,
  `default_rotation_id`, `default_escalation_policy_id`,
  `notification_policy_id`, `labels_json`, `tags`, `metadata_json`, `enabled`,
  `public`, `public_name`, `public_description`, `public_order`.
- Computed: `group_id`, `team_name`, `team_slug`, `default_rotation_name`,
  `notification_policy_name`, `default_escalation_policy_name`.
- JSON: `labels_json`, `metadata_json`.
- Import: numeric service ID.

### `incidentrelay_service_match_rule`

Rule mapping incoming alerts to a service.

- Required: `team_id`, `service_id`, `name`, `matchers_json`.
- Optional: `route_id`, `position`, `description`, `enabled`.
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
- Optional: `reason`, `matchers_json`, `enabled`.
- JSON: `matchers_json`.
- Import: numeric silence ID.

### `incidentrelay_maintenance_window`

Maintenance window with scoped suppression behavior.

- Required: `name`, `starts_at`, `ends_at`, `scopes_json`.
- Optional: `description`, `behavior`, `timezone`, `rrule`, `enabled`.
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
