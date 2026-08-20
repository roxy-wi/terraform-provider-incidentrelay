---
page_title: "incidentrelay_maintenance_window Resource - IncidentRelay"
subcategory: "Operations"
description: |-
  Manages a maintenance window.
---

# incidentrelay_maintenance_window

`scopes_json` is a JSON array of IncidentRelay maintenance scopes.

```hcl
resource "incidentrelay_maintenance_window" "deploy" {
  name      = "Platform deploy"
  starts_at = "2026-07-13T22:00:00"
  ends_at   = "2026-07-14T01:00:00"
  timezone  = "Europe/Moscow"

  apply_to_existing = true
  reactivate_on_end = true

  scopes_json = jsonencode([
    {
      scope_type = "team"
      team_id    = incidentrelay_team.platform.id
    }
  ])
}
```

`apply_to_existing` retroactively applies the maintenance behavior to matching
unresolved alerts. `reactivate_on_end` controls whether IncidentRelay resumes
alerts whose lifecycle was suppressed when the window ends. The API rejects
`apply_to_existing = true` with `behavior = "suppress_incident"`.

## Import

```sh
terraform import incidentrelay_maintenance_window.deploy 81
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_maintenance_window).
