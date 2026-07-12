# incidentrelay_escalation_policy

Manages an IncidentRelay escalation policy.

```hcl
resource "incidentrelay_escalation_policy" "critical" {
  team_id      = incidentrelay_team.platform.id
  name         = "Critical escalation"
  repeat_count = 1
}
```

