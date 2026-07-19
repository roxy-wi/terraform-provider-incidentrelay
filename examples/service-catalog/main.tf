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

provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  token    = var.incidentrelay_token
}

resource "incidentrelay_group" "infra" {
  slug = "infra"
  name = "Infrastructure"
}

resource "incidentrelay_team" "platform" {
  group_id = incidentrelay_group.infra.id
  slug     = "platform"
  name     = "Platform"
}

resource "incidentrelay_service" "api" {
  team_id      = incidentrelay_team.platform.id
  slug         = "platform-api"
  name         = "Platform API"
  service_type = "api"
  environment  = "production"
  criticality  = "high"
  tier         = "tier_2"

  labels_json = jsonencode({
    owner       = "platform"
    environment = "production"
  })

  metadata_json = jsonencode({
    repository = "https://github.com/example/platform-api"
    dashboard  = "https://grafana.example.com/d/platform-api"
  })
}

resource "incidentrelay_service" "database" {
  team_id      = incidentrelay_team.platform.id
  slug         = "platform-postgres"
  name         = "Platform PostgreSQL"
  service_type = "database"
  environment  = "production"
  criticality  = "high"
  tier         = "tier_1"
}

resource "incidentrelay_service_dependency" "api_database" {
  service_id                = incidentrelay_service.api.id
  depends_on_service_id     = incidentrelay_service.database.id
  dependency_type           = "hard"
  criticality               = "required"
  propagation_delay_seconds = 120
  correlation_enabled       = true
  enabled                   = true
  description               = "API requires PostgreSQL."
}

resource "incidentrelay_service_link" "api_dashboard" {
  service_id = incidentrelay_service.api.id
  link_type  = "dashboard"
  label      = "Grafana"
  url        = "https://grafana.example.com/d/platform-api"
  priority   = 10
  enabled    = true
}

resource "incidentrelay_service_runbook" "api_critical" {
  service_id = incidentrelay_service.api.id
  title      = "API critical alert runbook"
  url        = "https://runbooks.example.com/platform-api/critical"
  severity   = "critical"
  priority   = 10

  matchers_json = jsonencode({
    labels = {
      service = "platform-api"
    }
  })
}

resource "incidentrelay_service_match_rule" "api_labels" {
  team_id    = incidentrelay_team.platform.id
  service_id = incidentrelay_service.api.id
  name       = "Alertmanager service label"
  position   = 10

  matchers_json = jsonencode({
    labels = {
      service = "platform-api"
    }
  })
}

resource "incidentrelay_business_service" "checkout" {
  group_id           = incidentrelay_group.infra.id
  owner_team_id      = incidentrelay_team.platform.id
  slug               = "checkout"
  name               = "Checkout"
  criticality        = "critical"
  tier               = "tier_1"
  public             = true
  public_order       = 10
  public_name        = "Checkout"
  public_description = "Customer checkout path."
}

resource "incidentrelay_business_service_component" "checkout_api" {
  business_service_id = incidentrelay_business_service.checkout.id
  service_id          = incidentrelay_service.api.id
  component_type      = "technical_service"
  criticality         = "required"
  impact_weight       = 100
  position            = 10
  status_rule         = "inherit"
}
