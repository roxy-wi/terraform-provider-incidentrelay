---
page_title: "incidentrelay_business_service_component Resource - IncidentRelay"
subcategory: "Business Services"
description: |-
  Manages a technical service component inside a business service.
---

# incidentrelay_business_service_component

```hcl
resource "incidentrelay_business_service_component" "api" {
  business_service_id = incidentrelay_business_service.checkout.id
  service_id          = incidentrelay_service.api.id
  criticality         = "required"
  impact_weight       = 100
  status_rule         = "inherit"
}
```

## Import

Keep `business_service_id` in configuration before importing.

```sh
terraform import incidentrelay_business_service_component.api 101
```

See [resource reference](../guides/resource-reference.md#incidentrelay_business_service_component).
