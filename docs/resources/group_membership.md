---
page_title: "incidentrelay_group_membership Resource - IncidentRelay"
subcategory: "Identity"
description: |-
  Manages membership of an existing user in an IncidentRelay group.
---

# incidentrelay_group_membership

```hcl
resource "incidentrelay_group_membership" "viewer" {
  group_id = incidentrelay_group.infra.id
  user_id  = incidentrelay_admin_user.alice.id
  role     = "viewer"
}
```

## Import

Keep `group_id` in configuration before importing because the provider reads
memberships through the group user list endpoint.

```sh
terraform import incidentrelay_group_membership.viewer 3
```

See [resource reference](../guides/resource-reference.md#incidentrelay_group_membership)
for all arguments and attributes.
