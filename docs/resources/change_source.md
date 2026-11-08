---
page_title: "incidentrelay_change_source Resource - IncidentRelay"
subcategory: "Change Intelligence"
description: |-
  Manages a producer namespace on a Change Integration.
---

# incidentrelay_change_source

Requires IR 2.5 and the same [administrator JWT](../guides/authentication.md#change-integrations-ir-25) as the parent integration.

```hcl
resource "incidentrelay_change_source" "deploy" {
  integration_id = incidentrelay_change_integration.pipeline.id
  owner_team_id  = incidentrelay_team.platform.id
  source         = "production-deploys"
  ordering_mode  = "sequence"
}
```

- `integration_id` (required, replacement): parent integration.
- `owner_team_id`, `source` (required): the Team and producer namespace. The name is 1–255 characters. IR allows changing either value only before the first Change is recorded.
- `ordering_mode` (optional): `timestamp` (default) or `sequence`.
- `active` (optional, default `true`): a newly created source is active.
- `id`, `identity_editable` (computed): numeric source ID and whether IR still permits editing its identity.

Destroy deactivates the source. Its namespace remains reserved, including after deactivation. Import it to reuse it; use a fresh name to create another source. To grant access to the integration's first source without managing that source separately, use the integration's `initial_source_id`.

```sh
# integration_id/source_id
terraform import incidentrelay_change_source.deploy 12/34
```
