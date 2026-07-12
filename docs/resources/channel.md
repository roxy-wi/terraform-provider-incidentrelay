---
page_title: "incidentrelay_channel Resource - IncidentRelay"
subcategory: "Notifications"
description: |-
  Manages an outbound notification channel.
---

# incidentrelay_channel

```hcl
resource "incidentrelay_channel" "email" {
  team_id      = incidentrelay_team.platform.id
  name         = "platform-email"
  channel_type = "email"
  config_json  = jsonencode({})
}
```

## Import

```sh
terraform import incidentrelay_channel.email 20
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_channel).
