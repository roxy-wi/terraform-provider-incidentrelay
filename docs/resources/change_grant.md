---
page_title: "incidentrelay_change_grant Resource - IncidentRelay"
subcategory: "Change Intelligence"
description: |-
  Grants a Change source a capability on a Team or Service.
---

# incidentrelay_change_grant

Requires IR 2.5 and an [administrator JWT](../guides/authentication.md#change-integrations-ir-25).

```hcl
resource "incidentrelay_change_grant" "deploy" {
  integration_id    = incidentrelay_change_integration.pipeline.id
  source_id         = incidentrelay_change_source.deploy.id
  capability        = "changed"
  target_team_id    = incidentrelay_team.platform.id
  target_service_id = incidentrelay_service.api.id
}
```

- `integration_id` (required, replacement): parent integration.
- `source_id` (required): source receiving the grant. Moving to another source in the same integration preserves the grant ID.
- `capability` (required): `reference`, `changed`, or `reported_affected`.
- `target_team_id` (required): target Team, in scope for this integration.
- `target_service_id` (optional): restrict authority to one Service in the target Team. **Removing this attribute widens the grant to the entire Team**, including future Services.
- `active` (optional, default `true`): enable/disable the grant. IR creates grants active; Terraform rejects creating an inactive grant rather than briefly enabling it.
- `id` (computed): numeric grant ID.

Destroy deactivates the grant. IR retains the unique permission record: import it to manage or enable it again. Terraform reads the current parent revision before writing. A stale revision, duplicate grant or invalid identity is surfaced as an error; refresh and review the plan before retrying.

```sh
# integration_id/source_id/grant_id
terraform import incidentrelay_change_grant.deploy 12/34/56
```
