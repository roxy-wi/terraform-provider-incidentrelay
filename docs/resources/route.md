---
page_title: "incidentrelay_route Resource - IncidentRelay"
subcategory: "Alert Routing"
description: |-
  Manages an alert route.
---

# incidentrelay_route

```hcl
resource "incidentrelay_route" "alertmanager" {
  team_id              = incidentrelay_team.platform.id
  name                 = "platform-alertmanager"
  source               = "alertmanager"
  rotation_id          = incidentrelay_rotation.primary.id
  escalation_policy_id = incidentrelay_escalation_policy.critical.id
  channel_ids          = [incidentrelay_channel.email.id]

  matchers_json = jsonencode({
    labels = {
      team = "platform"
    }
  })

  group_by = ["alertname", "instance"]
}
```

When `escalation_policy_id` is set, the provider sends the API route in policy
escalation mode automatically.

The API returns `intake_token` only on create or regeneration. It is marked
sensitive in Terraform state.

IncidentRelay 1.2 adds Datadog as an incoming source:

```hcl
resource "incidentrelay_route" "datadog" {
  team_id     = incidentrelay_team.platform.id
  name        = "platform-datadog"
  source      = "datadog"
  channel_ids = [incidentrelay_channel.slack_socket.id]

  matchers_json           = jsonencode({})
  integration_config_json = jsonencode({})
}
```

Send Datadog webhooks to the route-specific IncidentRelay Datadog intake
endpoint using the one-time intake token returned when the route is created.

IncidentRelay 2.0 also supports Uptime Kuma's standard webhook payload:

```hcl
resource "incidentrelay_route" "uptime_kuma" {
  team_id = incidentrelay_team.platform.id
  name    = "platform-uptime-kuma"
  source  = "uptime_kuma"
}
```

IncidentRelay 2.1-2.3 add `new_relic`, `nagios`, `azure_monitor`, and
`cloud_ru`. Cloud.ru Advanced SMN requires a topic URN:

```hcl
resource "incidentrelay_route" "cloud_ru" {
  team_id = incidentrelay_team.platform.id
  name    = "platform-cloud-ru"
  source  = "cloud_ru"

  matcher_preset_id = var.matcher_preset_id

  integration_config_json = jsonencode({
    cloud_ru = {
      topic_urn = "urn:smn:ru-a:project-id:incidentrelay"
    }
  })
}
```

The route-specific `webhook_path` returned by IncidentRelay is computed server
state. The provider removes it from managed `integration_config_json` during
refresh, so it does not create a permanent Terraform diff. The same
normalization applies to AWS SNS and Sentry integration metadata.

## OpenSearch and Kibana (IncidentRelay 2.4+)

```hcl
resource "incidentrelay_route" "opensearch" {
  team_id = incidentrelay_team.platform.id
  name    = "platform-opensearch"
  source  = "opensearch"
  group_by = [
    "opensearch_monitor_id", "opensearch_trigger_id", "opensearch_bucket_key"
  ]
}

resource "incidentrelay_route" "kibana" {
  team_id  = incidentrelay_team.platform.id
  name     = "platform-kibana"
  source   = "kibana"
  group_by = ["kibana_rule_id", "kibana_alert_id"]
}
```

Send events to `POST /api/integrations/opensearch` or
`POST /api/integrations/kibana` with the corresponding route's `intake_token`
as a Bearer token. Neither source needs extra `integration_config_json` fields.
The provider manages the IncidentRelay route; configure the sending connector
in OpenSearch or Kibana separately.

## Import

```sh
terraform import incidentrelay_route.alertmanager 30
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_route).
