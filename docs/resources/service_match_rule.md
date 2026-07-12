---
page_title: "incidentrelay_service_match_rule Resource - IncidentRelay"
subcategory: "Service Catalog"
description: |-
  Manages a service match rule that maps incoming alerts to a service.
---

# incidentrelay_service_match_rule

```hcl
resource "incidentrelay_service_match_rule" "api" {
  team_id    = incidentrelay_team.platform.id
  service_id = incidentrelay_service.api.id
  name       = "API labels"

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
terraform import incidentrelay_service_match_rule.api 71
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_service_match_rule).
