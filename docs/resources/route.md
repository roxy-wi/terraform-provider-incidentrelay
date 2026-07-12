---
page_title: "incidentrelay_route Resource - IncidentRelay"
subcategory: "Alert Routing"
description: |-
  Manages an alert route.
---

# incidentrelay_route

```hcl
resource "incidentrelay_route" "alertmanager" {
  team_id              = incidentrelay_team.platform.id
  name                 = "platform-alertmanager"
  source               = "alertmanager"
  rotation_id          = incidentrelay_rotation.primary.id
  escalation_policy_id = incidentrelay_escalation_policy.critical.id
  channel_ids          = [incidentrelay_channel.email.id]

  matchers_json = jsonencode({
    labels = {
      team = "platform"
    }
  })

  group_by = ["alertname", "instance"]
}
```

The API returns `intake_token` only on create or regeneration. It is marked
sensitive in Terraform state.

## Import

```sh
terraform import incidentrelay_route.alertmanager 30
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_route).
