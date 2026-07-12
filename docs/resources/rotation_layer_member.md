---
page_title: "incidentrelay_rotation_layer_member Resource - IncidentRelay"
subcategory: "On-call"
description: |-
  Manages a member period in a rotation layer.
---

# incidentrelay_rotation_layer_member

```hcl
resource "incidentrelay_rotation_layer_member" "alice" {
  layer_id = incidentrelay_rotation_layer.business_hours.id
  user_id  = incidentrelay_admin_user.alice.id
  position = 0
}
```

## Import

Keep `layer_id` in configuration before importing.

```sh
terraform import incidentrelay_rotation_layer_member.alice 42
```

See [resource reference](../guides/resource-reference.md#incidentrelay_rotation_layer_member).
