# Classic Terraform Import Commands

Terraform 1.5+ import blocks are preferred for reviewable imports, but classic
CLI imports are supported too.

```sh
terraform import incidentrelay_group.infra 1
terraform import incidentrelay_admin_user.alice 2
terraform import incidentrelay_group_membership.alice 3
terraform import incidentrelay_team.platform 10
terraform import incidentrelay_team_membership.alice 11
terraform import incidentrelay_channel.email 20
terraform import incidentrelay_route.alertmanager 30
terraform import incidentrelay_rotation.primary 40
terraform import incidentrelay_rotation_layer.business_hours 41
terraform import incidentrelay_rotation_layer_member.alice 42
terraform import incidentrelay_escalation_policy.critical 50
terraform import incidentrelay_escalation_policy_rule.primary 51
terraform import incidentrelay_notification_policy.production 60
terraform import incidentrelay_notification_policy_rule.critical_email 61
terraform import incidentrelay_service.api 70
terraform import incidentrelay_service_match_rule.api_labels 71
terraform import incidentrelay_service_link.api_dashboard 72
terraform import incidentrelay_service_runbook.api_critical 73
terraform import incidentrelay_service_dependency.api_database 74
terraform import incidentrelay_silence.deploy 80
terraform import incidentrelay_maintenance_window.deploy 81
terraform import incidentrelay_heartbeat.api_cron 90
terraform import incidentrelay_business_service.checkout 100
terraform import incidentrelay_business_service_component.checkout_api 101
```
