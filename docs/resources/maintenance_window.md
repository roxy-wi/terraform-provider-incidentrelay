# incidentrelay_maintenance_window

Manages a maintenance window.

`scopes_json` is a JSON array of IncidentRelay maintenance scopes.

```hcl
resource "incidentrelay_maintenance_window" "deploy" {
  name      = "Platform deploy"
  starts_at = "2026-07-13T22:00:00"
  ends_at   = "2026-07-14T01:00:00"
  timezone  = "Europe/Moscow"

  scopes_json = jsonencode([
    {
      scope_type = "team"
      team_id    = incidentrelay_team.platform.id
    }
  ])
}
```

