# incidentrelay_service_match_rule

Manages a service match rule that maps incoming alerts to a service.

```hcl
resource "incidentrelay_service_match_rule" "api" {
  team_id    = incidentrelay_team.platform.id
  service_id = incidentrelay_service.api.id
  name       = "API labels"

  matchers_json = jsonencode({
    labels = {
      service = "platform-api"
    }
  })
}
```

