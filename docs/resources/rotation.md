---
page_title: "incidentrelay_rotation Resource - IncidentRelay"
subcategory: "On-call"
description: |-
  Manages an IncidentRelay on-call rotation.
---

# incidentrelay_rotation

```hcl
resource "incidentrelay_rotation" "primary" {
  team_id          = incidentrelay_team.platform.id
  name             = "Primary"
  start_at         = "2026-07-13T09:00:00"
  rotation_type    = "weekly"
  interval_value   = 1
  interval_unit    = "weeks"
  handoff_time     = "09:00"
  handoff_weekday  = 0
  timezone         = "Europe/Moscow"
  add_team_members = false
}
```

## Import

```sh
terraform import incidentrelay_rotation.primary 40
```

See [resource reference](../guides/resource-reference.md#incidentrelay_rotation)
for all arguments and attributes.
