# incidentrelay_service

Manages a service catalog service.

```hcl
resource "incidentrelay_service" "api" {
  team_id = incidentrelay_team.platform.id
  slug    = "platform-api"
  name    = "Platform API"
}
```

