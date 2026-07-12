---
page_title: "incidentrelay_notification_policy Resource - IncidentRelay"
subcategory: "Policies"
description: |-
  Manages a reusable service notification policy.
---

# incidentrelay_notification_policy

```hcl
resource "incidentrelay_notification_policy" "production" {
  team_id     = incidentrelay_team.platform.id
  name        = "Production notifications"
  description = "Default production notification policy."
  enabled     = true
}
```

## Import

```sh
terraform import incidentrelay_notification_policy.production 60
```

See [resource reference](../guides/resource-reference.md#incidentrelay_notification_policy).
