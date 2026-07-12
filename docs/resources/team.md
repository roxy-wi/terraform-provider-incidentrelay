# incidentrelay_team

Manages an IncidentRelay on-call team.

```hcl
resource "incidentrelay_team" "platform" {
  group_id = incidentrelay_group.infra.id
  slug     = "platform"
  name     = "Platform"
}
```

