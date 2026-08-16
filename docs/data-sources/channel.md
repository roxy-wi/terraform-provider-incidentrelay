---
page_title: "incidentrelay_channel Data Source - IncidentRelay"
subcategory: "Notifications"
description: |-
  Looks up an existing IncidentRelay notification channel.
---

# incidentrelay_channel

```hcl
data "incidentrelay_channel" "email" {
  group_slug   = "infrastructure"
  team_slug    = "platform"
  name         = "platform-email"
  channel_type = "email"
}

resource "incidentrelay_notification_policy_rule" "critical_email" {
  policy_id   = incidentrelay_notification_policy.production.id
  name        = "Critical alerts to email"
  channel_ids = [data.incidentrelay_channel.email.id]
}
```

## Lookup Arguments

- `channel_id` (Number) Notification channel ID.
- `group_id` (Number) Owner group ID.
- `group_slug` (String) Owner group slug.
- `team_id` (Number) Owner team ID.
- `team_slug` (String) Owner team slug.
- `name` (String) Notification channel name.
- `channel_type` (String) Notification channel type.

At least one lookup argument is required. Prefer `channel_id`, or combine the
group, team, and channel name when names may be duplicated. Ambiguous lookups
return an error.

## Attributes

- `id` (String) Notification channel ID.
- `channel_id` (Number) Notification channel ID.
- `group_id` (Number) Owner group ID.
- `group_slug` (String) Owner group slug.
- `group_name` (String) Owner group name.
- `team_id` (Number) Owner team ID.
- `team_slug` (String) Owner team slug.
- `team_name` (String) Owner team name.
- `name` (String) Notification channel name.
- `channel_type` (String) Notification channel type.
- `enabled` (Boolean) Whether the notification channel is enabled.

The data source deliberately does not expose `config_json`. IncidentRelay masks
stored channel secrets, so API responses cannot reconstruct a usable channel
configuration.
