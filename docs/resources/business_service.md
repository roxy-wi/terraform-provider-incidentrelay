---
page_title: "incidentrelay_business_service Resource - IncidentRelay"
subcategory: "Business Services"
description: |-
  Manages an IncidentRelay business service.
---

# incidentrelay_business_service

```hcl
resource "incidentrelay_business_service" "checkout" {
  group_id           = incidentrelay_group.infra.id
  owner_team_id      = incidentrelay_team.platform.id
  slug               = "checkout"
  name               = "Checkout"
  criticality        = "critical"
  tier               = "tier_1"
  public             = true
  public_order       = 10
  public_name        = "Checkout"
  public_description = "Customer checkout path."
}
```

## Import

```sh
terraform import incidentrelay_business_service.checkout 100
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_business_service).
