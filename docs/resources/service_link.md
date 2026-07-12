---
page_title: "incidentrelay_service_link Resource - IncidentRelay"
subcategory: "Service Catalog"
description: |-
  Manages a link attached to a service catalog service.
---

# incidentrelay_service_link

```hcl
resource "incidentrelay_service_link" "api_dashboard" {
  service_id = incidentrelay_service.api.id
  link_type  = "dashboard"
  label      = "Grafana"
  url        = "https://grafana.example.com/d/platform-api"
  priority   = 10
  enabled    = true
}
```

## Import

Keep `service_id` in configuration before importing.

```sh
terraform import incidentrelay_service_link.api_dashboard 72
```

See [resource reference](../guides/resource-reference.md#incidentrelay_service_link).
