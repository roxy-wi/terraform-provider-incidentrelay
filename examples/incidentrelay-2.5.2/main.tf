# Requires provider v0.11.0+ and IR 2.5.2+.
terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.11"
    }
  }
}

provider "incidentrelay" {}

variable "jira_pat" {
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

resource "incidentrelay_directory_sync_source" "jira" {
  name     = "Jira Assets"
  group_id = incidentrelay_group.platform.id
  # IR keeps this pending approval until you review and apply a preview in UX.
  enabled = true
  # Omit this argument to retain the previous disable-on-destroy behavior.
  destroy_behavior = "delete_keep_data"
  headers_json     = jsonencode({ Authorization = "Bearer ${var.jira_pat}" })
  config_json = jsonencode({
    source_type = "jira_assets"
    base_url    = "https://jira.example.com"
    # Older Insight installations can use /rest/insight/1.0/iql/objects and iql.
    api_path        = "/rest/assets/1.0/aql/objects"
    query_parameter = "qlQuery"
    services = {
      query                 = "objectType = System"
      owner_team_attribute  = "SRE Support Team"
      description_attribute = "Description"
      environment_attribute = "Environment"
      criticality_attribute = "Criticality"
    }
    business_services = {
      query                          = "objectType = Product"
      owner_team_attribute           = "Product Support Team"
      description_attribute          = "Long Description"
      fallback_description_attribute = "Short Description"
    }
    # Replace 42 with an external CMDB team object ID.
    team_matches           = { "42" = tonumber(incidentrelay_team.platform.id) }
    relationship_side      = "services"
    relationship_attribute = "Product"
  })
}

output "jira_schedule_state" {
  value = incidentrelay_directory_sync_source.jira.schedule_state
}
