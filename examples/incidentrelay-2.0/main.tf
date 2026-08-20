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

variable "automation_webhook_url" {
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

resource "incidentrelay_route" "uptime_kuma" {
  team_id = data.incidentrelay_team.platform.id
  name    = "Platform Uptime Kuma"
  source  = "uptime_kuma"
}

resource "incidentrelay_orchestration_webhook_action" "automation" {
  group_id = data.incidentrelay_group.infrastructure.id
  name     = "Automation receiver"
  url      = var.automation_webhook_url
  method   = "POST"

  headers_json = jsonencode({
    "Content-Type" = "application/json"
  })
}

resource "incidentrelay_event_orchestration" "production" {
  group_id           = data.incidentrelay_group.infrastructure.id
  name               = "Production event routing"
  compatibility_mode = "hybrid"
  mode               = "shadow"
  publish_comment    = "Managed by Terraform"

  rules_json = jsonencode([
    {
      name    = "Critical production"
      enabled = true
      condition_tree = {
        all = [
          {
            field    = "labels.environment"
            operator = "equals"
            value    = "production"
          },
          {
            field    = "event.severity"
            operator = "equals"
            value    = "critical"
          }
        ]
      }
      actions = [
        {
          type    = "set_team"
          team_id = data.incidentrelay_team.platform.id
        },
        {
          type      = "enqueue_webhook"
          action_id = incidentrelay_orchestration_webhook_action.automation.id
        }
      ]
      processing_mode = "continue"
      children        = []
    }
  ])
}
