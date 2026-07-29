---
page_title: "incidentrelay_sso_group_mapping Resource - IncidentRelay"
subcategory: "SSO"
description: |-
  Maps an external identity-provider group to an IncidentRelay group and optional team.
---

# incidentrelay_sso_group_mapping

```hcl
resource "incidentrelay_group" "infrastructure" {
  slug = "infrastructure"
  name = "Infrastructure"
}

resource "incidentrelay_team" "platform" {
  group_id = incidentrelay_group.infrastructure.id
  slug     = "platform"
  name     = "Platform"
}

resource "incidentrelay_sso_group_mapping" "platform" {
  provider_id    = incidentrelay_sso_provider.corporate.id
  external_group = "IncidentRelay-Platform"

  group_id   = incidentrelay_group.infrastructure.id
  group_role = "editor"

  team_id   = incidentrelay_team.platform.id
  team_role = "responder"

  priority = 100
  active   = true
}
```

When the configured SSO provider reports `IncidentRelay-Platform` in the
provider's `groups_claim`, IncidentRelay adds or updates the user's group and
team memberships according to this mapping.

The optional team must belong to the selected IncidentRelay group.

## Schema

### Required

- `provider_id` (Number) Parent SSO provider ID. Changing it recreates the
  mapping.
- `external_group` (String) External identity-provider group name.
- `group_id` (Number) IncidentRelay group ID.

### Optional

- `group_role` (String) `viewer`, `editor`, `user_admin`, or `global_admin`.
  Defaults to `viewer`.
- `team_id` (Number) Optional IncidentRelay team ID.
- `team_role` (String) `viewer`, `responder`, or `manager`. When `team_id` is
  set and `team_role` is omitted, IncidentRelay defaults it to `viewer`.
- `active` (Boolean) Whether the mapping is active. Defaults to `true`.
- `priority` (Number) Mapping priority from 0 to 10000. Lower values are
  evaluated first. Defaults to 100.

### Read-Only

- `group_slug` (String) IncidentRelay group slug.
- `group_name` (String) IncidentRelay group name.
- `team_slug` (String) IncidentRelay team slug.
- `team_name` (String) IncidentRelay team name.

## Import

Keep `provider_id` in the resource configuration, then import with the numeric
mapping ID:

```sh
terraform import incidentrelay_sso_group_mapping.platform 34
```
