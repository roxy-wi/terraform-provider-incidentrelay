# Terraform Provider for IncidentRelay

Terraform provider for managing IncidentRelay configuration as code.

The provider talks to the IncidentRelay HTTP API and supports both personal/API
Bearer tokens and username/password login.

## Requirements

- Terraform 1.7+
- Go 1.22.5+ to build locally
- IncidentRelay API access with permissions for the resources you manage

## Build

```sh
make build
```

The binary is written to `bin/terraform-provider-incidentrelay`.

## Provider Configuration

```hcl
terraform {
  required_providers {
    incidentrelay = {
      source  = "incidentrelay/incidentrelay"
      version = "0.1.0"
    }
  }
}

provider "incidentrelay" {
  base_url = "https://incidentrelay.example.com"
  token    = var.incidentrelay_token
}
```

Environment variables are also supported:

- `INCIDENTRELAY_BASE_URL`
- `INCIDENTRELAY_TOKEN`
- `INCIDENTRELAY_USERNAME`
- `INCIDENTRELAY_PASSWORD`
- `INCIDENTRELAY_INSECURE_SKIP_TLS_VERIFY`

Prefer `INCIDENTRELAY_TOKEN` for CI and automation. `username` and `password`
are useful for local development.

## Supported Resources

- `incidentrelay_group`
- `incidentrelay_admin_user`
- `incidentrelay_group_membership`
- `incidentrelay_team`
- `incidentrelay_team_membership`
- `incidentrelay_channel`
- `incidentrelay_route`
- `incidentrelay_rotation`
- `incidentrelay_rotation_layer`
- `incidentrelay_rotation_layer_member`
- `incidentrelay_escalation_policy`
- `incidentrelay_escalation_policy_rule`
- `incidentrelay_notification_policy`
- `incidentrelay_notification_policy_rule`
- `incidentrelay_service`
- `incidentrelay_service_match_rule`
- `incidentrelay_service_link`
- `incidentrelay_service_runbook`
- `incidentrelay_service_dependency`
- `incidentrelay_silence`
- `incidentrelay_maintenance_window`
- `incidentrelay_heartbeat`
- `incidentrelay_business_service`
- `incidentrelay_business_service_component`

## Supported Data Sources

- `incidentrelay_version`
- `incidentrelay_group`
- `incidentrelay_team`
- `incidentrelay_user`
- `incidentrelay_service`

Nested API objects such as channel `config`, route `matchers`, service
`labels`, maintenance `scopes`, and heartbeat `metadata` are represented as
validated JSON strings. This keeps the provider compatible with IncidentRelay's
matcher and integration DSL without forcing every nested field into Terraform.

## Local Installation

```sh
make install-local
```

This installs the provider under:

```text
~/.terraform.d/plugins/registry.terraform.io/incidentrelay/incidentrelay/0.1.0/<os>_<arch>/
```

## Example

See [examples/provider/example.tf](examples/provider/example.tf).

