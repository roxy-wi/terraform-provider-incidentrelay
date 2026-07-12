# incidentrelay_escalation_policy_rule

Manages a rule inside an escalation policy.

```hcl
resource "incidentrelay_escalation_policy_rule" "primary" {
  policy_id     = incidentrelay_escalation_policy.critical.id
  position      = 1
  delay_seconds = 300
  target_type   = "rotation"
  target_id     = incidentrelay_rotation.primary.id
}
```

