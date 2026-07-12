# incidentrelay_group_membership

Manages membership of an existing user in an IncidentRelay group.

```hcl
resource "incidentrelay_group_membership" "viewer" {
  group_id = incidentrelay_group.infra.id
  user_id  = incidentrelay_admin_user.alice.id
  role     = "viewer"
}
```

