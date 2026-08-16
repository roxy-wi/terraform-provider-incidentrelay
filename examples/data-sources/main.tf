terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.4"
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

data "incidentrelay_version" "current" {}

data "incidentrelay_group" "infra" {
  slug = "infra"
}

data "incidentrelay_team" "platform" {
  group_id = data.incidentrelay_group.infra.id
  slug     = "platform"
}

data "incidentrelay_user" "alice" {
  username = "alice"
}

data "incidentrelay_channel" "email" {
  group_slug   = data.incidentrelay_group.infra.slug
  team_slug    = data.incidentrelay_team.platform.slug
  name         = "platform-email"
  channel_type = "email"
}

data "incidentrelay_service" "api" {
  team_id = data.incidentrelay_team.platform.id
  slug    = "platform-api"
}

data "incidentrelay_service_match_rule" "api_labels" {
  service_id = data.incidentrelay_service.api.id
  name       = "API labels"
}

data "incidentrelay_rotation" "primary" {
  team_slug = data.incidentrelay_team.platform.slug
  name      = "Primary"
}

data "incidentrelay_incident_priority" "p1" {
  slug = "p1"
}

data "incidentrelay_escalation_policy" "critical" {
  team_id = data.incidentrelay_team.platform.id
  name    = "Critical escalation"
}

data "incidentrelay_notification_policy" "production" {
  team_id = data.incidentrelay_team.platform.id
  name    = "Production notifications"
}

output "incidentrelay_version" {
  value = data.incidentrelay_version.current.service_version
}

output "platform_team_id" {
  value = data.incidentrelay_team.platform.id
}

output "api_service_id" {
  value = data.incidentrelay_service.api.id
}

output "api_match_rule_id" {
  value = data.incidentrelay_service_match_rule.api_labels.id
}

output "email_channel_id" {
  value = data.incidentrelay_channel.email.id
}

output "primary_rotation_id" {
  value = data.incidentrelay_rotation.primary.id
}

output "p1_priority_id" {
  value = data.incidentrelay_incident_priority.p1.id
}

output "critical_escalation_policy_id" {
  value = data.incidentrelay_escalation_policy.critical.id
}

output "production_notification_policy_id" {
  value = data.incidentrelay_notification_policy.production.id
}
