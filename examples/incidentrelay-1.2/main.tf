terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.3"
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

variable "slack_bot_token" {
  type      = string
  sensitive = true
}

variable "slack_app_token" {
  type      = string
  sensitive = true
}

variable "slack_channel_id" {
  type = string
}

provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  token    = var.incidentrelay_token
}

resource "incidentrelay_group" "operations" {
  slug = "operations"
  name = "Operations"
}

resource "incidentrelay_team" "platform" {
  group_id = incidentrelay_group.operations.id
  slug     = "platform"
  name     = "Platform"
}

resource "incidentrelay_channel" "slack_socket" {
  team_id      = incidentrelay_team.platform.id
  name         = "platform-slack"
  channel_type = "slack"

  config_json = jsonencode({
    mode            = "bot_api"
    connection_mode = "socket_mode"
    bot_token       = var.slack_bot_token
    app_token       = var.slack_app_token
    channel_id      = var.slack_channel_id
  })
}

resource "incidentrelay_route" "datadog" {
  team_id     = incidentrelay_team.platform.id
  name        = "platform-datadog"
  source      = "datadog"
  channel_ids = [incidentrelay_channel.slack_socket.id]

  matchers_json           = jsonencode({})
  integration_config_json = jsonencode({})
}

output "datadog_intake_token" {
  value     = incidentrelay_route.datadog.intake_token
  sensitive = true
}
