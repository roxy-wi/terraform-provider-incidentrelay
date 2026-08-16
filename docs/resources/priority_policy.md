---
page_title: "incidentrelay_priority_policy Resource - IncidentRelay"
subcategory: "Policies"
description: |-
  Manages an IncidentRelay incident priority policy.
---

# incidentrelay_priority_policy

```hcl
data "incidentrelay_incident_priority" "p1" {
  slug = "p1"
}

resource "incidentrelay_priority_policy" "production" {
  team_id     = incidentrelay_team.platform.id
  name        = "Production priority"
  description = "Automatic incident priority for production services."

  enabled              = true
  default_for_team     = true
  update_mode          = "raise_only"
  source_priority_mode = "ignore"

  fallback_mode        = "fixed_priority"
  fallback_priority_id = data.incidentrelay_incident_priority.p1.id
}
```

## Arguments

- `team_id` (Number, Required) Owner team ID. Changing it recreates the policy.
- `name` (String, Required) Policy name.
- `description` (String, Optional) Policy description.
- `enabled` (Boolean, Optional) Whether the policy is enabled. Defaults to
  `true`.
- `default_for_team` (Boolean, Optional) Make this the team's default priority
  policy. Defaults to `false`. IncidentRelay allows one default policy per
  team.
- `update_mode` (String, Optional) How the policy updates an existing incident:
  `raise_only`, `recalculate`, or `initial_only`. Defaults to `raise_only`.
- `source_priority_mode` (String, Optional) Whether to `ignore` or `prefer` an
  incoming source priority. Defaults to `ignore`.
- `fallback_mode` (String, Optional) `severity_mapping` or `fixed_priority`.
  Defaults to `severity_mapping`.
- `fallback_priority_id` (Number, Optional) Incident priority ID. Required by
  the IncidentRelay API when `fallback_mode = "fixed_priority"`.

## Read-Only Attributes

- `team_name` (String) Owner team name.
- `team_slug` (String) Owner team slug.
- `rules_count` (Number) Number of active policy rules.
- `services_count` (Number) Number of services using the policy.

IncidentRelay rejects deletion while the policy is assigned to a service.

## Import

```sh
terraform import incidentrelay_priority_policy.production 62
```

See the [priority policy rule resource](priority_policy_rule.md).
