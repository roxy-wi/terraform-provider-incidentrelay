# IncidentRelay Terraform Examples

These examples are intentionally small enough to copy into a real Terraform
configuration and adapt.

## Examples

- `provider`: minimal provider configuration and a small starter setup.
- `authentication`: token and username/password provider patterns.
- `core-oncall`: groups, users, teams, memberships, channels, rotations, routes,
  escalation, incident priority, and notification policies.
- `service-catalog`: services, match rules, links, runbooks, dependencies,
  business services, and business service components.
- `maintenance`: silences and maintenance windows.
- `heartbeat`: heartbeat dead-man-switch configuration.
- `incidentrelay-1.2`: Datadog routing and Slack Socket Mode with masked-secret
  refresh compatibility.
- `incidentrelay-2.0`: Event Orchestration, reusable webhook actions, and an
  Uptime Kuma route.
- `sso`: OIDC provider configuration and external group mapping.
- `data-sources`: looking up existing groups, teams, users, notification
  channels, services, rotations, incident priorities, escalation and
  notification policies, service match rules, and service version information.
- `imports`: Terraform 1.5+ import blocks and classic `terraform import`
  commands.

Set credentials with environment variables:

```sh
export INCIDENTRELAY_BASE_URL="https://incidentrelay.example.com"
export INCIDENTRELAY_TOKEN="..."
```
