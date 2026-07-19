terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.3"
    }
  }
}

variable "incidentrelay_base_url" {
  type        = string
  description = "IncidentRelay base URL."
  default     = "http://localhost:5000"
}

variable "incidentrelay_token" {
  type        = string
  description = "IncidentRelay API token."
  sensitive   = true
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

resource "incidentrelay_team" "platform" {
  group_id                   = incidentrelay_group.infra.id
  slug                       = "platform"
  name                       = "Platform"
  escalation_enabled         = true
  escalation_after_reminders = 2
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

resource "incidentrelay_route" "alertmanager" {
  team_id     = incidentrelay_team.platform.id
  name        = "platform-alertmanager"
  source      = "alertmanager"
  rotation_id = incidentrelay_rotation.primary.id
  channel_ids = [incidentrelay_channel.email.id]

  matchers_json = jsonencode({
    labels = {
      team = "platform"
    }
  })

  group_by = ["alertname", "instance"]
}

resource "incidentrelay_service" "api" {
  team_id      = incidentrelay_team.platform.id
  slug         = "platform-api"
  name         = "Platform API"
  service_type = "api"
  environment  = "production"
  criticality  = "high"
  tier         = "tier_2"

  labels_json = jsonencode({
    owner = "platform"
  })
}

output "group_id" {
  value = incidentrelay_group.infra.id
}

output "team_id" {
  value = incidentrelay_team.platform.id
}

output "route_intake_token_prefix" {
  value = incidentrelay_route.alertmanager.intake_token_prefix
}
