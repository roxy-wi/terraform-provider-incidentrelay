---
page_title: "incidentrelay_admin_user Resource - IncidentRelay"
subcategory: "Identity"
description: |-
  Manages a local IncidentRelay user through the admin API.
---

# incidentrelay_admin_user

Use `password` only on create or when rotating a password; it is marked
sensitive in Terraform state.

```hcl
resource "incidentrelay_admin_user" "alice" {
  username     = "alice"
  display_name = "Alice Responder"
  email        = "alice@example.com"
  active       = true
  group_id     = incidentrelay_group.infra.id
  group_role   = "user_admin"
}
```

## Import

```sh
terraform import incidentrelay_admin_user.alice 2
```

See [resource reference](../guides/resource-reference.md#incidentrelay_admin_user)
for all arguments and attributes.
