terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.1"
    }
  }
}

variable "incidentrelay_base_url" {
  type = string
}

variable "incidentrelay_token" {
  type      = string
  sensitive = true
}

provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  token    = var.incidentrelay_token
}

resource "incidentrelay_group" "infra" {
  slug        = "infra"
  name        = "Infrastructure"
  description = "Infrastructure access boundary"
}

resource "incidentrelay_admin_user" "alice" {
  username     = "alice"
  display_name = "Alice Responder"
  email        = "alice@example.com"
  active       = true
  group_id     = incidentrelay_group.infra.id
  group_role   = "user_admin"
}

resource "incidentrelay_admin_user" "bob" {
  username     = "bob"
  display_name = "Bob Responder"
  email        = "bob@example.com"
  active       = true
  group_id     = incidentrelay_group.infra.id
  group_role   = "viewer"
}

resource "incidentrelay_team" "platform" {
  group_id                   = incidentrelay_group.infra.id
  slug                       = "platform"
  name                       = "Platform"
  escalation_enabled         = true
  escalation_after_reminders = 2
}

resource "incidentrelay_team_membership" "alice" {
  team_id = incidentrelay_team.platform.id
  user_id = incidentrelay_admin_user.alice.id
  role    = "manager"
}

resource "incidentrelay_team_membership" "bob" {
  team_id = incidentrelay_team.platform.id
  user_id = incidentrelay_admin_user.bob.id
  role    = "responder"
}

resource "incidentrelay_channel" "email" {
  team_id      = incidentrelay_team.platform.id
  name         = "platform-email"
  channel_type = "email"

  config_json = jsonencode({
    notify_on_severities = ["critical", "high"]
  })
}

resource "incidentrelay_rotation" "primary" {
  team_id                   = incidentrelay_team.platform.id
  name                      = "Primary"
  start_at                  = "2026-07-13T09:00:00"
  rotation_type             = "weekly"
  interval_value            = 1
  interval_unit             = "weeks"
  handoff_time              = "09:00"
  handoff_weekday           = 0
  timezone                  = "Europe/Moscow"
  reminder_interval_seconds = 300
  add_team_members          = false
}

resource "incidentrelay_rotation_layer" "business_hours" {
  rotation_id   = incidentrelay_rotation.primary.id
  name          = "Business hours"
  priority      = 100
  start_at      = "2026-07-13T09:00:00"
  rotation_type = "weekly"
  interval_unit = "weeks"
  handoff_time  = "09:00"
  timezone      = "Europe/Moscow"
}

resource "incidentrelay_rotation_layer_member" "alice" {
  layer_id = incidentrelay_rotation_layer.business_hours.id
  user_id  = incidentrelay_admin_user.alice.id
  position = 0
}

resource "incidentrelay_rotation_layer_member" "bob" {
  layer_id = incidentrelay_rotation_layer.business_hours.id
  user_id  = incidentrelay_admin_user.bob.id
  position = 1
}

resource "incidentrelay_escalation_policy" "critical" {
  team_id      = incidentrelay_team.platform.id
  name         = "Critical escalation"
  repeat_count = 1
}

resource "incidentrelay_escalation_policy_rule" "primary_rotation" {
  policy_id     = incidentrelay_escalation_policy.critical.id
  position      = 1
  delay_seconds = 300
  target_type   = "rotation"
  target_id     = incidentrelay_rotation.primary.id
}

resource "incidentrelay_notification_policy" "production" {
  team_id = incidentrelay_team.platform.id
  name    = "Production notifications"
}

resource "incidentrelay_notification_policy_rule" "critical_email" {
  policy_id   = incidentrelay_notification_policy.production.id
  name        = "Critical alerts to email"
  position    = 1
  event_types = ["notification", "reminder", "escalation"]
  channel_ids = [incidentrelay_channel.email.id]

  matchers_json = jsonencode({
    severity = "critical"
  })
}

resource "incidentrelay_route" "alertmanager" {
  team_id              = incidentrelay_team.platform.id
  name                 = "platform-alertmanager"
  source               = "alertmanager"
  rotation_id          = incidentrelay_rotation.primary.id
  escalation_policy_id = incidentrelay_escalation_policy.critical.id
  channel_ids          = [incidentrelay_channel.email.id]

  matchers_json = jsonencode({
    labels = {
      team = "platform"
    }
  })

  group_by = ["alertname", "instance"]
}

output "route_intake_token_prefix" {
  value = incidentrelay_route.alertmanager.intake_token_prefix
}
