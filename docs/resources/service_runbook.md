---
page_title: "incidentrelay_service_runbook Resource - IncidentRelay"
subcategory: "Service Catalog"
description: |-
  Manages a runbook attached to a service catalog service.
---

# incidentrelay_service_runbook

```hcl
resource "incidentrelay_service_runbook" "api_critical" {
  service_id = incidentrelay_service.api.id
  title      = "API critical alert runbook"
  url        = "https://runbooks.example.com/platform-api/critical"
  severity   = "critical"
  priority   = 10

  matchers_json = jsonencode({
    labels = {
      service = "platform-api"
    }
  })
}
```

## Import

Keep `service_id` in configuration before importing.

```sh
terraform import incidentrelay_service_runbook.api_critical 73
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_service_runbook).
