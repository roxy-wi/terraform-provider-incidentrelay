---
page_title: "incidentrelay_rotation_override Resource - IncidentRelay"
subcategory: "On-call"
description: |-
  Manages a temporary override for an IncidentRelay rotation.
---

# incidentrelay_rotation_override

```hcl
resource "incidentrelay_rotation_override" "alice_cover" {
  rotation_id = incidentrelay_rotation.primary.id
  user_id     = incidentrelay_admin_user.alice.id
  starts_at   = "2026-07-14T09:00:00"
  ends_at     = "2026-07-14T13:00:00"
  reason      = "Alice covers the morning shift."
}
```

IncidentRelay rotation overrides are create/delete API objects. Changing
`rotation_id`, `user_id`, `starts_at`, `ends_at`, or `reason` recreates the
override.

Keep `rotation_id` in configuration before importing so the provider can read
the override from the rotation override list endpoint.

## Import

```sh
terraform import incidentrelay_rotation_override.alice_cover 43
```

See [resource reference](../guides/resource-reference.md#incidentrelay_rotation_override).
