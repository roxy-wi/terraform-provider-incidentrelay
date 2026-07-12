# incidentrelay_silence

Manages a temporary alert silence.

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

