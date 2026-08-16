---
page_title: "JSON Fields - IncidentRelay Provider"
subcategory: "Guides"
description: |-
  How to model IncidentRelay JSON fields in Terraform.
---

# JSON Fields

IncidentRelay uses JSON objects for matchers, labels, metadata, integration
configuration, channel configuration, and maintenance scopes. The provider
represents those API fields as Terraform strings with JSON validation.

Always prefer `jsonencode(...)` instead of writing JSON by hand:

```hcl
labels_json = jsonencode({
  owner       = "platform"
  environment = "production"
})
```

## Channel Configuration

```hcl
resource "incidentrelay_channel" "webhook" {
  team_id      = incidentrelay_team.platform.id
  name         = "platform-webhook"
  channel_type = "webhook"

  config_json = jsonencode({
    webhook_url          = var.webhook_url
    notify_on_severities = ["critical", "high"]
  })
}
```

Channel configuration is marked sensitive because it can contain tokens and
signing secrets. Sensitive values remain in Terraform state even though CLI
output is redacted, so keep state in an encrypted backend with restricted
access.

IncidentRelay 1.2 replaces Slack secrets with
`__INCIDENTRELAY_SECRET__` in API responses. The provider restores those
placeholders from the existing Terraform state during refresh. This prevents
false drift while keeping the API response secret-free. Imports cannot recover
the original values; configure the real Slack secrets before updating an
imported channel.

## Route Matchers

```hcl
resource "incidentrelay_route" "alertmanager" {
  team_id = incidentrelay_team.platform.id
  name    = "platform-alertmanager"
  source  = "alertmanager"

  matchers_json = jsonencode({
    labels = {
      team        = "platform"
      environment = "production"
    }
  })
}
```

## Priority Policy Matchers

```hcl
resource "incidentrelay_priority_policy_rule" "critical_production" {
  policy_id   = incidentrelay_priority_policy.production.id
  name        = "Critical production"
  priority_id = data.incidentrelay_incident_priority.p1.id

  matchers_json = jsonencode({
    severity = "critical"
    fields = {
      "service.environment" = "production"
    }
  })
}
```

## Integration Configuration

```hcl
integration_config_json = jsonencode({
  alertmanager = {
    accept_resolved = true
  }
})
```

## Service Labels And Metadata

```hcl
resource "incidentrelay_service" "api" {
  team_id = incidentrelay_team.platform.id
  slug    = "platform-api"
  name    = "Platform API"

  labels_json = jsonencode({
    owner = "platform"
    tier  = "tier_2"
  })

  metadata_json = jsonencode({
    repository = "https://github.com/example/platform-api"
    dashboard  = "https://grafana.example.com/d/platform-api"
  })
}
```

## Maintenance Scopes

```hcl
resource "incidentrelay_maintenance_window" "deploy" {
  name      = "Platform deploy"
  starts_at = "2026-07-13T22:00:00"
  ends_at   = "2026-07-14T01:00:00"
  timezone  = "Europe/Moscow"

  scopes_json = jsonencode([
    {
      scope_type = "team"
      team_id    = incidentrelay_team.platform.id
    },
    {
      scope_type = "service"
      service_id = incidentrelay_service.api.id
    }
  ])
}
```

## Drift Behavior

The provider normalizes JSON state so whitespace and key order do not create
Terraform drift. API-side semantic changes still appear in `terraform plan`.
