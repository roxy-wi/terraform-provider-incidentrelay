---
page_title: "incidentrelay_group Resource - IncidentRelay"
subcategory: "Identity"
description: |-
  Manages an IncidentRelay access group.
---

# incidentrelay_group

```hcl
resource "incidentrelay_group" "infra" {
  slug        = "infra"
  name        = "Infrastructure"
  description = "Infrastructure access boundary"
  active      = true
}
```

## Import

```sh
terraform import incidentrelay_group.infra 1
```

See [resource reference](../guides/resource-reference.md#incidentrelay_group)
for all arguments and attributes.
