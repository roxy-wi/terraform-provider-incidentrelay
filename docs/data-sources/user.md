---
page_title: "incidentrelay_user Data Source - IncidentRelay"
subcategory: "Identity"
description: |-
  Looks up an IncidentRelay user by ID, username, or email.
---

# incidentrelay_user

```hcl
data "incidentrelay_user" "alice" {
  username = "alice"
}
```

## Lookup Arguments

- `user_id` (Number) User ID.
- `username` (String) Username.
- `email` (String) Email address.

At least one lookup argument is required. If more than one user matches, add
more filters.

## Attributes

- `id` (String) User ID.
- `username` (String) Username.
- `display_name` (String) Display name.
- `email` (String) Email address.
- `phone` (String) Phone number.
- `telegram_user_id` (String) Telegram user ID.
- `slack_user_id` (String) Slack user ID.
- `mattermost_user_id` (String) Mattermost user ID.
