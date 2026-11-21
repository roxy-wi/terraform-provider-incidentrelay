# Requires provider v0.10.0+ and IncidentRelay 2.5.1+.
terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.10"
    }
  }
}

provider "incidentrelay" {}

variable "directory_api_token" {
  type      = string
  sensitive = true
}

resource "incidentrelay_group" "platform" {
  name = "Platform"
  slug = "platform"
}

resource "incidentrelay_team" "platform" {
  group_id = incidentrelay_group.platform.id
  name     = "Platform"
  slug     = "platform"
}

resource "incidentrelay_service" "api" {
  team_id = incidentrelay_team.platform.id
  name    = "API"
  slug    = "api"
}

resource "incidentrelay_change_integration" "gitlab" {
  name           = "GitLab"
  owner_group_id = incidentrelay_group.platform.id
  owner_team_id  = incidentrelay_team.platform.id
}

resource "incidentrelay_change_source" "gitlab" {
  integration_id = incidentrelay_change_integration.gitlab.id
  owner_team_id  = incidentrelay_team.platform.id
  source         = "gitlab-production"
  ordering_mode  = "timestamp"
  gitlab_mappings_json = jsonencode([{
    project_id  = 42
    environment = "production"
    service_ids = [tonumber(incidentrelay_service.api.id)]
  }])
}

resource "incidentrelay_change_grant" "api" {
  integration_id    = incidentrelay_change_integration.gitlab.id
  source_id         = incidentrelay_change_source.gitlab.id
  capability        = "changed"
  target_team_id    = incidentrelay_team.platform.id
  target_service_id = incidentrelay_service.api.id
}

output "gitlab_webhook_path" {
  value = incidentrelay_change_source.gitlab.gitlab_webhook_path
}

resource "incidentrelay_directory_sync_source" "directory" {
  name     = "Company directory"
  group_id = incidentrelay_group.platform.id
  # Review and apply a preview in the IR UX before setting this to true.
  enabled = false
  headers_json = jsonencode({
    Authorization = "Bearer ${var.directory_api_token}"
  })
  config_json = jsonencode({
    base_url = "https://directory.example.com"
    operations = {
      teams  = { path = "/teams", items = "teams" }
      shifts = { path = "/teams/{team_id}/shifts", items = "shifts" }
    }
    team_fields = { id = "id", name = "name" }
    members = {
      path   = "members"
      fields = { id = "id", username = "username", email = "email", name = "name" }
    }
    schedules = {
      path   = "schedules"
      fields = { id = "id", timezone = "timezone" }
    }
    shift_fields = {
      id    = "id", schedule_id = "schedule_id", user_id = "user_id"
      start = "start", end = "end"
    }
    replacements = {
      path   = "replacements"
      fields = { id = "id", user_id = "user_id", start = "start", end = "end" }
    }
    user_fields = { id = "id", username = "username", email = "email", name = "name" }
  })
}
