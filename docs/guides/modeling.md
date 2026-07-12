---
page_title: "Modeling IncidentRelay In Terraform"
subcategory: "Guides"
description: |-
  Suggested Terraform modeling patterns for IncidentRelay teams, routes, rotations, services, and maintenance.
---

# Modeling IncidentRelay In Terraform

Use Terraform to describe stable operational structure: groups, teams, routes,
services, policies, maintenance scopes, and business service relationships.
Keep highly temporary incident data in IncidentRelay itself.

## Suggested Module Boundaries

- `identity`: groups, admin users, group memberships.
- `teams`: teams, team memberships, channels, routes, rotations.
- `policies`: escalation policies and notification policies.
- `services`: service catalog services, links, runbooks, dependencies, match rules.
- `operations`: silences, maintenance windows, heartbeats.
- `business-services`: customer-facing business services and components.

## Naming

Prefer stable slugs and descriptive display names:

```hcl
resource "incidentrelay_team" "platform" {
  group_id = incidentrelay_group.infra.id
  slug     = "platform"
  name     = "Platform"
}
```

Use Terraform resource names that match IncidentRelay slugs when possible. That
makes import, review, and drift investigation easier.

## Routes And Services

Model routes close to the teams that own intake tokens, and model services close
to the service catalog owner:

```hcl
resource "incidentrelay_route" "alertmanager" {
  team_id    = incidentrelay_team.platform.id
  name       = "platform-alertmanager"
  source     = "alertmanager"
  service_id = incidentrelay_service.api.id
}
```

## Policies

Use escalation policies to control who gets paged and notification policies to
control how notifications are delivered.

```hcl
resource "incidentrelay_escalation_policy" "critical" {
  team_id      = incidentrelay_team.platform.id
  name         = "Critical escalation"
  repeat_count = 1
}

resource "incidentrelay_notification_policy" "production" {
  team_id = incidentrelay_team.platform.id
  name    = "Production notifications"
}
```

## State Hygiene

Some API values are intentionally sensitive or one-time:

- Route `intake_token`.
- Heartbeat `token`.
- Heartbeat `ping_url`.
- User `password`.

Keep Terraform state encrypted and access-controlled.
