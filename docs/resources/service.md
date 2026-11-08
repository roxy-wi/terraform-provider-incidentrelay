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

  priority_policy_id = incidentrelay_priority_policy.production.id

  labels_json = jsonencode({
    owner = "platform"
  })
}
```

## Change correlation

For IR 2.5+, set the following arguments on the Service to opt its alert
episodes into automatic Change suggestions:

```hcl
change_correlation_enabled         = true
change_correlation_exclusions_json = jsonencode([
  { source = "heartbeat" },
  { labels = { environment = "test" } }
])
```

The exclusions are a JSON array of up to 50 rules. Each rule needs a source
or at least one string label; all clauses within one rule must match the same
child alert. Omit either attribute to leave its current server value unmanaged.
Use `false` to disable correlation and `jsonencode([])` to clear the rules.
Existing configurations can still manage Services on pre-2.5 servers.

## Import

```sh
terraform import incidentrelay_service.api 70
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_service).
