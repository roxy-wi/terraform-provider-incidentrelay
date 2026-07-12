# incidentrelay_channel

Manages an outbound notification channel.

```hcl
resource "incidentrelay_channel" "email" {
  team_id      = incidentrelay_team.platform.id
  name         = "platform-email"
  channel_type = "email"
  config_json  = jsonencode({})
}
```

