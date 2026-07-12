---
page_title: "incidentrelay_escalation_policy Resource - IncidentRelay"
subcategory: "Policies"
description: |-
  Manages an IncidentRelay escalation policy.
---

# incidentrelay_escalation_policy

```hcl
resource "incidentrelay_escalation_policy" "critical" {
  team_id      = incidentrelay_team.platform.id
  name         = "Critical escalation"
  repeat_count = 1
}
```

## Import

```sh
terraform import incidentrelay_escalation_policy.critical 50
```

See [resource reference](../guides/resource-reference.md#incidentrelay_escalation_policy).
