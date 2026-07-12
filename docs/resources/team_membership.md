---
page_title: "incidentrelay_team_membership Resource - IncidentRelay"
subcategory: "Teams"
description: |-
  Manages membership of an existing group user in an IncidentRelay team.
---

# incidentrelay_team_membership

```hcl
resource "incidentrelay_team_membership" "responder" {
  team_id = incidentrelay_team.platform.id
  user_id = incidentrelay_admin_user.alice.id
  role    = "responder"
}
```

## Import

Keep `team_id` in configuration before importing because the provider reads
memberships through the team user list endpoint.

```sh
terraform import incidentrelay_team_membership.responder 11
```

See [resource reference](../guides/resource-reference.md#incidentrelay_team_membership)
for all arguments and attributes.
