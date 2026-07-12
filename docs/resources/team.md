---
page_title: "incidentrelay_team Resource - IncidentRelay"
subcategory: "Teams"
description: |-
  Manages an IncidentRelay on-call team.
---

# incidentrelay_team

```hcl
resource "incidentrelay_team" "platform" {
  group_id = incidentrelay_group.infra.id
  slug     = "platform"
  name     = "Platform"
}
```

## Import

```sh
terraform import incidentrelay_team.platform 10
```

See [resource reference](../guides/resource-reference.md#incidentrelay_team)
for all arguments and attributes.
