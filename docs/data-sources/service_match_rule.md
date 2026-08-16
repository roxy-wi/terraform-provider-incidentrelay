---
page_title: "incidentrelay_service_match_rule Data Source - IncidentRelay"
subcategory: "Service Catalog"
description: |-
  Looks up an existing IncidentRelay service match rule.
---

# incidentrelay_service_match_rule

```hcl
data "incidentrelay_service" "api" {
  team_id = incidentrelay_team.platform.id
  slug    = "platform-api"
}

data "incidentrelay_service_match_rule" "api_labels" {
  service_id = data.incidentrelay_service.api.id
  name       = "API labels"
}
```

## API Scope

The IncidentRelay API requires at least one scope for listing service match
rules. Configure one or more of:

- `team_id` (Number) Owner team ID.
- `route_id` (Number) Route ID.
- `service_id` (Number) Target service ID.

## Additional Lookup Arguments

- `match_rule_id` (Number) Service match rule ID.
- `name` (String) Service match rule name.

If the selected scope returns more than one matching rule, add
`match_rule_id` or `name`. Ambiguous lookups return an error.

## Attributes

- `id` (String) Service match rule ID.
- `match_rule_id` (Number) Service match rule ID.
- `team_id` (Number) Owner team ID.
- `team_slug` (String) Owner team slug.
- `team_name` (String) Owner team name.
- `route_id` (Number) Route ID, or `0` for an unscoped rule.
- `route_name` (String) Route name.
- `service_id` (Number) Target service ID.
- `service_slug` (String) Target service slug.
- `service_name` (String) Target service name.
- `position` (Number) Evaluation position.
- `name` (String) Rule name.
- `description` (String) Rule description.
- `matcher_preset_id` (Number) Matcher preset ID, or `0` when unset.
- `matchers_json` (String) Local matcher object encoded as JSON.
- `enabled` (Boolean) Whether the rule is enabled.
