---
page_title: "incidentrelay_silence Resource - IncidentRelay"
subcategory: "Operations"
description: |-
  Manages a temporary alert silence.
---

# incidentrelay_silence

```hcl
resource "incidentrelay_silence" "maintenance" {
  team_id   = incidentrelay_team.platform.id
  name      = "Planned maintenance"
  starts_at = "2026-07-13T22:00:00"
  ends_at   = "2026-07-14T01:00:00"

  apply_to_existing = true
  reactivate_on_end = true

  matchers_json = jsonencode({
    labels = {
      environment = "production"
    }
  })
}
```

`matcher_preset_id` can select a reusable IncidentRelay matcher preset instead
of, or together with, `matchers_json`. `apply_to_existing` retroactively
suppresses matching unresolved alerts. `reactivate_on_end` controls whether
IncidentRelay resumes alerts suppressed by this silence when the silence ends.
The `enabled` argument uses the IncidentRelay 2.0 enable/disable lifecycle API.

## Import

```sh
terraform import incidentrelay_silence.maintenance 80
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_silence).
