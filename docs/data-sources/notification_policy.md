---
page_title: "incidentrelay_notification_policy Data Source - IncidentRelay"
subcategory: "Policies"
description: |-
  Looks up an existing IncidentRelay notification policy.
---

# incidentrelay_notification_policy

```hcl
data "incidentrelay_notification_policy" "production" {
  team_id = incidentrelay_team.platform.id
  name    = "Production notifications"
}

resource "incidentrelay_service" "api" {
  team_id                = incidentrelay_team.platform.id
  slug                   = "platform-api"
  name                   = "Platform API"
  notification_policy_id = data.incidentrelay_notification_policy.production.id
}
```

## Lookup Arguments

- `policy_id` (Number) Notification policy ID.
- `team_id` (Number) Owner team ID.
- `team_slug` (String) Owner team slug.
- `name` (String) Notification policy name.

At least one lookup argument is required. Prefer `policy_id`, or combine the
owner team and policy name. Ambiguous lookups return an error.

## Attributes

- `id` (String) Notification policy ID.
- `policy_id` (Number) Notification policy ID.
- `team_id` (Number) Owner team ID.
- `team_slug` (String) Owner team slug.
- `team_name` (String) Owner team name.
- `name` (String) Notification policy name.
- `description` (String) Notification policy description.
- `enabled` (Boolean) Whether the policy is enabled.
- `rules_count` (Number) Number of active policy rules.
- `services_count` (Number) Number of services using the policy.

Rules are managed separately with `incidentrelay_notification_policy_rule`.
