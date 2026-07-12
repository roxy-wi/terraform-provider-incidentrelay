# incidentrelay_route

Manages an alert route.

```hcl
resource "incidentrelay_route" "alertmanager" {
  team_id     = incidentrelay_team.platform.id
  name        = "platform-alertmanager"
  source      = "alertmanager"
  channel_ids = [incidentrelay_channel.email.id]
}
```

