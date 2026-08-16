---
page_title: "incidentrelay_escalation_policy Data Source - IncidentRelay"
subcategory: "Policies"
description: |-
  Looks up an existing IncidentRelay escalation policy.
---

# incidentrelay_escalation_policy

```hcl
data "incidentrelay_escalation_policy" "critical" {
  group_slug = "infrastructure"
  team_slug  = "platform"
  name       = "Critical escalation"
}

resource "incidentrelay_service" "api" {
  team_id                      = incidentrelay_team.platform.id
  slug                         = "platform-api"
  name                         = "Platform API"
  default_escalation_policy_id = data.incidentrelay_escalation_policy.critical.id
}
```

## Lookup Arguments

- `policy_id` (Number) Escalation policy ID.
- `group_id` (Number) Owner group ID.
- `group_slug` (String) Owner group slug.
- `team_id` (Number) Owner team ID.
- `team_slug` (String) Owner team slug.
- `name` (String) Escalation policy name.

At least one lookup argument is required. Prefer `policy_id`, or combine the
owner and policy name. Ambiguous lookups return an error.

## Attributes

- `id` (String) Escalation policy ID.
- `policy_id` (Number) Escalation policy ID.
- `group_id` (Number) Owner group ID.
- `group_slug` (String) Owner group slug.
- `team_id` (Number) Owner team ID.
- `team_slug` (String) Owner team slug.
- `team_name` (String) Owner team name.
- `name` (String) Escalation policy name.
- `description` (String) Escalation policy description.
- `enabled` (Boolean) Whether the policy is enabled.
- `repeat_count` (Number) Number of additional complete rule-chain repeats.

Rules are managed separately with `incidentrelay_escalation_policy_rule`.
