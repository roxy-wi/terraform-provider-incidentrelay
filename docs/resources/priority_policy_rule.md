---
page_title: "incidentrelay_priority_policy_rule Resource - IncidentRelay"
subcategory: "Policies"
description: |-
  Manages a rule inside an IncidentRelay incident priority policy.
---

# incidentrelay_priority_policy_rule

```hcl
data "incidentrelay_incident_priority" "p1" {
  slug = "p1"
}

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

  enabled = true
}
```

## Arguments

- `policy_id` (Number, Required) Parent priority policy ID. Changing it
  recreates the rule.
- `name` (String, Required) Rule name.
- `priority_id` (Number, Required) Incident priority assigned when the rule
  matches.
- `description` (String, Optional) Rule description.
- `position` (Number, Optional/Computed) Evaluation order from 1 to 1000. When
  omitted, IncidentRelay appends the rule and returns its position.
- `matchers_json` (String, Optional) Alert matcher object encoded as JSON.
  Defaults to `{}`.
- `matcher_preset_id` (Number, Optional) Matcher preset assigned to this rule.
  The preset must be usable by the policy's team.
- `enabled` (Boolean, Optional) Whether the rule is enabled. Defaults to
  `true`.

## Import

Use the parent policy ID and rule ID separated by `/`:

```sh
terraform import incidentrelay_priority_policy_rule.critical_production 62/63
```

See the [JSON fields guide](../guides/json-fields.md).
