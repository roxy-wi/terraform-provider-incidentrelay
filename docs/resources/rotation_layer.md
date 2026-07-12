---
page_title: "incidentrelay_rotation_layer Resource - IncidentRelay"
subcategory: "On-call"
description: |-
  Manages a schedule layer inside an IncidentRelay rotation.
---

# incidentrelay_rotation_layer

```hcl
resource "incidentrelay_rotation_layer" "business_hours" {
  rotation_id   = incidentrelay_rotation.primary.id
  name          = "Business hours"
  priority      = 100
  start_at      = "2026-07-13T09:00:00"
  rotation_type = "weekly"
  interval_unit = "weeks"
  handoff_time  = "09:00"
  timezone      = "Europe/Moscow"
}
```

Layer restrictions are intentionally not modeled as a separate Terraform
resource yet; use the IncidentRelay API/UI for advanced restriction windows.

## Import

Keep `rotation_id` in configuration before importing.

```sh
terraform import incidentrelay_rotation_layer.business_hours 41
```

See [resource reference](../guides/resource-reference.md#incidentrelay_rotation_layer).
