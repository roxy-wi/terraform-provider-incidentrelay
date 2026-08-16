---
page_title: "Importing Existing IncidentRelay Resources"
subcategory: "Guides"
description: |-
  Import existing IncidentRelay resources into Terraform state.
---

# Importing Existing Resources

Every IncidentRelay resource supports Terraform import. The import ID is the
IncidentRelay numeric API ID unless noted otherwise.

Terraform 1.5+ import blocks are recommended because they make imports
reviewable in code:

```hcl
import {
  to = incidentrelay_group.infra
  id = "1"
}
```

Classic CLI imports also work:

```sh
terraform import incidentrelay_group.infra 1
```

## Recommended Workflow

1. Write the resource block with the desired Terraform address.
2. Add the import block or run `terraform import`.
3. Run `terraform plan`.
4. Fill in missing required arguments until the plan is empty or only contains
   intended changes.
5. Remove temporary import blocks after the import has been applied.

For child resources that are read through a parent list endpoint, keep the
parent ID in configuration before importing. Examples include memberships,
rotation layers, rotation layer members, rotation overrides, escalation policy
rules, notification policy rules, SSO group mappings, business service
components, priority policy rules, and service child resources.

## Import IDs

| Resource | Import ID | Notes |
| --- | --- | --- |
| `incidentrelay_group` | Group ID | Reads from `/api/groups`. |
| `incidentrelay_admin_user` | User ID | Password is not read back from the API. |
| `incidentrelay_group_membership` | Membership ID | Configure `group_id` before import. |
| `incidentrelay_sso_provider` | SSO provider ID | Secrets are not read back from the API. |
| `incidentrelay_sso_group_mapping` | Mapping ID | Configure `provider_id` before import. |
| `incidentrelay_team` | Team ID | Direct read. |
| `incidentrelay_team_membership` | Membership ID | Configure `team_id` before import. |
| `incidentrelay_channel` | Channel ID | Direct read. |
| `incidentrelay_route` | Route ID | Intake token is only returned on create/regeneration. |
| `incidentrelay_rotation` | Rotation ID | Direct read. |
| `incidentrelay_rotation_layer` | Layer ID | Configure `rotation_id` before import. |
| `incidentrelay_rotation_layer_member` | Member ID | Configure `layer_id` before import. |
| `incidentrelay_rotation_override` | Override ID | Configure `rotation_id` before import. |
| `incidentrelay_escalation_policy` | Policy ID | Direct read. |
| `incidentrelay_escalation_policy_rule` | Rule ID | Configure `policy_id` before import. |
| `incidentrelay_priority_policy` | Policy ID | Direct read. `team_id` cannot be changed. |
| `incidentrelay_priority_policy_rule` | `policy_id/rule_id` | Parent policy and rule IDs are required by the nested API. |
| `incidentrelay_notification_policy` | Policy ID | Direct read. |
| `incidentrelay_notification_policy_rule` | Rule ID | Configure `policy_id` before import. |
| `incidentrelay_service` | Service ID | Direct read. |
| `incidentrelay_service_match_rule` | Rule ID | Configure `service_id` before import. |
| `incidentrelay_service_link` | Link ID | Configure `service_id` before import. |
| `incidentrelay_service_runbook` | Runbook ID | Configure `service_id` before import. |
| `incidentrelay_service_dependency` | Dependency ID | Configure `service_id` before import. |
| `incidentrelay_silence` | Silence ID | Direct read. |
| `incidentrelay_maintenance_window` | Maintenance window ID | Direct read. |
| `incidentrelay_heartbeat` | Heartbeat ID | Token and ping URL are only returned on create/regeneration. |
| `incidentrelay_business_service` | Business service ID | Direct read. |
| `incidentrelay_business_service_component` | Component ID | Configure `business_service_id` before import. |

See [examples/imports/main.tf](../../examples/imports/main.tf) for import block
examples.
