---
page_title: "incidentrelay_service_dependency Resource - IncidentRelay"
subcategory: "Service Catalog"
description: |-
  Manages an upstream service dependency.
---

# incidentrelay_service_dependency

```hcl
resource "incidentrelay_service_dependency" "database" {
  service_id                = incidentrelay_service.api.id
  depends_on_service_id     = incidentrelay_service.database.id
  dependency_type           = "hard"
  criticality               = "required"
  propagation_delay_seconds = 120
  correlation_enabled       = true
}
```

## Import

Keep `service_id` in configuration before importing.

```sh
terraform import incidentrelay_service_dependency.database 74
```

See [resource reference](../guides/resource-reference.md#incidentrelay_service_dependency).
