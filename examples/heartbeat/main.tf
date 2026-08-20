terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.6"
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

data "incidentrelay_team" "platform" {
  slug = "platform"
}

data "incidentrelay_service" "api" {
  slug = "platform-api"
}

resource "incidentrelay_route" "heartbeat" {
  team_id    = data.incidentrelay_team.platform.id
  service_id = data.incidentrelay_service.api.id
  name       = "platform-heartbeats"
  source     = "heartbeat"

  matchers_json = jsonencode({
    labels = {
      source = "heartbeat"
    }
  })
}

resource "incidentrelay_heartbeat" "api_cron" {
  team_id                   = data.incidentrelay_team.platform.id
  route_id                  = incidentrelay_route.heartbeat.id
  service_id                = data.incidentrelay_service.api.id
  name                      = "API nightly cron"
  slug                      = "platform-api-nightly-cron"
  mode                      = "scheduled"
  schedule_kind             = "daily"
  schedule_time             = "02:30"
  timezone                  = "Europe/Moscow"
  grace_period_seconds      = 900
  severity                  = "critical"
  priority_slug             = "p1"
  auto_resolve              = true
  instance_tracking_enabled = false

  labels_json = jsonencode({
    service = "platform-api"
    job     = "nightly-cron"
  })
}

output "heartbeat_ping_url_hint" {
  value = incidentrelay_heartbeat.api_cron.ping_url_hint
}
