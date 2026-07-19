---
page_title: "incidentrelay_channel Resource - IncidentRelay"
subcategory: "Notifications"
description: |-
  Manages an outbound notification channel.
---

# incidentrelay_channel

```hcl
resource "incidentrelay_channel" "email" {
  team_id      = incidentrelay_team.platform.id
  name         = "platform-email"
  channel_type = "email"
  config_json  = jsonencode({})
}
```

IncidentRelay 1.2 supports Slack Bot API actions over HTTP or Socket Mode. The
Socket Mode configuration is useful when IncidentRelay cannot accept inbound
connections from Slack:

```hcl
variable "slack_bot_token" {
  type      = string
  sensitive = true
}

variable "slack_app_token" {
  type      = string
  sensitive = true
}

resource "incidentrelay_channel" "slack_socket" {
  team_id      = incidentrelay_team.platform.id
  name         = "platform-slack"
  channel_type = "slack"

  config_json = jsonencode({
    mode            = "bot_api"
    connection_mode = "socket_mode"
    bot_token       = var.slack_bot_token
    app_token       = var.slack_app_token
    channel_id      = "C0123456789"
  })
}
```

For HTTP actions, set `connection_mode = "http"` and provide
`signing_secret` instead of `app_token`.

`config_json` is sensitive. IncidentRelay 1.2 returns placeholders instead of
stored Slack secrets; the provider keeps the configured secret values during
refresh to prevent a permanent diff. After importing an existing Slack channel,
set its real secrets in configuration before applying changes because masked API
responses cannot recover them.

## Import

```sh
terraform import incidentrelay_channel.email 20
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_channel).
