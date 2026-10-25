# Requires IncidentRelay 2.4+ and provider v0.8.0+.
terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.8"
    }
  }
}

# Set INCIDENTRELAY_BASE_URL and INCIDENTRELAY_TOKEN in the environment.
provider "incidentrelay" {}

resource "incidentrelay_group" "search" {
  slug = "search-operations"
  name = "Search operations"
}

resource "incidentrelay_team" "search" {
  group_id = incidentrelay_group.search.id
  slug     = "search"
  name     = "Search"
}

resource "incidentrelay_channel" "fallback" {
  team_id      = incidentrelay_team.search.id
  name         = "Search email fallback"
  channel_type = "email"
  config_json = jsonencode({
    recipient_mode  = "fallback"
    recipient_email = "oncall@example.com"
  })
}

resource "incidentrelay_route" "search" {
  for_each = {
    opensearch = ["opensearch_monitor_id", "opensearch_trigger_id", "opensearch_bucket_key"]
    kibana     = ["kibana_rule_id", "kibana_alert_id"]
  }
  team_id     = incidentrelay_team.search.id
  name        = "Search ${each.key}"
  source      = each.key
  channel_ids = [incidentrelay_channel.fallback.id]
  group_by    = each.value
}

resource "incidentrelay_event_orchestration" "reopen" {
  group_id           = incidentrelay_group.search.id
  name               = "Search AlertGroup reopen"
  compatibility_mode = "hybrid"
  mode               = "active"

  rules_json = jsonencode([{
    name = "Reopen matching search alerts within five minutes"
    condition_tree = {
      any = [
        { field = "event.source", operator = "equals", value = "opensearch" },
        { field = "event.source", operator = "equals", value = "kibana" }
      ]
    }
    actions = [{ type = "set_grouping", reopen_window_seconds = 300 }]
  }])
}

output "intake_tokens" {
  value     = { for source, route in incidentrelay_route.search : source => route.intake_token }
  sensitive = true
}
