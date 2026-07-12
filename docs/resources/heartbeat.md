---
page_title: "incidentrelay_heartbeat Resource - IncidentRelay"
subcategory: "Operations"
description: |-
  Manages a heartbeat dead-man-switch check.
---

# incidentrelay_heartbeat

```hcl
resource "incidentrelay_heartbeat" "api_cron" {
  team_id              = incidentrelay_team.platform.id
  route_id             = incidentrelay_route.heartbeat.id
  service_id           = incidentrelay_service.api.id
  name                 = "API nightly cron"
  slug                 = "platform-api-nightly-cron"
  mode                 = "scheduled"
  schedule_kind        = "daily"
  schedule_time        = "02:30"
  timezone             = "Europe/Moscow"
  grace_period_seconds = 900
  severity             = "critical"
  priority_slug        = "p1"
}
```

The API returns `token` and `ping_url` only once on creation or token
regeneration. Terraform stores those computed values as sensitive.

## Import

```sh
terraform import incidentrelay_heartbeat.api_cron 90
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_heartbeat).
