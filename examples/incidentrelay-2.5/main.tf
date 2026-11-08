# Requires provider v0.9.0+ and IR 2.5.0+.
# Use a global administrator login/JWT when creating the Group in this example.
terraform {
  required_providers {
    incidentrelay = {
      source = "roxy-wi/incidentrelay"
    }
  }
}

provider "incidentrelay" {}

resource "incidentrelay_group" "platform" {
  name = "Platform"
  slug = "platform"
}

resource "incidentrelay_team" "deploy" {
  group_id = incidentrelay_group.platform.id
  name     = "Deployments"
  slug     = "deployments"
}

resource "incidentrelay_service" "api" {
  team_id                            = incidentrelay_team.deploy.id
  name                               = "Checkout API"
  slug                               = "checkout-api"
  change_correlation_enabled         = true
  change_correlation_exclusions_json = jsonencode([{ labels = { environment = "sandbox" } }])
}

resource "incidentrelay_route" "google" {
  team_id    = incidentrelay_team.deploy.id
  service_id = incidentrelay_service.api.id
  name       = "Google Cloud Monitoring"
  source     = "google_cloud_monitoring"
  group_by   = ["gcp_project_id", "gcp_incident_id"]
}

resource "incidentrelay_change_integration" "pipeline" {
  name           = "Release pipeline"
  owner_group_id = incidentrelay_group.platform.id
  owner_team_id  = incidentrelay_team.deploy.id
}

resource "incidentrelay_change_source" "production" {
  integration_id = incidentrelay_change_integration.pipeline.id
  owner_team_id  = incidentrelay_team.deploy.id
  source         = "production-deploys"
  ordering_mode  = "sequence"
}

resource "incidentrelay_change_grant" "checkout" {
  integration_id    = incidentrelay_change_integration.pipeline.id
  source_id         = incidentrelay_change_source.production.id
  capability        = "changed"
  target_team_id    = incidentrelay_team.deploy.id
  target_service_id = incidentrelay_service.api.id
}
