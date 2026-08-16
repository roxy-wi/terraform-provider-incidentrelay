---
page_title: "incidentrelay_service Data Source - IncidentRelay"
subcategory: "Service Catalog"
description: |-
  Looks up an IncidentRelay service by ID, team ID, slug, or name.
---

# incidentrelay_service

```hcl
data "incidentrelay_team" "platform" {
  slug = "platform"
}

data "incidentrelay_service" "api" {
  team_id = data.incidentrelay_team.platform.id
  slug    = "platform-api"
}
```

## Lookup Arguments

- `service_id` (Number) Service ID.
- `team_id` (Number) Owner team ID.
- `slug` (String) Service slug.
- `name` (String) Service name.

At least one lookup argument is required. If more than one service matches, add
more filters.

## Attributes

- `id` (String) Service ID.
- `team_id` (Number) Owner team ID.
- `slug` (String) Service slug.
- `name` (String) Service name.
- `description` (String) Service description.
- `team_name` (String) Owner team name.
- `team_slug` (String) Owner team slug.
- `group_id` (Number) Owner group ID.
- `default_rotation_id` (Number) Default rotation ID.
- `default_rotation_name` (String) Default rotation name.
- `default_escalation_policy_id` (Number) Default escalation policy ID.
- `default_escalation_policy_name` (String) Default escalation policy name.
- `notification_policy_id` (Number) Assigned notification policy ID.
- `notification_policy_name` (String) Assigned notification policy name.
- `priority_policy_id` (Number) Assigned incident priority policy ID, or `0`
  when the service uses the team's default policy.
- `priority_policy_name` (String) Assigned incident priority policy name.
