terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.6"
    }
  }
}

provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  token    = var.incidentrelay_token
}

variable "incidentrelay_base_url" {
  type = string
}

variable "incidentrelay_token" {
  type      = string
  sensitive = true
}

# Terraform 1.5+ import blocks.
# Replace example IDs with IDs from your IncidentRelay instance.

import {
  to = incidentrelay_group.infra
  id = "1"
}

import {
  to = incidentrelay_admin_user.alice
  id = "2"
}

import {
  to = incidentrelay_sso_provider.corporate
  id = "3"
}

import {
  to = incidentrelay_team.platform
  id = "10"
}

import {
  to = incidentrelay_channel.email
  id = "20"
}

import {
  to = incidentrelay_route.alertmanager
  id = "30"
}

import {
  to = incidentrelay_rotation.primary
  id = "40"
}

import {
  to = incidentrelay_escalation_policy.critical
  id = "50"
}

import {
  to = incidentrelay_escalation_policy_rule.primary_rotation
  id = "51"
}

import {
  to = incidentrelay_notification_policy.production
  id = "60"
}

import {
  to = incidentrelay_notification_policy_rule.critical_email
  id = "61"
}

import {
  to = incidentrelay_priority_policy.production
  id = "62"
}

import {
  to = incidentrelay_priority_policy_rule.critical_production
  id = "62/63"
}

import {
  to = incidentrelay_service.api
  id = "70"
}

import {
  to = incidentrelay_service_match_rule.api_labels
  id = "71"
}

import {
  to = incidentrelay_service_runbook.api_critical
  id = "73"
}

import {
  to = incidentrelay_service_dependency.api_database
  id = "74"
}

import {
  to = incidentrelay_silence.deploy
  id = "79"
}

import {
  to = incidentrelay_maintenance_window.deploy
  id = "80"
}

import {
  to = incidentrelay_heartbeat.api_cron
  id = "90"
}

import {
  to = incidentrelay_business_service.checkout
  id = "100"
}

import {
  to = incidentrelay_business_service_component.checkout_api
  id = "101"
}

# Child resources read from parent list endpoints. Keep parent IDs in the
# resource configuration before importing.

import {
  to = incidentrelay_group_membership.alice
  id = "1001"
}

import {
  to = incidentrelay_sso_group_mapping.platform
  id = "1007"
}

import {
  to = incidentrelay_team_membership.alice
  id = "1002"
}

import {
  to = incidentrelay_rotation_layer.business_hours
  id = "1003"
}

import {
  to = incidentrelay_rotation_layer_member.alice
  id = "1004"
}

import {
  to = incidentrelay_rotation_override.alice_cover
  id = "1005"
}

import {
  to = incidentrelay_service_link.api_dashboard
  id = "1006"
}
