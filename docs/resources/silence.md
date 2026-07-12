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

  matchers_json = jsonencode({
    labels = {
      environment = "production"
    }
  })
}
```

## Import

```sh
terraform import incidentrelay_silence.maintenance 80
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_silence).
