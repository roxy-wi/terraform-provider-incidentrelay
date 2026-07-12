---
page_title: "incidentrelay_escalation_policy_rule Resource - IncidentRelay"
subcategory: "Policies"
description: |-
  Manages a rule inside an escalation policy.
---

# incidentrelay_escalation_policy_rule

```hcl
resource "incidentrelay_escalation_policy_rule" "primary" {
  policy_id     = incidentrelay_escalation_policy.critical.id
  position      = 1
  delay_seconds = 300
  target_type   = "rotation"
  target_id     = incidentrelay_rotation.primary.id
}
```

## Import

Keep `policy_id` in configuration before importing.

```sh
terraform import incidentrelay_escalation_policy_rule.primary 51
```

See [resource reference](../guides/resource-reference.md#incidentrelay_escalation_policy_rule).
