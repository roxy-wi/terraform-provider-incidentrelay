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

data "incidentrelay_team" "platform" {
  slug = "platform"
}

data "incidentrelay_service" "api" {
  slug = "platform-api"
}

resource "incidentrelay_silence" "deploy" {
  team_id   = data.incidentrelay_team.platform.id
  name      = "Platform deploy silence"
  reason    = "Planned production deploy"
  starts_at = "2026-07-13T22:00:00"
  ends_at   = "2026-07-14T01:00:00"

  matchers_json = jsonencode({
    labels = {
      service = "platform-api"
    }
  })
}

resource "incidentrelay_maintenance_window" "deploy" {
  name        = "Platform deploy"
  description = "Suppress notifications while platform-api is being deployed."
  behavior    = "suppress_notifications"
  timezone    = "Europe/Moscow"
  starts_at   = "2026-07-13T22:00:00"
  ends_at     = "2026-07-14T01:00:00"

  scopes_json = jsonencode([
    {
      scope_type = "team"
      team_id    = data.incidentrelay_team.platform.id
    },
    {
      scope_type = "service"
      service_id = data.incidentrelay_service.api.id
    }
  ])
}
