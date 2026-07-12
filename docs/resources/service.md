---
page_title: "incidentrelay_service Resource - IncidentRelay"
subcategory: "Service Catalog"
description: |-
  Manages a service catalog service.
---

# incidentrelay_service

```hcl
resource "incidentrelay_service" "api" {
  team_id      = incidentrelay_team.platform.id
  slug         = "platform-api"
  name         = "Platform API"
  service_type = "api"
  environment  = "production"
  criticality  = "high"
  tier         = "tier_2"

  labels_json = jsonencode({
    owner = "platform"
  })
}
```

## Import

```sh
terraform import incidentrelay_service.api 70
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_service).
