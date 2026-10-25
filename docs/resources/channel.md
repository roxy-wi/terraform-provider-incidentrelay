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

## Email recipients (IncidentRelay 2.4+)

An empty `config_json` retains the legacy behavior: send to the assigned user's
profile email. An explicit `recipient_mode = "assignee"` means the same thing.
IR 2.4 also accepts `fallback` (use the configured address if the assignee has no
email) and `fixed` (always use the configured address):

```hcl
resource "incidentrelay_channel" "email_fallback" {
  team_id      = incidentrelay_team.platform.id
  name         = "platform-email-fallback"
  channel_type = "email"
  config_json = jsonencode({
    recipient_mode  = "fallback"
    recipient_email = "oncall@example.com"
  })
}
```

Use exactly one mailbox in `recipient_email`, without a display name or address
list. It is required for `fallback` and `fixed`, and ignored/removed by the API
in `assignee` mode. Set the mode to `assignee` or return to `{}` to switch back.
SMTP transport remains configured on the IncidentRelay server. The UI/API test
action always sends to the current user's profile email, regardless of mode.

The provider treats API-normalized email addresses and the omitted/explicit
`assignee` default as equivalent during refresh and planning. Other changes,
including a different recipient, mode, template or severity filter, remain
visible. Ordinary mailbox local-part case is preserved; the API lowercases
RFC 2142 role mailbox names such as `support` and `postmaster`.

## Slack

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

IncidentRelay 2.3 also supports Feishu/Lark custom bots:

```hcl
variable "lark_webhook_url" {
  type      = string
  sensitive = true
}

variable "lark_signing_secret" {
  type      = string
  sensitive = true
}

resource "incidentrelay_channel" "lark" {
  team_id      = incidentrelay_team.platform.id
  name         = "platform-lark"
  channel_type = "lark"

  config_json = jsonencode({
    webhook_url    = var.lark_webhook_url
    signing_secret = var.lark_signing_secret
  })
}
```

`config_json` is sensitive. IncidentRelay returns placeholders instead of
stored channel secrets; the provider keeps configured values during refresh to
prevent a permanent diff. After importing a channel, set its real secrets in
configuration before applying changes because masked API responses cannot
recover them.

## Import

```sh
terraform import incidentrelay_channel.email 20
```

See [JSON fields guide](../guides/json-fields.md) and
[resource reference](../guides/resource-reference.md#incidentrelay_channel).
