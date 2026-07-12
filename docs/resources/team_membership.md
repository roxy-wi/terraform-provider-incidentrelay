# incidentrelay_team_membership

Manages membership of an existing group user in an IncidentRelay team.

```hcl
resource "incidentrelay_team_membership" "responder" {
  team_id = incidentrelay_team.platform.id
  user_id = incidentrelay_admin_user.alice.id
  role    = "responder"
}
```

