---
page_title: "incidentrelay_notification_policy_rule Resource - IncidentRelay"
subcategory: "Policies"
description: |-
  Manages a rule inside a service notification policy.
---

# incidentrelay_notification_policy_rule

```hcl
resource "incidentrelay_notification_policy_rule" "critical" {
  policy_id   = incidentrelay_notification_policy.production.id
  name        = "Critical alerts"
  event_types = ["notification", "reminder", "escalation"]
  channel_ids = [incidentrelay_channel.email.id]

  matchers_json = jsonencode({
    severity = "critical"
  })
}
```

## Import

Keep `policy_id` in configuration before importing.

```sh
terraform import incidentrelay_notification_policy_rule.critical 61
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_notification_policy_rule).
