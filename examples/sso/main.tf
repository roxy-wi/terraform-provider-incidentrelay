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

variable "oidc_client_secret" {
  type      = string
  sensitive = true
}

provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  token    = var.incidentrelay_token
}

resource "incidentrelay_group" "infrastructure" {
  slug = "infrastructure"
  name = "Infrastructure"
}

resource "incidentrelay_team" "platform" {
  group_id = incidentrelay_group.infrastructure.id
  slug     = "platform"
  name     = "Platform"
}

resource "incidentrelay_sso_provider" "corporate" {
  slug     = "corporate-oidc"
  label    = "Corporate login"
  protocol = "oidc"

  client_id         = "incidentrelay"
  client_secret     = var.oidc_client_secret
  oidc_metadata_url = "https://idp.example.com/.well-known/openid-configuration"
  oidc_scope        = "openid email profile groups"

  allowed_domains = ["example.com"]

  auto_create_users                = true
  auto_link_by_email               = true
  require_verified_email           = true
  sync_group_memberships           = true
  remove_missing_group_memberships = false
}

resource "incidentrelay_sso_group_mapping" "platform" {
  provider_id    = incidentrelay_sso_provider.corporate.id
  external_group = "IncidentRelay-Platform"

  group_id   = incidentrelay_group.infrastructure.id
  group_role = "editor"

  team_id   = incidentrelay_team.platform.id
  team_role = "responder"
}
