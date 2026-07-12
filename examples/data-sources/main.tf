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

data "incidentrelay_service" "api" {
  team_id = data.incidentrelay_team.platform.id
  slug    = "platform-api"
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
