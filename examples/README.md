# IncidentRelay Terraform Examples

These examples are intentionally small enough to copy into a real Terraform
configuration and adapt.

## Examples

- `provider`: minimal provider configuration and a small starter setup.
- `authentication`: token and username/password provider patterns.
- `core-oncall`: groups, users, teams, memberships, channels, rotations, routes,
  escalation, and notification policies.
- `service-catalog`: services, match rules, links, runbooks, dependencies,
  business services, and business service components.
- `maintenance`: silences and maintenance windows.
- `heartbeat`: heartbeat dead-man-switch configuration.
- `data-sources`: looking up existing groups, teams, users, services, and
  service version information.
- `imports`: Terraform 1.5+ import blocks and classic `terraform import`
  commands.

Set credentials with environment variables:

```sh
export INCIDENTRELAY_BASE_URL="https://incidentrelay.example.com"
export INCIDENTRELAY_TOKEN="..."
```
