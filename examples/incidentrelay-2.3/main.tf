terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.7"
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

variable "lark_webhook_url" {
  type      = string
  sensitive = true
}

variable "lark_signing_secret" {
  type      = string
  sensitive = true
}

provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  token    = var.incidentrelay_token
}

data "incidentrelay_group" "infrastructure" {
  slug = "infrastructure"
}

data "incidentrelay_team" "platform" {
  group_id = data.incidentrelay_group.infrastructure.id
  slug     = "platform"
}

resource "incidentrelay_channel" "lark" {
  team_id      = data.incidentrelay_team.platform.id
  name         = "Platform Lark"
  channel_type = "lark"

  config_json = jsonencode({
    webhook_url    = var.lark_webhook_url
    signing_secret = var.lark_signing_secret
  })
}

resource "incidentrelay_route" "cloud_ru" {
  team_id     = data.incidentrelay_team.platform.id
  name        = "Platform Cloud.ru"
  source      = "cloud_ru"
  channel_ids = [incidentrelay_channel.lark.id]

  integration_config_json = jsonencode({
    cloud_ru = {
      topic_urn = "urn:smn:ru-a:project-id:incidentrelay"
    }
  })
}

resource "incidentrelay_sso_provider" "corporate" {
  slug     = "corporate-oidc"
  label    = "Corporate OIDC"
  protocol = "oidc"
  enabled  = false

  profile_claim_mappings_json = jsonencode({
    slack_user_id      = "slack_id"
    telegram_user_id   = "telegram_id"
    mattermost_user_id = "mattermost_id"
  })
}

resource "incidentrelay_notification_policy" "platform" {
  team_id = data.incidentrelay_team.platform.id
  name    = "Platform notifications"
}

resource "incidentrelay_notification_policy_rule" "critical" {
  policy_id   = incidentrelay_notification_policy.platform.id
  name        = "Critical production alerts"
  event_types = ["notification", "reminder", "escalation"]
  channel_ids = [incidentrelay_channel.lark.id]

  matchers_json = jsonencode({
    priority = ["p1"]
    severity = ["critical"]
    source   = ["alertmanager"]
    fields = {
      "service.environment" = ["production"]
    }
  })
}
